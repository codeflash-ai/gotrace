package rewriter

import (
	"go/ast"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func ShouldInstrument(pkg *packages.Package, cfg *Config) bool {
	path := pkg.PkgPath

	if isStdlib(path) {
		return false
	}
	if strings.HasPrefix(path, "gotrace_tracer_runtime") {
		return false
	}

	if len(cfg.IncludePatterns) > 0 {
		for _, pattern := range cfg.IncludePatterns {
			if matchPattern(pattern, path) {
				return true
			}
		}
		return false
	}

	for _, pattern := range cfg.ExcludePatterns {
		if matchPattern(pattern, path) {
			return false
		}
	}

	// Only instrument packages within the same module
	if cfg.ModulePath != "" {
		return strings.HasPrefix(path, cfg.ModulePath)
	}

	return true
}

func ShouldSkipFile(file *ast.File, filename string) bool {
	if ast.IsGenerated(file) {
		return true
	}

	base := filepath.Base(filename)
	if strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, "_generated.go") ||
		strings.HasSuffix(base, "_gen.go") {
		return true
	}

	return false
}

func ShouldSkipFunc(fn *ast.FuncDecl) bool {
	return fn.Body == nil
}

func isStdlib(path string) bool {
	if path == "" {
		return false
	}
	return !strings.Contains(path, ".")
}

func matchPattern(pattern, path string) bool {
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(path, prefix)
	}
	if strings.HasSuffix(pattern, "/...") {
		prefix := strings.TrimSuffix(pattern, "/...")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	matched, _ := filepath.Match(pattern, path)
	return matched || pattern == path
}
