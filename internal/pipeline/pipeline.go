package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/codeflash-ai/gotrace/internal/loader"
	"github.com/codeflash-ai/gotrace/internal/rewriter"
)

type Config struct {
	Dir             string
	Patterns        []string
	IncludePatterns []string
	ExcludePatterns []string
	BuildFlags      []string
	RunArgs         []string
	TestMode        bool
	TracerPkgDir    string
	Verbose         bool
}

type Result struct {
	TraceFilePath string
	ExitCode      int
	Stdout        []byte
	Stderr        []byte
}

func Run(ctx context.Context, cfg *Config) (*Result, error) {
	srcDir := cfg.Dir
	if srcDir == "" {
		var err error
		srcDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getwd: %w", err)
		}
	}

	absDir, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}

	ws, err := NewWorkspace(absDir)
	if err != nil {
		return nil, fmt.Errorf("workspace: %w", err)
	}
	defer ws.Cleanup()
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "gotrace: workspace at %s\n", ws.Dir())
	}

	loadCfg := &loader.Config{
		Dir:        absDir,
		Patterns:   cfg.Patterns,
		BuildFlags: cfg.BuildFlags,
	}
	loadResult, err := loader.Load(loadCfg)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}

	// Determine module path to filter out third-party packages
	modulePath := ""
	for _, pkg := range loadResult.Packages {
		if pkg.Module != nil {
			modulePath = pkg.Module.Path
			break
		}
	}

	rwCfg := &rewriter.Config{
		TracerImportPath: "gotrace_tracer_runtime",
		IncludePatterns:  cfg.IncludePatterns,
		ExcludePatterns:  cfg.ExcludePatterns,
		ModulePath:       modulePath,
	}
	rwResult, err := rewriter.Rewrite(loadResult.Packages, loadResult.Fset, rwCfg)
	if err != nil {
		return nil, fmt.Errorf("rewrite: %w", err)
	}

	if err := ws.WriteRewrittenFiles(rwResult); err != nil {
		return nil, fmt.Errorf("write rewritten: %w", err)
	}

	tracerDir := cfg.TracerPkgDir
	if tracerDir == "" {
		exe, err := os.Executable()
		if err == nil {
			tracerDir = filepath.Join(filepath.Dir(exe), "..", "pkg", "tracer")
		}
	}
	if tracerDir == "" {
		return nil, fmt.Errorf("tracer package directory not found")
	}

	if err := ws.InjectTracerPackage(tracerDir); err != nil {
		return nil, fmt.Errorf("inject tracer: %w", err)
	}

	if err := ws.UpdateGoMod(); err != nil {
		return nil, fmt.Errorf("update go.mod: %w", err)
	}

	patterns := cfg.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	traceFile, err := os.CreateTemp("", "gotrace-trace-*.bin")
	if err != nil {
		return nil, fmt.Errorf("create trace file: %w", err)
	}
	traceOutput := traceFile.Name()
	traceFile.Close()

	binary, err := Build(ctx, ws.Dir(), patterns, cfg.TestMode, cfg.BuildFlags)
	if err != nil {
		return nil, fmt.Errorf("build: %w", err)
	}

	execResult, err := Execute(ctx, binary, cfg.RunArgs, traceOutput)
	if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &Result{
		TraceFilePath: execResult.TraceFilePath,
		ExitCode:      execResult.ExitCode,
		Stdout:        execResult.Stdout,
		Stderr:        execResult.Stderr,
	}, nil
}
