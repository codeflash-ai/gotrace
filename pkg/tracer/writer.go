package tracer

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
)

const (
	traceMagic   = uint32(0x474F5452) // "GOTR"
	traceVersion = uint32(1)
)

func writeTrace(path string, names []string, evts []Event) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gotrace: failed to write trace: %v\n", err)
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	binary.Write(w, binary.LittleEndian, traceMagic)
	binary.Write(w, binary.LittleEndian, traceVersion)

	binary.Write(w, binary.LittleEndian, uint32(len(names)))
	for _, name := range names {
		binary.Write(w, binary.LittleEndian, uint16(len(name)))
		w.WriteString(name)
	}

	binary.Write(w, binary.LittleEndian, uint64(len(evts)))
	for i := range evts {
		binary.Write(w, binary.LittleEndian, evts[i].Type)
		binary.Write(w, binary.LittleEndian, evts[i].FuncID)
		binary.Write(w, binary.LittleEndian, evts[i].GoroutineID)
		binary.Write(w, binary.LittleEndian, evts[i].Timestamp)
		binary.Write(w, binary.LittleEndian, evts[i].ParentGID)
	}

	w.Flush()
}
