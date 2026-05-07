package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagInclude []string
	flagExclude []string
	flagVerbose bool
)

var rootCmd = &cobra.Command{
	Use:   "gotrace",
	Short: "AST-based Go function tracer",
	Long:  "GoTrace instruments Go source code to trace every function call with exact timings.",
}

func init() {
	rootCmd.PersistentFlags().StringSliceVar(&flagInclude, "include", nil, "Package patterns to instrument (e.g., myapp/*)")
	rootCmd.PersistentFlags().StringSliceVar(&flagExclude, "exclude", nil, "Package patterns to skip")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose output")
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
