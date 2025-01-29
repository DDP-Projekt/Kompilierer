package ast

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func TestIsExternFunc(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsExternFunc(&FuncDecl{
		Body: nil,
		Def:  nil,
	}))
	assert.False(IsExternFunc(&FuncDecl{
		Body: &BlockStmt{},
		Def:  nil,
	}))
	assert.False(IsExternFunc(&FuncDecl{
		Body: nil,
		Def:  &FuncDef{},
	}))
	assert.False(IsExternFunc(&FuncDecl{
		Body: &BlockStmt{},
		Def:  &FuncDef{},
	}))
}

func TestIsForwardDecl(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsForwardDecl(&FuncDecl{
		Body: nil,
		Def:  &FuncDef{},
	}))
	assert.False(IsForwardDecl(&FuncDecl{
		Body: nil,
		Def:  nil,
	}))
	assert.False(IsForwardDecl(&FuncDecl{
		Body: &BlockStmt{},
		Def:  nil,
	}))
}

func TestIsOperatorOverload(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsOperatorOverload(&FuncDecl{
		Operator: BIN_OR,
	}))
	assert.False(IsOperatorOverload(&FuncDecl{
		Operator: nil,
	}))
}

func TestTrimStringLit(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("", TrimStringLit(nil))
	assert.Equal("test", TrimStringLit(&token.Token{Literal: `"test"`}))
	assert.Equal(`"test"`, TrimStringLit(&token.Token{Literal: `""test""`}))
	assert.Equal("", TrimStringLit(&token.Token{Literal: `""`}))
}

func TestIsGlobalScope(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsGlobalScope(&SymbolTable{Enclosing: nil}))
	assert.False(IsGlobalScope(&SymbolTable{Enclosing: &SymbolTable{}}))
}
