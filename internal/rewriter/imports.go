package rewriter

import (
	"go/ast"
	"go/token"
	"strconv"
)

func injectImport(file *ast.File, tracerPath string) {
	importSpec := &ast.ImportSpec{
		Name: ast.NewIdent("__gotrace_tracer"),
		Path: &ast.BasicLit{
			Kind:  token.STRING,
			Value: strconv.Quote(tracerPath),
		},
	}

	found := false
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		gd.Specs = append(gd.Specs, importSpec)
		found = true
		break
	}

	if !found {
		importDecl := &ast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: 1,
			Specs:  []ast.Spec{importSpec},
		}
		file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
	}
}
