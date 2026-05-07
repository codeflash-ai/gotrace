package rewriter

import "go/ast"

func rewriteGoStmts(body *ast.BlockStmt) {
	for i, stmt := range body.List {
		if goStmt, ok := stmt.(*ast.GoStmt); ok {
			body.List[i] = rewriteGoStmt(goStmt)
		}
	}
}

func rewriteGoStmt(stmt *ast.GoStmt) *ast.GoStmt {
	wrapperBody := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.ExprStmt{X: stmt.Call},
		},
	}
	wrapperFunc := &ast.FuncLit{
		Type: &ast.FuncType{Params: &ast.FieldList{}},
		Body: wrapperBody,
	}

	stmt.Call = &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   ast.NewIdent("__gotrace_tracer"),
			Sel: ast.NewIdent("Go"),
		},
		Args: []ast.Expr{wrapperFunc},
	}
	return stmt
}

func rewriteGoStmtsDeep(node ast.Node) {
	ast.Inspect(node, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		rewriteGoStmts(block)
		return true
	})
}
