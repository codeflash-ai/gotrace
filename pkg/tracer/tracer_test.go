package tracer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnterExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.bin")

	Init(path)

	fid := RegisterFunc("test.Hello")
	token := Enter(fid)
	Exit(token)

	Flush()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("trace file not written: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Size() == 0 {
		t.Fatal("trace file is empty")
	}
}

func TestRegisterFuncIdempotent(t *testing.T) {
	funcNameMu.Lock()
	funcNames = nil
	for k := range funcIndex {
		delete(funcIndex, k)
	}
	funcNameMu.Unlock()

	id1 := RegisterFunc("pkg.Foo")
	id2 := RegisterFunc("pkg.Foo")
	id3 := RegisterFunc("pkg.Bar")

	if id1 != id2 {
		t.Errorf("same name got different IDs: %d vs %d", id1, id2)
	}
	if id1 == id3 {
		t.Error("different names got same ID")
	}
}

func TestGoroutineID(t *testing.T) {
	gid := parseGIDFromStack()
	if gid == 0 {
		t.Error("parseGIDFromStack returned 0")
	}

	gid2 := parseGIDFromStack()
	if gid != gid2 {
		t.Errorf("same goroutine got different IDs: %d vs %d", gid, gid2)
	}
}
