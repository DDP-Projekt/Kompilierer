package parser

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/stretchr/testify/assert"
)

func TestForRangeWithoutIndex(t *testing.T) {
	assert := assert.New(t)

	symbols := createSymbols("l", ddptypes.ListType{Underlying: ddptypes.ZAHL})

	given := createParser(t, parser{
		module: &ast.Module{
			Ast: &ast.Ast{
				Symbols: symbols,
			},
		},
		tokens: scanTokens(t,
			`Für jede Zahl z in l, mache:
				Speichere 1 in z.`),
	})
	given.cur = 1 // skip FÜR

	raw_stmt := given.forStatement()
	if !success(assert, given, raw_stmt) {
		t.FailNow()
	}
	assert.IsType(&ast.ForRangeStmt{}, raw_stmt)
	for_stmt := raw_stmt.(*ast.ForRangeStmt)

	assert.Nil(for_stmt.Index)
	assert.Equal(for_stmt.Initializer.Name(), "z")
	assert.Equal(for_stmt.Initializer.Type, ddptypes.ZAHL)
}

func TestForRangeWithIndex(t *testing.T) {
	assert := assert.New(t)

	symbols := createSymbols("l", ddptypes.ListType{Underlying: ddptypes.ZAHL})

	given := createParser(t, parser{
		module: &ast.Module{
			Ast: &ast.Ast{
				Symbols: symbols,
			},
		},
		tokens: scanTokens(t,
			`Für jede Zahl z mit Index i in l, mache:
				Speichere z in i.`),
	})
	given.cur = 1 // skip FÜR

	raw_stmt := given.forStatement()
	if !success(assert, given, raw_stmt) {
		t.FailNow()
	}
	assert.IsType(&ast.ForRangeStmt{}, raw_stmt)
	for_stmt := raw_stmt.(*ast.ForRangeStmt)

	assert.Equal("i", for_stmt.Index.Name())
	assert.Equal(for_stmt.Initializer.Name(), "z")
	assert.Equal(for_stmt.Initializer.Type, ddptypes.ZAHL)
}
