package typechecker

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/stretchr/testify/assert"
)

func TestIsAssignable(t *testing.T) {
	assert := assert.New(t)

	_, ok := isAssignable(&ast.Ident{})
	assert.True(ok)

	_, ok = isAssignable(&ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{},
	})
	assert.True(ok)

	_, ok = isAssignable(&ast.Grouping{Expr: &ast.BinaryExpr{
		Lhs:      &ast.Ident{},
		Operator: ast.BIN_FIELD_ACCESS,
		Rhs:      &ast.Ident{},
	}})
	assert.True(ok)
}
