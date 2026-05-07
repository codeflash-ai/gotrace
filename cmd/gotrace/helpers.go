package main

import (
	"os"
	"path/filepath"
	"runtime"
)

func splitArgs(args []string) (packages []string, runArgs []string) {
	if len(args) == 0 {
		return []string{"."}, nil
	}
	return []string{args[0]}, args[1:]
}

func findTracerPkg() string {
	// Check relative to executable
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "..", "pkg", "tracer")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Check relative to source (for development)
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Join(filepath.Dir(filename), "..", "..", "pkg", "tracer")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// Check working directory
	wd, err := os.Getwd()
	if err == nil {
		dir := filepath.Join(wd, "pkg", "tracer")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	return ""
}
