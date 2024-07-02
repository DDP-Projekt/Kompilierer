package parser

import (
	"reflect"
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

func createParser(t *testing.T, tokens []token.Token) *parser {
	return &parser{
		tokens:       tokens,
		comments:     []token.Token{},
		errorHandler: testHandler(t),
		module: &ast.Module{
			FileName: t.Name(),
		},
		typeNames: make(map[string]ddptypes.Type),
	}
}

func createParser2(test *testing.T, overrider parser) *parser {
	errHndl := testHandler(test)
	symbols := &ast.SymbolTable{
		Declarations: make(map[string]ast.Declaration),
	}
	module := &ast.Module{
		FileName: test.Name(),
		Ast: &ast.Ast{
			Statements: make([]ast.Statement, 0),
			Symbols:    symbols,
		},
		PublicDecls: make(map[string]ast.Declaration),
	}

	parser := parser{
		tokens:       []token.Token{{Type: token.EOF}},
		comments:     []token.Token{},
		cur:          0,
		errorHandler: errHndl,
		lastError:    ddperror.Error{},
		module:       module,
		aliases:      at.New[*token.Token, ast.Alias](tokenEqual, tokenLess),
		typeNames:    make(map[string]ddptypes.Type),
	}
	parser.resolver, _ = resolver.New(module, errHndl, module.FileName, &parser.panicMode)
	parser.typechecker, _ = typechecker.New(module, errHndl, module.FileName, &parser.panicMode)

	overrider_value := reflect.ValueOf(&overrider)
	overrider_type := overrider_value.Type()

	parser_value := reflect.ValueOf(&parser)

	fields := reflect.VisibleFields(overrider_type)
	for _, field := range fields {
		value := overrider_value.FieldByName(field.Name)
		if value.IsZero() {
			continue
		}
		switch value.Kind() {
		case reflect.Struct:
		default:
			parser_value.Elem().FieldByName(field.Name).Set(value)
		}
	}

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
		createTokens(
			token.WIR, token.NENNEN, token.EINE, token.ZAHL, token.AUCH, token.EINE,
			token.IDENTIFIER, "Nummer",
			token.DOT,
		),
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
