package typechecker

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/stretchr/testify/assert"
)

func TestIsAssignable(t *testing.T) {
	assert := assert.New(t)

	ok := IsAssignable(&ast.Ident{})
	assert.True(ok)

	ok = IsAssignable(&ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{},
	})
	assert.True(ok)

	ok = IsAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{},
	}})
	assert.True(ok)

	ok = IsAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.CastExpr{Lhs: &ast.Ident{}},
	}})
	assert.True(ok)

	ok = IsAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.CastExpr{Lhs: &ast.IntLit{}},
	}})
	assert.False(ok)

	ok = IsAssignable(&ast.BinaryExpr{
		Lhs:          &ast.Ident{},
		Operator:     ast.BIN_FIELD_ACCESS,
		Rhs:          &ast.Ident{},
		OverloadedBy: &ast.OperatorOverload{Call: &ast.FuncCall{Func: &ast.FuncDecl{ReturnType: ddptypes.VoidType{}}}},
	})
	assert.False(ok)

	ok = IsAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.CastExpr{Lhs: &ast.Ident{}, OverloadedBy: &ast.OperatorOverload{Call: &ast.FuncCall{Func: &ast.FuncDecl{ReturnType: ddptypes.VoidType{}}}}},
	}})
	assert.False(ok)

	ok = IsAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.CastExpr{Lhs: &ast.Ident{}, OverloadedBy: &ast.OperatorOverload{Call: &ast.FuncCall{Func: &ast.FuncDecl{ReturnType: ddptypes.ReferenceType{Type: ddptypes.ZAHL}}}}},
	}})
	assert.True(ok)

	ok = IsAssignable(&ast.UnaryExpr{
		Operator: ast.UN_DEREF,
		Rhs:      &ast.FuncCall{Func: &ast.FuncDecl{ReturnType: ddptypes.ReferenceType{Type: ddptypes.ZAHL}}},
	})
	assert.True(ok)
}

func TestMarkAssignable(t *testing.T) {
	assert := assert.New(t)

	var expr ast.Expression = &ast.Ident{Declaration: &ast.VarDecl{Type: ddptypes.ZAHL}}
	markAssigneable(expr, ddptypes.ZAHL)
	assert.True(expr.HasMetadata(AssigneableMetaKind))

	expr = &ast.Ident{Declaration: &ast.VarDecl{Type: ddptypes.ReferenceType{Type: ddptypes.ZAHL}}}
	markAssigneable(expr, ddptypes.ZAHL)
	assert.False(expr.HasMetadata(AssigneableMetaKind))

	expr = &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{Declaration: &ast.VarDecl{Type: ddptypes.ReferenceType{Type: &ddptypes.StructType{}}}},
		Typ:      ddptypes.ReferenceType{Type: ddptypes.ZAHL},
	}
	markAssigneable(expr, ddptypes.ReferenceType{Type: ddptypes.ZAHL})
	assert.True(expr.HasMetadata(AssigneableMetaKind))
	assert.False(expr.(*ast.BinaryExpr).Rhs.HasMetadata(AssigneableMetaKind))

	expr = &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{},
		Typ:      ddptypes.ZAHL,
	}
	markAssigneable(expr, ddptypes.ReferenceType{Type: ddptypes.ZAHL})
	assert.True(expr.HasMetadata(AssigneableMetaKind))
	assert.True(expr.(*ast.BinaryExpr).Rhs.HasMetadata(AssigneableMetaKind))

	expr = &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs: &ast.BinaryExpr{
			Lhs:      &ast.Ident{},
			Operator: ast.BIN_FIELD_ACCESS,
			Rhs:      &ast.Ident{Declaration: &ast.VarDecl{Type: ddptypes.ReferenceType{Type: &ddptypes.StructType{}}}},
			Typ:      &ddptypes.StructType{},
		},
		Typ: ddptypes.ZAHL,
	}
	markAssigneable(expr, ddptypes.ZAHL)
	assert.True(expr.HasMetadata(AssigneableMetaKind))
	assert.True(expr.(*ast.BinaryExpr).Rhs.HasMetadata(AssigneableMetaKind))
	assert.False(expr.(*ast.BinaryExpr).Rhs.(*ast.BinaryExpr).Rhs.HasMetadata(AssigneableMetaKind))
}
