package rewriter

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/packages"
)

func instrumentFuncDecl(body *ast.BlockStmt, varName string) {
	enterStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("__gotrace_token")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{
			&ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X:   ast.NewIdent("__gotrace_tracer"),
					Sel: ast.NewIdent("Enter"),
				},
				Args: []ast.Expr{ast.NewIdent(varName)},
			},
		},
	}

	exitStmt := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent("__gotrace_tracer"),
				Sel: ast.NewIdent("Exit"),
			},
			Args: []ast.Expr{ast.NewIdent("__gotrace_token")},
		},
	}

	body.List = append([]ast.Stmt{enterStmt, exitStmt}, body.List...)
}

func resolveFuncName(pkg *packages.Package, decl *ast.FuncDecl) string {
	pkgPath := pkg.PkgPath
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		typeName := receiverTypeName(decl.Recv.List[0].Type)
		return fmt.Sprintf("%s.%s.%s", pkgPath, typeName, decl.Name.Name)
	}
	return fmt.Sprintf("%s.%s", pkgPath, decl.Name.Name)
}

func resolveAnonName(enclosing string, index int) string {
	return enclosing + ".func" + strconv.Itoa(index)
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	default:
		return "_"
	}
}

func makeVarName(name string) string {
	safe := ""
	for _, c := range name {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			safe += string(c)
		default:
			safe += "_"
		}
	}
	return "__gotrace_fid_" + safe
}
