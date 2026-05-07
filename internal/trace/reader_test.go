package trace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/codeflash-ai/gotrace/pkg/tracer"
)

func TestReadTrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trace.bin")

	tracer.Init(path)

	fMain := tracer.RegisterFunc("main.main")
	fA := tracer.RegisterFunc("main.a")

	tokMain := tracer.Enter(fMain)
	time.Sleep(1 * time.Millisecond)

	tokA := tracer.Enter(fA)
	time.Sleep(2 * time.Millisecond)
	tracer.Exit(tokA)

	tracer.Exit(tokMain)
	tracer.Flush()

	frames, err := ReadTrace(path)
	if err != nil {
		t.Fatalf("ReadTrace: %v", err)
	}

	if len(frames) == 0 {
		t.Fatal("no frames returned")
	}

	root := frames[0]
	if root.Name != "main.main" {
		t.Errorf("root name = %q, want main.main", root.Name)
	}
	if len(root.Children) != 1 {
		t.Fatalf("root has %d children, want 1", len(root.Children))
	}
	if root.Children[0].Name != "main.a" {
		t.Errorf("child name = %q, want main.a", root.Children[0].Name)
	}
	if root.Duration < time.Millisecond {
		t.Errorf("root duration too short: %v", root.Duration)
	}
}

func TestBuildTree(t *testing.T) {
	names := []string{"main.main", "main.a", "main.b"}
	events := []tracer.Event{
		{Type: tracer.EventEnter, FuncID: 0, GoroutineID: 1, Timestamp: 0},
		{Type: tracer.EventEnter, FuncID: 1, GoroutineID: 1, Timestamp: 100},
		{Type: tracer.EventExit, FuncID: 1, GoroutineID: 1, Timestamp: 200},
		{Type: tracer.EventEnter, FuncID: 2, GoroutineID: 1, Timestamp: 300},
		{Type: tracer.EventExit, FuncID: 2, GoroutineID: 1, Timestamp: 400},
		{Type: tracer.EventExit, FuncID: 0, GoroutineID: 1, Timestamp: 500},
	}

	frames := BuildTree(events, names)
	if len(frames) != 1 {
		t.Fatalf("got %d roots, want 1", len(frames))
	}

	root := frames[0]
	if root.Name != "main.main" {
		t.Errorf("root = %q", root.Name)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root children = %d, want 2", len(root.Children))
	}
	if root.Children[0].Name != "main.a" {
		t.Errorf("child[0] = %q", root.Children[0].Name)
	}
	if root.Children[1].Name != "main.b" {
		t.Errorf("child[1] = %q", root.Children[1].Name)
	}
}

func TestBuildTreeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")

	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadTrace(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}
