package typechecker

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
)

// casts the given expression to an assigneable
// returns the assignable and wether the cast was successful
func isAssignable(expr ast.Expression) (ast.Assigneable, bool) {
	if ass, ok := expr.(ast.Assigneable); ok {
		return ass, true
	}

	switch ass := expr.(type) {
	case *ast.Grouping:
		return isAssignable(ass.Expr)
	case *ast.BinaryExpr:
		return isBinaryExprAssignable(ass)
	default:
		return nil, false
	}
}

func isBinaryExprAssignable(expr *ast.BinaryExpr) (ast.Assigneable, bool) {
	switch expr.Operator {
	case ast.BIN_FIELD_ACCESS:
		ident, isIdent := expr.Lhs.(*ast.Ident)
		if !isIdent {
			break
		}

		ass, ok := isAssignable(expr.Rhs)
		if ok {
			return &ast.FieldAccess{
				Field: ident,
				Rhs:   ass,
			}, true
		}
	case ast.BIN_INDEX:
		ass, ok := isAssignable(expr.Lhs)
		if ok {
			return &ast.Indexing{
				Lhs:   ass,
				Index: expr.Rhs,
			}, true
		}
	default:
		return nil, false
	}
	return nil, false
}
