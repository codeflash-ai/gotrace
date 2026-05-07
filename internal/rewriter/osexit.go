package rewriter

import "go/ast"

// rewriteOsExitCalls finds os.Exit() calls and prepends a tracer.Flush() before them.
func rewriteOsExitCalls(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}

		var newList []ast.Stmt
		modified := false
		for _, stmt := range block.List {
			if containsOsExit(stmt) {
				flushStmt := &ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("__gotrace_tracer"),
							Sel: ast.NewIdent("Flush"),
						},
					},
				}
				newList = append(newList, flushStmt)
				modified = true
			}
			newList = append(newList, stmt)
		}
		if modified {
			block.List = newList
		}
		return true
	})
}

func containsOsExit(stmt ast.Stmt) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "os" && sel.Sel.Name == "Exit" {
			found = true
			return false
		}
		return true
	})
	return found
}

// injectAtExit adds a Flush() call registered via atexit pattern at the end of main().
func injectAtExit(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Recv != nil {
			continue
		}

		// Also wrap any return statements in main with a flush
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			var newList []ast.Stmt
			modified := false
			for _, stmt := range block.List {
				if _, isReturn := stmt.(*ast.ReturnStmt); isReturn {
					flushStmt := &ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent("__gotrace_tracer"),
								Sel: ast.NewIdent("Flush"),
							},
						},
					}
					newList = append(newList, flushStmt)
					modified = true
				}
				newList = append(newList, stmt)
			}
			if modified {
				block.List = newList
			}
			return true
		})

		// Add Flush at the very end of main (for implicit return)
		flushEnd := &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("__gotrace_tracer"),
					Sel: ast.NewIdent("Flush"),
				},
			},
		}
		fn.Body.List = append(fn.Body.List, flushEnd)

		// Also handle os.Exit calls anywhere in the function
		rewriteOsExitCalls(fn.Body)
		break
	}
}

// rewriteOsExitCallsInFile rewrites os.Exit calls in all functions in a file.
func rewriteOsExitCallsInFile(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		rewriteOsExitCalls(fn.Body)
	}
}

// rewriteLogFatalCalls finds log.Fatal/log.Fatalf/log.Fatalln calls and prepends Flush.
func rewriteLogFatalCalls(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}

		var newList []ast.Stmt
		modified := false
		for _, stmt := range block.List {
			if containsLogFatal(stmt) {
				flushStmt := &ast.ExprStmt{
					X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent("__gotrace_tracer"),
							Sel: ast.NewIdent("Flush"),
						},
					},
				}
				newList = append(newList, flushStmt)
				modified = true
			}
			newList = append(newList, stmt)
		}
		if modified {
			block.List = newList
		}
		return true
	})
}

func containsLogFatal(stmt ast.Stmt) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == "log" && (sel.Sel.Name == "Fatal" || sel.Sel.Name == "Fatalf" || sel.Sel.Name == "Fatalln") {
			found = true
			return false
		}
		return true
	})
	return found
}

// addFlushBeforeExits rewrites os.Exit and log.Fatal calls in all functions to flush first.
func addFlushBeforeExits(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		rewriteOsExitCalls(fn.Body)
		rewriteLogFatalCalls(fn.Body)
	}

	// Also handle func literals
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.FuncLit)
		if !ok || lit.Body == nil {
			return true
		}
		rewriteOsExitCalls(lit.Body)
		rewriteLogFatalCalls(lit.Body)
		return true
	})
}

// injectRuntimeExitHook injects a runtime.SetExitHook call if available (Go 1.26+),
// or falls back to atexit-style Flush injection.
func injectRuntimeExitHook(file *ast.File) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "main" || fn.Recv != nil {
			continue
		}

		// Inject: runtime.SetExitHook(func(code int) { __gotrace_tracer.Flush() })
		hookStmt := &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("__gotrace_runtime"),
					Sel: ast.NewIdent("SetExitHook"),
				},
				Args: []ast.Expr{
					&ast.FuncLit{
						Type: &ast.FuncType{
							Params: &ast.FieldList{
								List: []*ast.Field{{
									Names: []*ast.Ident{ast.NewIdent("_")},
									Type:  ast.NewIdent("int"),
								}},
							},
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.ExprStmt{
									X: &ast.CallExpr{
										Fun: &ast.SelectorExpr{
											X:   ast.NewIdent("__gotrace_tracer"),
											Sel: ast.NewIdent("Flush"),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Insert after Init/Flush defer
		fn.Body.List = append([]ast.Stmt{hookStmt}, fn.Body.List...)
		break
	}
}
