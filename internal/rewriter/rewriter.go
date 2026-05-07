package rewriter

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/packages"
)

type Config struct {
	TracerImportPath string
	IncludePatterns  []string
	ExcludePatterns  []string
	ModulePath       string
}

type Result struct {
	Files []*RewrittenFile
}

type RewrittenFile struct {
	Fset     *token.FileSet
	File     *ast.File
	Original string
}

func Rewrite(pkgs []*packages.Package, fset *token.FileSet, cfg *Config) (*Result, error) {
	result := &Result{}

	visited := make(map[string]bool)
	for _, pkg := range pkgs {
		if err := rewriteRecursive(pkg, fset, cfg, result, visited); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func rewriteRecursive(pkg *packages.Package, fset *token.FileSet, cfg *Config, result *Result, visited map[string]bool) error {
	if visited[pkg.PkgPath] {
		return nil
	}
	visited[pkg.PkgPath] = true

	// Walk imports first (depth-first) to instrument dependencies
	for _, imp := range pkg.Imports {
		if ShouldInstrument(imp, cfg) {
			if err := rewriteRecursive(imp, fset, cfg, result, visited); err != nil {
				return err
			}
		}
	}

	if !ShouldInstrument(pkg, cfg) {
		return nil
	}
	return rewritePackage(pkg, fset, cfg, result)
}

func rewritePackage(pkg *packages.Package, fset *token.FileSet, cfg *Config, result *Result) error {
	for i, file := range pkg.Syntax {
		var filename string
		if i < len(pkg.CompiledGoFiles) {
			filename = pkg.CompiledGoFiles[i]
		} else if i < len(pkg.GoFiles) {
			filename = pkg.GoFiles[i]
		} else {
			tokFile := fset.File(file.Pos())
			if tokFile != nil {
				filename = tokFile.Name()
			}
		}
		if filename == "" {
			continue
		}

		if ShouldSkipFile(file, filename) {
			continue
		}

		rewritten := rewriteFile(pkg, file, cfg)
		if rewritten {
			result.Files = append(result.Files, &RewrittenFile{
				Fset:     fset,
				File:     file,
				Original: filename,
			})
		}
	}
	return nil
}

// func printAST(file *ast.File) {
// 	var buf bytes.Buffer

// 	err := format.Node(&buf, token.NewFileSet(), file)
// 	if err != nil {
// 		panic(err)
// 	}

// 	fmt.Println("========================================")
// 	fmt.Println(buf.String())
// 	fmt.Println("========================================")
// }

func rewriteFile(pkg *packages.Package, file *ast.File, cfg *Config) bool {
	var funcVars []funcRegistration
	anonCounters := make(map[string]int)
	modified := false

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || ShouldSkipFunc(fn) {
			continue
		}

		funcName := resolveFuncName(pkg, fn)
		varName := makeVarName(funcName)

		funcVars = append(funcVars, funcRegistration{
			varName:  varName,
			funcName: funcName,
		})

		instrumentFuncDecl(fn.Body, varName)
		rewriteGoStmtsDeep(fn)

		instrumentAnonFuncs(fn.Body, funcName, anonCounters, &funcVars)

		// printAST(file)
		modified = true
	}

	if !modified {
		return false
	}

	injectImport(file, cfg.TracerImportPath)
	injectRegistrations(file, funcVars)
	addFlushBeforeExits(file)

	if pkg.Name == "main" {
		injectMainInit(file)
	}

	return true
}

type funcRegistration struct {
	varName  string
	funcName string
}

func instrumentAnonFuncs(node ast.Node, enclosing string, counters map[string]int, regs *[]funcRegistration) {
	ast.Inspect(node, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok {
			return true
		}
		if lit.Body == nil || len(lit.Body.List) == 0 {
			return true
		}

		counters[enclosing]++
		anonName := resolveAnonName(enclosing, counters[enclosing])
		varName := makeVarName(anonName)

		*regs = append(*regs, funcRegistration{
			varName:  varName,
			funcName: anonName,
		})

		instrumentFuncDecl(lit.Body, varName)
		return true
	})
}

func injectRegistrations(file *ast.File, regs []funcRegistration) {
	if len(regs) == 0 {
		return
	}

	specs := make([]ast.Spec, len(regs))
	for i, reg := range regs {
		specs[i] = &ast.ValueSpec{
			Names: []*ast.Ident{ast.NewIdent(reg.varName)},
			Values: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent("__gotrace_tracer"),
						Sel: ast.NewIdent("RegisterFunc"),
					},
					Args: []ast.Expr{
						&ast.BasicLit{
							Kind:  token.STRING,
							Value: `"` + reg.funcName + `"`,
						},
					},
				},
			},
		}
	}

	genDecl := &ast.GenDecl{
		Tok:    token.VAR,
		Lparen: 1,
		Specs:  specs,
	}

	file.Decls = append(file.Decls, genDecl)
}

func injectMainInit(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Recv != nil {
			continue
		}

		// Find the Enter/Exit stmts we just injected (first 2 stmts)
		// Insert Init before them and defer Flush after
		initCall := &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("__gotrace_tracer"),
					Sel: ast.NewIdent("Init"),
				},
				Args: []ast.Expr{
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("__gotrace_os"),
							Sel: ast.NewIdent("Getenv"),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: `"GOTRACE_OUTPUT"`,
							},
						},
					},
				},
			},
		}

		flushDefer := &ast.DeferStmt{
			Call: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("__gotrace_tracer"),
					Sel: ast.NewIdent("Flush"),
				},
			},
		}

		fn.Body.List = append([]ast.Stmt{initCall, flushDefer}, fn.Body.List...)

		injectOsImport(file)
		break
	}
}

func injectOsImport(file *ast.File) {
	importSpec := &ast.ImportSpec{
		Name: ast.NewIdent("__gotrace_os"),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: `"os"`,
		},
	}

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		gd.Specs = append(gd.Specs, importSpec)
		return
	}
}
