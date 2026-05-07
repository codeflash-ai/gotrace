package loader

import (
	"fmt"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type Config struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
	Env        []string
}

type Result struct {
	Fset     *token.FileSet
	Packages []*packages.Package
}

func Load(cfg *Config) (*Result, error) {
	fset := token.NewFileSet()

	loadCfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedSyntax |
			packages.NeedTypes |
			packages.NeedTypesInfo |
			packages.NeedModule |
			packages.NeedImports |
			packages.NeedDeps,
		Dir:        cfg.Dir,
		Fset:       fset,
		BuildFlags: cfg.BuildFlags,
		Env:        cfg.Env,
	}

	patterns := cfg.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	pkgs, err := packages.Load(loadCfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}

	var errs []error
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, e)
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("package errors: %v", errs)
	}

	return &Result{Fset: fset, Packages: pkgs}, nil
}
