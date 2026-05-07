package main

import (
	"os"

	"github.com/codeflash-ai/gotrace/internal/output"
	"github.com/codeflash-ai/gotrace/internal/pipeline"
	"github.com/codeflash-ai/gotrace/internal/trace"
	"github.com/spf13/cobra"
)

var jsonCmd = &cobra.Command{
	Use:   "json [packages] [-- args...]",
	Short: "Build and run with tracing, output JSON trace",
	RunE:  runJSON,
}

func init() {
	rootCmd.AddCommand(jsonCmd)
}

func runJSON(cmd *cobra.Command, args []string) error {
	packages, runArgs := splitArgs(args)

	cfg := &pipeline.Config{
		Dir:             ".",
		Patterns:        packages,
		IncludePatterns: flagInclude,
		ExcludePatterns: flagExclude,
		RunArgs:         runArgs,
		TracerPkgDir:    findTracerPkg(),
		Verbose:         flagVerbose,
	}

	result, err := pipeline.Run(cmd.Context(), cfg)
	if err != nil {
		return err
	}

	frames, err := trace.ReadTrace(result.TraceFilePath)
	os.Remove(result.TraceFilePath)
	if err != nil {
		return err
	}

	return output.RenderJSON(os.Stdout, frames)
}
