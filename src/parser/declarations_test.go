package parser

import (
	"cmp"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/parser/resolver"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func testHandler(t *testing.T) ddperror.Handler {
	return func(err ddperror.Error) {
		t.Error(err.Error())
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

// returns a parser filled with good default values
// overridden by the ones passed in
func createParser(test *testing.T, overrider parser) *parser {
	// prevent nil-pointer exceptions
	overrider.module = cmp.Or(overrider.module, &ast.Module{})
	overrider.module.Ast = cmp.Or(overrider.module.Ast, &ast.Ast{})
	overrider.module.Ast.Symbols = cmp.Or(overrider.module.Ast.Symbols, &ast.SymbolTable{})

	symbols := &ast.SymbolTable{
		Enclosing:    overrider.module.Ast.Symbols.Enclosing,
		Declarations: NotNilMap(overrider.module.Ast.Symbols.Declarations),
	}
	module := &ast.Module{
		FileName:             cmp.Or(overrider.module.FileName, test.Name()),
		FileNameToken:        overrider.module.FileNameToken,
		Imports:              NotNilSlice(overrider.module.Imports),
		Comment:              overrider.module.Comment,
		ExternalDependencies: NotNilMap(overrider.module.ExternalDependencies),
		Ast: &ast.Ast{
			Statements: NotNilSlice(overrider.module.Ast.Statements),
			Comments:   NotNilSlice(overrider.module.Ast.Comments),
			Symbols:    cmp.Or(overrider.module.Ast.Symbols, symbols),
			Faulty:     overrider.module.Ast.Faulty,
		},
		PublicDecls: NotNilMap(overrider.module.PublicDecls),
	}

	errorHandler := overrider.errorHandler
	if errorHandler == nil {
		errorHandler = testHandler(test)
	}

	parser := parser{
		tokens:                NotNilSlice(overrider.tokens),
		comments:              NotNilSlice(overrider.comments),
		cur:                   overrider.cur,
		errorHandler:          errorHandler,
		lastError:             overrider.lastError,
		module:                cmp.Or(overrider.module, module),
		predefinedModules:     NotNilMap(overrider.predefinedModules),
		aliases:               cmp.Or(overrider.aliases, at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)),
		typeNames:             NotNilMap(overrider.typeNames),
		currentFunction:       overrider.currentFunction,
		isCurrentFunctionBool: overrider.isCurrentFunctionBool,
		panicMode:             overrider.panicMode,
		errored:               overrider.errored,
		resolver:              overrider.resolver,
		typechecker:           overrider.typechecker,
	}
	parser.resolver = cmp.Or(parser.resolver, resolver.New(parser.module, errorHandler, parser.module.FileName, &parser.panicMode))
	parser.typechecker = cmp.Or(parser.typechecker, typechecker.New(parser.module, errorHandler, parser.module.FileName, &parser.panicMode))

	return &parser
}

func createTokens(args ...any) (result []token.Token) {
	range_index := uint(0)
	for i, arg := range args {
		t, ok := arg.(token.TokenType)
		if !ok {
			continue
		}

		tok := token.Token{
			Type:    t,
			Literal: t.String(),
			Indent:  0,
		}

		switch t {
		case token.IDENTIFIER:
			tok.Literal = args[i+1].(string)
		default:
		}

		tok.Range.Start = token.Position{Line: 1, Column: range_index}
		range_index += uint(1 + len(tok.Literal) + 1) // <space> + literal + <space>
		tok.Range.End = token.Position{Line: 1, Column: range_index - 1}

		result = append(result, tok)
	}
	return result
}

func TestTypeAliasDecl(t *testing.T) {
	assert := assert.New(t)
	given := createParser(t,
		parser{
			tokens: createTokens(
				token.WIR, token.NENNEN, token.EINE, token.ZAHL, token.AUCH, token.EINE,
				token.IDENTIFIER, "Nummer",
				token.DOT,
			),
		},
	)
	given.cur = 1 // skip WIR

	raw_decl := given.typeAliasDecl()
	assert.NotNil(raw_decl)
	decl := raw_decl.(*ast.TypeAliasDecl)

	assert.Equal(given.tokens[0], decl.Tok)
	assert.Equal(given.tokens[6], decl.NameTok)
	assert.False(decl.IsPublic)
	assert.Equal(given.module, decl.Mod)
	assert.Equal(decl.Underlying, ddptypes.ZAHL)
	assert.Equal(decl.Type, &ddptypes.TypeAlias{Name: "Nummer", Underlying: ddptypes.ZAHL, GramGender: ddptypes.FEMININ})
}
