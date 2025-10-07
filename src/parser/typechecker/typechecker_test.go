package typechecker

import (
	"cmp"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func testHandler(t *testing.T) ddperror.Handler {
	return func(err ddperror.Error) {
		t.Errorf("%v", err)
	}
}

func NotNilMap[K comparable, V any](m map[K]V) map[K]V {
	if m != nil {
		return m
	}
	return make(map[K]V)
}

func NotNilSlice[T any](s []T) []T {
	if s != nil {
		return s
	}
	return make([]T, 0)
}

func createTypechecker(test *testing.T, overrider Typechecker) *Typechecker {
	// prevent nil-pointer exceptions
	overrider.Module = cmp.Or(overrider.Module, &ast.Module{})
	overrider.Module.Ast = cmp.Or(overrider.Module.Ast, &ast.Ast{})
	overrider.Module.Ast.Symbols = cmp.Or(overrider.Module.Ast.Symbols, ast.SymbolTable(&ast.BasicSymbolTable{}))

	symbols := &ast.BasicSymbolTable{
		EnclosingTable: overrider.Module.Ast.Symbols.Enclosing(),
		Declarations:   NotNilMap(overrider.Module.Ast.Symbols.(*ast.BasicSymbolTable).Declarations),
	}
	module := &ast.Module{
		FileName:             cmp.Or(overrider.Module.FileName, test.Name()),
		FileNameToken:        overrider.Module.FileNameToken,
		Imports:              NotNilSlice(overrider.Module.Imports),
		Comment:              overrider.Module.Comment,
		ExternalDependencies: NotNilMap(overrider.Module.ExternalDependencies),
		Ast: &ast.Ast{
			Statements: NotNilSlice(overrider.Module.Ast.Statements),
			Comments:   NotNilSlice(overrider.Module.Ast.Comments),
			Symbols:    symbols,
			Faulty:     overrider.Module.Ast.Faulty,
		},
		PublicDecls: NotNilMap(overrider.Module.PublicDecls),
		Operators:   NotNilMap(overrider.Module.Operators),
	}

	errorHandler := overrider.ErrorHandler
	if errorHandler == nil {
		errorHandler = testHandler(test)
	}
	panicMode := false
	return &Typechecker{
		ErrorHandler:       errorHandler,
		CurrentTable:       module.Ast.Symbols,
		Operators:          NotNilMap(overrider.Operators),
		Module:             module,
		latestReturnedType: ddptypes.VoidType{},
		panicMode:          &panicMode,
		instantiator:       nil,
	}
}

// takes name type pairs to create a symbol table
// a string followed by a ddptypes.Type creates a variable
// a string followed by nil creates a FuncDecl
func createSymbols(args ...any) ast.SymbolTable {
	symbols := ast.NewSymbolTable(nil)
	for i := 0; i < len(args); i += 2 {
		switch typ := args[i+1].(type) {
		case ddptypes.Type:
			symbols.InsertDecl(args[i].(string),
				&ast.VarDecl{
					NameTok:  token.Token{Type: token.IDENTIFIER, Literal: args[i].(string)},
					Type:     args[i+1].(ddptypes.Type),
					InitType: args[i+1].(ddptypes.Type),
				},
			)
		case *ast.FuncDecl:
			typ.NameTok = token.Token{Type: token.IDENTIFIER, Literal: args[i].(string)}
			symbols.InsertDecl(args[i].(string), typ)
		case *ast.StructDecl:
			typ.NameTok = token.Token{Type: token.IDENTIFIER, Literal: args[i].(string)}
			symbols.InsertDecl(args[i].(string), typ)
		}

		if args[i+1] == nil {
			symbols.InsertDecl(args[i].(string),
				&ast.FuncDecl{
					NameTok: token.Token{Type: token.IDENTIFIER, Literal: args[i].(string)},
				},
			)
			continue
		}
	}
	return symbols
}

func TestReferences(t *testing.T) {
	assert := assert.New(t)

	symbols := createSymbols("b", ddptypes.BUCHSTABE, "br", ddptypes.ReferenceType{Type: ddptypes.BUCHSTABE})

	ty := createTypechecker(t, Typechecker{CurrentTable: symbols, Module: &ast.Module{Ast: &ast.Ast{Symbols: symbols}}})

	// valid casts
	ty.TypecheckNode(&ast.CastExpr{Lhs: &ast.IntLit{}, TargetType: ddptypes.ZAHL})
	ty.TypecheckNode(&ast.CastExpr{Lhs: &ast.Ident{Literal: token.Token{Literal: "b"}}, TargetType: ddptypes.ZAHL})
	ty.TypecheckNode(&ast.CastExpr{Lhs: &ast.Ident{Literal: token.Token{Literal: "br"}}, TargetType: ddptypes.ZAHL})
	ty.TypecheckNode(&ast.CastExpr{Lhs: &ast.IntLit{}, TargetType: ddptypes.ReferenceType{Type: ddptypes.ZAHL}})

	ty.TypecheckNode(&ast.FuncCall{Args: map[string]ast.Expression{"b": &ast.Ident{Literal: token.Token{Literal: "b"}}}, Func: &ast.FuncDecl{
		Parameters: []ast.ParameterInfo{
			{Type: ddptypes.ReferenceType{Type: ddptypes.BUCHSTABE}, Name: token.Token{Literal: "b"}},
		},
	}})

	// TODO: more test cases

	assert.False(*ty.panicMode)
}
