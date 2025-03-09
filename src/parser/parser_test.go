package parser

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/stretchr/testify/assert"
)

func TestInsertOperatorOverload(t *testing.T) {
	assert := assert.New(t)

	errorCollector := ddperror.Collector{}
	given := createParser(t, parser{
		errorHandler: errorCollector.GetHandler(),
	})

	op := func(a, b ddptypes.Type, aref, bref bool) *ast.FuncDecl {
		return &ast.FuncDecl{
			Operator: ast.BIN_PLUS,
			Parameters: []ast.ParameterInfo{
				{Type: ddptypes.ParameterType{Type: a, IsReference: aref}},
				{Type: ddptypes.ParameterType{Type: b, IsReference: bref}},
			},
		}
	}

	op_zahl_zahl := op(ddptypes.ZAHL, ddptypes.ZAHL, false, false)
	op_ref_zahl := op(ddptypes.ZAHL, ddptypes.ZAHL, true, false)
	op_ref_ref := op(ddptypes.ZAHL, ddptypes.ZAHL, true, true)

	given.insertOperatorOverload(op_zahl_zahl)
	given.insertOperatorOverload(op_ref_zahl)
	given.insertOperatorOverload(op_ref_ref)

	assert.Equal([]*ast.FuncDecl{op_ref_ref, op_ref_zahl, op_zahl_zahl}, given.Operators[ast.BIN_PLUS])
	assert.False(errorCollector.DidError())

	op_text_text := op(ddptypes.TEXT, ddptypes.TEXT, false, false)
	op_text_ref := op(ddptypes.TEXT, ddptypes.TEXT, false, true)

	given.insertOperatorOverload(op_text_text)
	given.insertOperatorOverload(op_text_ref)

	assert.Equal([]*ast.FuncDecl{op_ref_ref, op_text_ref, op_ref_zahl, op_text_text, op_zahl_zahl}, given.Operators[ast.BIN_PLUS])
	assert.False(errorCollector.DidError())

	op_generic_generic := op(&ddptypes.GenericType{Name: "T"}, &ddptypes.GenericType{Name: "R"}, false, false)

	given.insertOperatorOverload(op_generic_generic)

	assert.Equal([]*ast.FuncDecl{op_ref_ref, op_text_ref, op_ref_zahl, op_text_text, op_zahl_zahl, op_generic_generic}, given.Operators[ast.BIN_PLUS])
	assert.False(errorCollector.DidError())

	op_generic_ref := op(&ddptypes.GenericType{Name: "T"}, &ddptypes.GenericType{Name: "R"}, false, true)
	op_ref_ref_gen := op(&ddptypes.GenericType{Name: "T"}, &ddptypes.GenericType{Name: "R"}, true, true)

	given.insertOperatorOverload(op_generic_ref)
	given.insertOperatorOverload(op_ref_ref_gen)

	assert.Equal([]*ast.FuncDecl{op_ref_ref, op_text_ref, op_ref_zahl, op_text_text, op_zahl_zahl, op_ref_ref_gen, op_generic_ref, op_generic_generic}, given.Operators[ast.BIN_PLUS])
	assert.False(errorCollector.DidError())

	op_generic_generic2 := op(&ddptypes.GenericType{Name: "T"}, &ddptypes.GenericType{Name: "R"}, false, false)

	given.insertOperatorOverload(op_generic_generic2)

	assert.Equal([]*ast.FuncDecl{op_ref_ref, op_text_ref, op_ref_zahl, op_text_text, op_zahl_zahl, op_ref_ref_gen, op_generic_ref, op_generic_generic}, given.Operators[ast.BIN_PLUS])
	assert.True(errorCollector.DidError())
}
