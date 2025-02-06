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
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
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
			Symbols:    symbols,
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
		module:                module,
		predefinedModules:     NotNilMap(overrider.predefinedModules),
		aliases:               cmp.Or(overrider.aliases, at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)),
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

func scanTokens(t *testing.T, src string) []token.Token {
	result, err := scanner.Scan(scanner.Options{
		FileName:     "test.ddp",
		ScannerMode:  scanner.ModeStrictCapitalization,
		ErrorHandler: testHandler(t),
		Source:       []byte(src),
	})
	if err != nil {
		t.Error("scanner error: " + err.Error())
	}
	return result
}

// takes name type pairs to create a symbol table
func createSymbols(args ...any) *ast.SymbolTable {
	symbols := ast.NewSymbolTable(nil)
	for i := 0; i < len(args); i += 2 {
		symbols.InsertDecl(args[i].(string),
			&ast.VarDecl{
				NameTok:  token.Token{Type: token.IDENTIFIER, Literal: args[i].(string)},
				Type:     ddptypes.ListType{Underlying: args[i+1].(ddptypes.Type)},
				InitType: args[i+1].(ddptypes.Type),
			})
	}
	return symbols
}

func success(assert *assert.Assertions, parser *parser, node ast.Node) bool {
	return cmp.Or(
		assert.NotNil(node),
		assert.False(parser.panicMode),
		assert.False(parser.errored),
	)
}

func TestTypeAliasDecl(t *testing.T) {
	assert := assert.New(t)
	given := createParser(t,
		parser{
			tokens: scanTokens(t,
				`Wir nennen eine Zahl auch eine Nummer.`,
			),
		},
	)
	given.cur = 1 // skip WIR

	raw_decl := given.typeAliasDecl()
	if !success(assert, given, raw_decl) {
		t.FailNow()
	}
	decl := raw_decl.(*ast.TypeAliasDecl)

	assert.Equal(given.tokens[0], decl.Tok)
	assert.Equal(given.tokens[6], decl.NameTok)
	assert.False(decl.IsPublic)
	assert.Equal(given.module, decl.Mod)
	assert.Equal(decl.Underlying, ddptypes.ZAHL)
	assert.Equal(decl.Type, &ddptypes.TypeAlias{Name: "Nummer", Underlying: ddptypes.ZAHL, GramGender: ddptypes.FEMININ})
}

func TestTypeAliasDeclError(t *testing.T) {
	assert := assert.New(t)
	panicMode := false
	structType := &ddptypes.StructType{
		Name: "Struktur",
	}
	given := typechecker.New(&ast.Module{
		Ast: &ast.Ast{
			Symbols: &ast.SymbolTable{
				Declarations: map[string]ast.Declaration{
					"Struktur": &ast.StructDecl{
						IsPublic: false,
						Type:     structType,
					},
				},
			},
		},
	}, func(err ddperror.Error) {
		assert.Equal(ddperror.SEM_BAD_PUBLIC_MODIFIER, err.Code)
	}, t.Name(), &panicMode)

	given.TypecheckNode(&ast.TypeAliasDecl{
		IsPublic:   true,
		Underlying: structType,
	})
	assert.True(panicMode)
}

func TestTypeAliasAliasInsert(t *testing.T) {
	assert := assert.New(t)
	eof := token.Token{Type: token.EOF}
	expectedType := &ddptypes.TypeAlias{Name: "Hausnummer", GramGender: ddptypes.FEMININ, Underlying: ddptypes.ZAHL}

	assert.True(ddptypes.ParamTypesEqual(ddptypes.ParameterType{
		Type: expectedType,
	}, ddptypes.ParameterType{
		Type: ddptypes.ZAHL,
	}))

	expectedAlias := &ast.FuncAlias{
		Original: token.Token{Literal: "Schreibe <p1>"},
		Func:     &ast.FuncDecl{},
	}

	aliasTrie := at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)
	aliasTrie.Insert([]*token.Token{
		{Type: token.IDENTIFIER, Literal: "Schreibe"},
		{Type: token.ALIAS_PARAMETER, Literal: "<p1>", AliasInfo: &ddptypes.ParameterType{
			Type: ddptypes.ZAHL,
		}},
	}, expectedAlias)

	exists, actualAlias := aliasTrie.Contains([]*token.Token{
		{Type: token.IDENTIFIER, Literal: "Schreibe"},
		{Type: token.ALIAS_PARAMETER, Literal: "<p1>", AliasInfo: &ddptypes.ParameterType{
			Type: ddptypes.ZAHL,
		}},
	})
	assert.True(exists)
	assert.Equal(expectedAlias, actualAlias)
	exists, actualAlias = aliasTrie.Contains([]*token.Token{
		{Type: token.IDENTIFIER, Literal: "Schreibe"},
		{Type: token.ALIAS_PARAMETER, Literal: "<hnummer>", AliasInfo: &ddptypes.ParameterType{
			Type: expectedType,
		}},
	})
	assert.True(exists)
	assert.NotNil(actualAlias)
	assert.Equal(expectedAlias, actualAlias)

	given := createParser(t, parser{
		aliases: aliasTrie,
	})

	testTokens := []token.Token{
		{Type: token.IDENTIFIER, Literal: "Schreibe"},
		{Type: token.ALIAS_PARAMETER, Literal: "<hnummer>", AliasInfo: &ddptypes.ParameterType{
			Type: expectedType,
		}},
		eof,
	}

	exists, isFunc, actualFuncAlias, pTokens := given.aliasExists(testTokens)
	assert.True(exists)
	assert.True(isFunc)
	assert.Equal(expectedAlias, actualFuncAlias)
	assert.Equal([]*token.Token{&testTokens[0], &testTokens[1]}, pTokens)
}

func TestFuncDeclProperties(t *testing.T) {
	assert := assert.New(t)

	runTest := func(tokens string, public, generic, shouldSucceed bool) {
		var errorHandler ddperror.Handler
		mockHandler := ddperror.MockHandler{}
		if !shouldSucceed {
			errorHandler = mockHandler.GetHandler()
		}
		given := createParser(t,
			parser{
				tokens:       scanTokens(t, tokens),
				errorHandler: errorHandler,
			},
		)

		decl_stmt := given.declaration()
		if !shouldSucceed {
			assert.True(mockHandler.DidError())
			return
		}
		if !success(assert, given, decl_stmt) {
			t.FailNow()
		}

		func_decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)
		assert.Equal(public, func_decl.IsPublic)
		assert.Equal(generic, ast.IsGeneric(func_decl))
	}
	runTest(`
Die Funktion foo gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo"`, false, false, true,
	)
	runTest(`
Die öffentliche Funktion foo gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo"`, true, false, true,
	)
	// illegal case
	runTest(`
Die generische Funktion foo gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo"`, false, true, false,
	)
	// illegal case
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ Zahl, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, false, true, false,
	)
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, false, true, true,
	)
	// illegal case
	runTest(`
Die öffentliche generische Funktion foo gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo"`, true, true, false,
	)
	runTest(`
Die öffentliche generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, true, true, true,
	)
	// illegal case
	runTest(`
Die generische öffentliche Funktion foo gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo"`, true, true, false,
	)
}

func TestGenericFuncDeclParameterTypes(t *testing.T) {
	assert := assert.New(t)

	runTest := func(tokens string, genericTypes []string, returnType string) {
		given := createParser(t,
			parser{
				tokens: scanTokens(t, tokens),
				errorHandler: func(err ddperror.Error) {
					if err.Code != ddperror.TYP_WRONG_RETURN_TYPE {
						t.Errorf("%v", err)
					}
				},
			},
		)

		decl_stmt := given.declaration()
		// if !success(assert, given, decl_stmt) {
		// 	t.FailNow()
		// }

		func_decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)
		assert.NotNil(func_decl.Generic)
		assert.ElementsMatch(genericTypes, maps.Keys(func_decl.Generic.Types))
		assert.Equal(returnType, func_decl.ReturnType.String())
	}
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, []string{"T"}, "nichts",
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt ein R zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T", "R"}, "R",
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und T, gibt eine Zahl zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "Zahl",
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und Zahl, gibt ein T zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T",
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Liste und Zahl, gibt ein T zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T",
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und Zahl, gibt eine T Liste zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T Liste",
	)
}

func TestGenericFuncDeclBodyTokens(t *testing.T) {
	assert := assert.New(t)

	runTest := func(src string, from, to int, shouldSucceed bool) {
		var errorHandler ddperror.Handler
		mockHandler := ddperror.MockHandler{}
		if !shouldSucceed {
			errorHandler = mockHandler.GetHandler()
		}
		tokens := scanTokens(t, src)
		given := createParser(t,
			parser{
				tokens:       tokens,
				errorHandler: errorHandler,
			},
		)

		decl_stmt := given.declaration()
		if !shouldSucceed {
			assert.True(mockHandler.DidError())
			return
		}
		if !success(assert, given, decl_stmt) {
			t.FailNow()
		}

		func_decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)
		assert.NotNil(func_decl.Generic)
		assert.Equal(tokens[from:to], func_decl.Generic.Tokens)
	}
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, 0, 0, true,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, 22, 26, true,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück,
ist in "libddpstdlib.a"
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, false,
	)
}
