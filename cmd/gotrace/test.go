package main

import (
	"fmt"
	"os"

	"github.com/codeflash-ai/gotrace/internal/output"
	"github.com/codeflash-ai/gotrace/internal/pipeline"
	"github.com/codeflash-ai/gotrace/internal/trace"
	"github.com/spf13/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [packages] [-- test flags...]",
	Short: "Build and run tests with tracing",
	RunE:  runTest,
}

func init() {
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	packages, testArgs := splitArgs(args)

	cfg := &pipeline.Config{
		Dir:             ".",
		Patterns:        packages,
		IncludePatterns: flagInclude,
		ExcludePatterns: flagExclude,
		RunArgs:         testArgs,
		TestMode:        true,
		TracerPkgDir:    findTracerPkg(),
		Verbose:         flagVerbose,
	}

	result, err := pipeline.Run(cmd.Context(), cfg)
	if err != nil {
		return err
	}

	if len(result.Stdout) > 0 {
		os.Stdout.Write(result.Stdout)
	}
	if len(result.Stderr) > 0 {
		os.Stderr.Write(result.Stderr)
	}

	frames, err := trace.ReadTrace(result.TraceFilePath)
	os.Remove(result.TraceFilePath)
	if err != nil {
		return fmt.Errorf("read trace: %w", err)
	}

	fmt.Fprintln(os.Stdout)
	return output.RenderTree(os.Stdout, frames)
}
