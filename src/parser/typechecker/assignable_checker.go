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
	var checker assignableCaster
	ast.VisitNode(&checker, expr, nil)
	return checker.ass, checker.ass != nil
}

type assignableCaster struct {
	ass ast.Assigneable
}

var (
	_ ast.Visitor            = (*assignableCaster)(nil)
	_ ast.IdentVisitor       = (*assignableCaster)(nil)
	_ ast.IndexingVisitor    = (*assignableCaster)(nil)
	_ ast.FieldAccessVisitor = (*assignableCaster)(nil)
	_ ast.BinaryExprVisitor  = (*assignableCaster)(nil)
)

func (*assignableCaster) Visitor() {}

func (a *assignableCaster) VisitIdent(i *ast.Ident) ast.VisitResult {
	a.ass = i
	return ast.VisitRecurse
}

func (a *assignableCaster) VisitIndexing(i *ast.Indexing) ast.VisitResult {
	a.ass = i
	return ast.VisitSkipChildren
}

func (a *assignableCaster) VisitFieldAccess(f *ast.FieldAccess) ast.VisitResult {
	a.ass = f
	return ast.VisitSkipChildren
}

func (a *assignableCaster) VisitBinaryExpr(expr *ast.BinaryExpr) ast.VisitResult {
	switch expr.Operator {
	case ast.BIN_FIELD_ACCESS:
		ident, isIdent := expr.Lhs.(*ast.Ident)
		if !isIdent {
			a.ass = nil
			break
		}

		ast.VisitNode(a, expr.Rhs, nil)
		if a.ass != nil {
			a.ass = &ast.FieldAccess{
				Field: ident,
				Rhs:   a.ass,
			}
		}
	case ast.BIN_INDEX:
		ast.VisitNode(a, expr.Lhs, nil)
		if a.ass != nil {
			a.ass = &ast.Indexing{
				Lhs:   a.ass,
				Index: expr.Rhs,
			}
		}
	default:
		a.ass = nil
	}
	return ast.VisitSkipChildren
}
