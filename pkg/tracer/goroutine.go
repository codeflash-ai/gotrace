package tracer

import (
	"runtime"
	"sync"
	"sync/atomic"
)

var (
	gidCounter atomic.Uint64
	gidSlot    sync.Map // runtime goroutine ID -> assigned GID
)

func getGoroutineID() uint64 {
	rGID := parseGIDFromStack()
	if gid, ok := gidSlot.Load(rGID); ok {
		return gid.(uint64)
	}
	newGID := gidCounter.Add(1)
	actual, loaded := gidSlot.LoadOrStore(rGID, newGID)
	if loaded {
		return actual.(uint64)
	}
	return newGID
}

func parseGIDFromStack() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	// Format: "goroutine 123 [running]:\n..."
	s := buf[:n]
	// Skip "goroutine "
	i := 0
	for i < len(s) && s[i] != ' ' {
		i++
	}
	i++ // skip space
	var id uint64
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		id = id*10 + uint64(s[i]-'0')
		i++
	}
	return id
}
