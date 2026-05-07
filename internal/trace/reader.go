package trace

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/codeflash-ai/gotrace/pkg/tracer"
)

func ReadTrace(path string) ([]*tracer.Frame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	var magic, version uint32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("read magic: %w", err)
	}
	if magic != 0x474F5452 {
		return nil, fmt.Errorf("invalid trace file (bad magic: 0x%X)", magic)
	}
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	if version != 1 {
		return nil, fmt.Errorf("unsupported trace version: %d", version)
	}

	var nameCount uint32
	if err := binary.Read(r, binary.LittleEndian, &nameCount); err != nil {
		return nil, fmt.Errorf("read name count: %w", err)
	}

	names := make([]string, nameCount)
	for i := uint32(0); i < nameCount; i++ {
		var length uint16
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return nil, fmt.Errorf("read name length: %w", err)
		}
		buf := make([]byte, length)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("read name: %w", err)
		}
		names[i] = string(buf)
	}

	var eventCount uint64
	if err := binary.Read(r, binary.LittleEndian, &eventCount); err != nil {
		return nil, fmt.Errorf("read event count: %w", err)
	}

	events := make([]tracer.Event, eventCount)
	for i := uint64(0); i < eventCount; i++ {
		if err := binary.Read(r, binary.LittleEndian, &events[i].Type); err != nil {
			return nil, fmt.Errorf("read event %d: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &events[i].FuncID); err != nil {
			return nil, fmt.Errorf("read event %d: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &events[i].GoroutineID); err != nil {
			return nil, fmt.Errorf("read event %d: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &events[i].Timestamp); err != nil {
			return nil, fmt.Errorf("read event %d: %w", i, err)
		}
		if err := binary.Read(r, binary.LittleEndian, &events[i].ParentGID); err != nil {
			return nil, fmt.Errorf("read event %d: %w", i, err)
		}
	}

	return BuildTree(events, names), nil
}

func BuildTree(events []tracer.Event, funcNames []string) []*tracer.Frame {
	stacks := make(map[uint64][]*tracer.Frame)
	parentMap := make(map[uint64]uint64) // child GID -> parent GID
	var roots []*tracer.Frame

	for _, ev := range events {
		switch ev.Type {
		case tracer.EventSpawn:
			parentMap[ev.GoroutineID] = ev.ParentGID

		case tracer.EventEnter:
			name := ""
			if int(ev.FuncID) < len(funcNames) {
				name = funcNames[ev.FuncID]
			}
			frame := &tracer.Frame{
				Name:        name,
				GoroutineID: ev.GoroutineID,
				Start:       time.Duration(ev.Timestamp),
			}
			stack := stacks[ev.GoroutineID]
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, frame)
			} else {
				roots = append(roots, frame)
			}
			stacks[ev.GoroutineID] = append(stack, frame)

		case tracer.EventExit:
			stack := stacks[ev.GoroutineID]
			if len(stack) > 0 {
				frame := stack[len(stack)-1]
				frame.End = time.Duration(ev.Timestamp)
				frame.Duration = frame.End - frame.Start
				stacks[ev.GoroutineID] = stack[:len(stack)-1]
			}
		}
	}

	// Link goroutine roots to their parent goroutine's current frame
	linked := make([]*tracer.Frame, 0, len(roots))
	for _, root := range roots {
		if parentGID, ok := parentMap[root.GoroutineID]; ok {
			stack := stacks[parentGID]
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, root)
				continue
			}
		}
		linked = append(linked, root)
	}

	if len(linked) > 0 {
		return linked
	}
	return roots
}
