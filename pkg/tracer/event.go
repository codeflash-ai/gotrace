package tracer

import "time"

type EventType uint8

const (
	EventEnter EventType = 1
	EventExit  EventType = 2
	EventSpawn EventType = 3
)

type Event struct {
	Type        EventType
	FuncID      uint32
	GoroutineID uint64
	Timestamp   int64
	ParentGID   uint64
}

type Frame struct {
	Name        string
	Start       time.Duration
	End         time.Duration
	Duration    time.Duration
	GoroutineID uint64
	ParentGID   uint64
	Children    []*Frame
}
