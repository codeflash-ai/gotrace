package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Build(ctx context.Context, workDir string, patterns []string, testMode bool, flags []string) (string, error) {
	outputBinary := filepath.Join(workDir, "gotrace_binary")

	var args []string
	if testMode {
		args = append([]string{"test", "-c", "-o", outputBinary}, flags...)
	} else {
		args = append([]string{"build", "-o", outputBinary}, flags...)
	}
	args = append(args, patterns...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w\n%s", err, stderr.String())
	}
	return outputBinary, nil
}

type ExecResult struct {
	TraceFilePath string
	ExitCode      int
	Stdout        []byte
	Stderr        []byte
}

func Execute(ctx context.Context, binaryPath string, args []string, traceOutput string) (*ExecResult, error) {
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Env = append(os.Environ(), "GOTRACE_OUTPUT="+traceOutput)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return nil, fmt.Errorf("execute: %w", err)
	}

	return &ExecResult{
		TraceFilePath: traceOutput,
		ExitCode:      exitCode,
		Stdout:        stdout.Bytes(),
		Stderr:        stderr.Bytes(),
	}, nil
}
