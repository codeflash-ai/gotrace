package tracer

import (
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	funcNames  []string
	funcNameMu sync.Mutex
	funcIndex  = make(map[string]uint32)

	events   []Event
	eventIdx atomic.Uint64

	startTime  int64
	outputPath string
	initOnce   sync.Once
)

const defaultMaxEvents = 16 * 1024 * 1024

func Init(path string) {
	initOnce.Do(func() {
		outputPath = path
		events = make([]Event, defaultMaxEvents)
		startTime = time.Now().UnixNano()

		mainGID := gidCounter.Add(1)
		rGID := parseGIDFromStack()
		gidSlot.Store(rGID, mainGID)

		// Flush on signals and atexit since os.Exit skips defers
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-c
			Flush()
			os.Exit(1)
		}()
	})
}

func RegisterFunc(name string) uint32 {
	funcNameMu.Lock()
	defer funcNameMu.Unlock()

	if id, ok := funcIndex[name]; ok {
		return id
	}
	id := uint32(len(funcNames))
	funcNames = append(funcNames, name)
	funcIndex[name] = id
	return id
}

func Enter(funcID uint32) uint64 {
	if events == nil {
		return ^uint64(0)
	}
	idx := eventIdx.Add(1) - 1
	if idx >= uint64(len(events)) {
		return ^uint64(0)
	}
	gid := getGoroutineID()
	events[idx] = Event{
		Type:        EventEnter,
		FuncID:      funcID,
		GoroutineID: gid,
		Timestamp:   time.Now().UnixNano() - startTime,
	}
	return idx
}

func Exit(token uint64) {
	if token == ^uint64(0) {
		return
	}
	idx := eventIdx.Add(1) - 1
	if idx >= uint64(len(events)) {
		return
	}
	gid := getGoroutineID()
	events[idx] = Event{
		Type:        EventExit,
		FuncID:      events[token].FuncID,
		GoroutineID: gid,
		Timestamp:   time.Now().UnixNano() - startTime,
	}
}

func Go(fn func()) {
	parentGID := getGoroutineID()
	childGID := gidCounter.Add(1)

	go func() {
		rGID := parseGIDFromStack()
		gidSlot.Store(rGID, childGID)

		idx := eventIdx.Add(1) - 1
		if idx < uint64(len(events)) {
			events[idx] = Event{
				Type:        EventSpawn,
				GoroutineID: childGID,
				ParentGID:   parentGID,
				Timestamp:   time.Now().UnixNano() - startTime,
			}
		}

		fn()
	}()
}

func Flush() {
	if outputPath == "" {
		return
	}
	count := eventIdx.Load()
	if count > uint64(len(events)) {
		count = uint64(len(events))
	}
	writeTrace(outputPath, funcNames, events[:count])
}

func GetFuncNames() []string {
	return funcNames
}

func GetEvents() []Event {
	count := eventIdx.Load()
	if count > uint64(len(events)) {
		count = uint64(len(events))
	}
	return events[:count]
}
