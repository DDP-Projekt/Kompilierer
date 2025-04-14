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
	overrider.module.Ast.Symbols = cmp.Or(overrider.module.Ast.Symbols, ast.SymbolTable(&ast.BasicSymbolTable{}))

	symbols := &ast.BasicSymbolTable{
		EnclosingTable: overrider.module.Ast.Symbols.Enclosing(),
		Declarations:   NotNilMap(overrider.module.Ast.Symbols.(*ast.BasicSymbolTable).Declarations),
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
		Operators:   NotNilMap(overrider.module.Operators),
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
		Operators:             NotNilMap(overrider.Operators),
	}
	parser.resolver = cmp.Or(parser.resolver, resolver.New(parser.module, parser.Operators, errorHandler, &parser.panicMode))
	parser.typechecker = cmp.Or(parser.typechecker, typechecker.New(parser.module, parser.Operators, errorHandler, &parser, &parser.panicMode))

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
			Symbols: &ast.BasicSymbolTable{
				Declarations: map[string]ast.Declaration{
					"Struktur": &ast.StructDecl{
						IsPublic: false,
						Type:     structType,
					},
				},
			},
		},
	}, nil, func(err ddperror.Error) {
		assert.Equal(ddperror.SEM_BAD_PUBLIC_MODIFIER, err.Code)
	}, nil, &panicMode)

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
		mockHandler := ddperror.Collector{}
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

	runTest := func(tokens string, genericTypes []string, returnType string, parameterTypes []string) {
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
		assert.ElementsMatch(parameterTypes, mapSlice(func_decl.Parameters, func(p ast.ParameterInfo) string { return p.Type.Type.String() }))
		assert.Equal(returnType, func_decl.ReturnType.String())
	}
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, []string{"T"}, "nichts", []string{"T"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt ein R zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T", "R"}, "R", []string{"T", "R"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und T, gibt eine Zahl zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "Zahl", []string{"T", "T"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und Zahl, gibt ein T zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T", []string{"T", "Zahl"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Liste und Zahl, gibt ein T zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T", []string{"T Liste", "Zahl"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und Zahl, gibt eine T Liste zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T Liste", []string{"T", "Zahl"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Liste und Zahl, gibt eine T Liste zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T Liste", []string{"T Liste", "Zahl"},
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Listen Referenz und Zahl, gibt eine T Liste zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, []string{"T"}, "T Liste", []string{"T Liste", "Zahl"},
	)
}

func TestGenericFuncDeclBodyTokens(t *testing.T) {
	assert := assert.New(t)

	runTest := func(src string, from, to int, shouldSucceed bool, code ddperror.Code) {
		var errorHandler ddperror.Handler
		mockHandler := ddperror.Collector{}
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
			assert.Contains(mapSlice(mockHandler.Errors, func(err ddperror.Error) ddperror.Code {
				return err.Code
			}), code)
			return
		}
		if !success(assert, given, decl_stmt) {
			t.FailNow()
		}

		func_decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)
		assert.NotNil(func_decl.Generic)
		if !ast.IsExternFunc(func_decl) {
			assert.Equal(tokens[from:to], func_decl.Generic.Tokens)
		} else {
			assert.Nil(func_decl.Generic.Tokens)
		}
	}
	runTest(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
	"foo <a>"`, 17, 18, true, 0,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück, macht:
	Gib 1 zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`, 21, 26, true, 0,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Referenz und R Referenz, gibt eine Zahl zurück,
ist in "libddpstdlib.a" definiert
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, true, 0,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Referenz und R Referenz, gibt eine T Liste zurück,
ist in "libddpstdlib.a" definiert
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, true, 0,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück,
ist in "libddpstdlib.a" definiert
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, false, ddperror.TYP_GENERIC_EXTERN_FUNCTION_BAD_PARAM_OR_RETURN,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T Referenz und R Referenz, gibt ein T zurück,
ist in "libddpstdlib.a" definiert
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, false, ddperror.TYP_GENERIC_EXTERN_FUNCTION_BAD_PARAM_OR_RETURN,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück,
wird später definiert
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, false, ddperror.SEM_GENERIC_FUNCTION_BODY_UNDEFINED,
	)
	runTest(`
Die generische Funktion foo mit den Parametern a und b vom Typ T und R, gibt nichts zurück, ist extern sichtbar, macht:
Und kann so benutzt werden:
	"foo <a> <b>"`, 0, 0, false, ddperror.SEM_GENERIC_FUNCTION_EXTERN_VISIBLE,
	)
}

func TestGenericFuncDeclContext(t *testing.T) {
	assert := assert.New(t)

	parseAndAssertConext := func(src string, symbols ast.SymbolTable, aliases [][]*token.Token, decls ...string) (*ast.FuncDecl, *parser) {
		trie := at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)
		for _, alias := range aliases {
			trie.Insert(alias, &ast.FuncAlias{})
		}

		given := createParser(t,
			parser{
				tokens:  scanTokens(t, src),
				aliases: trie,
				module:  &ast.Module{Ast: &ast.Ast{Symbols: symbols}},
			},
		)

		decl_stmt := given.declaration()
		if !success(assert, given, decl_stmt) {
			t.FailNow()
		}

		func_decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)
		assert.NotNil(func_decl.Generic)
		assert.NotNil(func_decl.Generic.Context)
		for _, decl := range decls {
			_, exists, _ := func_decl.Generic.Context.Symbols.LookupDecl(decl)
			assert.True(exists)
		}
		for _, alias := range aliases {
			exists, _ := func_decl.Generic.Context.Aliases.Contains(alias)
			assert.True(exists)
		}
		return func_decl, given
	}

	symbols := createSymbols(
		"i", ddptypes.ZAHL,
		"bar", &ast.FuncDecl{},
	)

	decl, given := parseAndAssertConext(`
Die generische Funktion foo mit dem Parameter a vom Typ T, gibt nichts zurück, macht:
Und kann so benutzt werden:
"foo <a>"`, symbols, [][]*token.Token{
		toPointerSlice(scanTokens(t, `bar`)),
		toPointerSlice(scanTokens(t, `baz`)),
		toPointerSlice(scanTokens(t, `test`)),
	}, "i", "foo",
	)
	_, exists, _ := decl.Generic.Context.Symbols.LookupDecl("test")
	assert.False(exists)
	symbols.InsertDecl("test", &ast.VarDecl{})
	_, exists, _ = decl.Generic.Context.Symbols.LookupDecl("test")
	assert.True(exists)

	foo_alias := scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: decl.Generic.Types["T"], IsReference: false},
	})

	exists, alias := given.aliases.Contains(foo_alias.GetKey())
	assert.True(exists)
	assert.Equal(decl, alias.Decl())
}

func TestGenericStructDecl(t *testing.T) {
	assert := assert.New(t)
	_ = assert

	given := createParser(t, parser{
		tokens: scanTokens(t, `
Wir nennen die generische Kombination aus
	dem T x,
	dem T y,
	dem R z,
	der R Liste l,
eine Struktur, und erstellen sie so:
	"Struktur"
		`),
	})

	decl_stmt := given.declaration()
	decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.StructDecl)

	assert.True(ast.IsGeneric(decl))
	assert.IsType(&ddptypes.GenericStructType{}, decl.Type)
	assert.Equal([]ddptypes.GenericType{{Name: "T"}, {Name: "R"}}, decl.Type.(*ddptypes.GenericStructType).GenericTypes)
}

func TestValidateStructAlias(t *testing.T) {
	assert := assert.New(t)

	given := createParser(t, parser{})

	alias := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL},
		"b": {Type: ddptypes.ZAHL},
	})
	err, _ := given.validateStructAlias(alias.GetTokens(), []*ast.VarDecl{
		{NameTok: token.Token{Literal: "a"}, Type: ddptypes.ZAHL},
		{NameTok: token.Token{Literal: "b"}, Type: ddptypes.ZAHL},
	})
	assert.Nil(err)

	alias = scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL},
		"b": {Type: ddptypes.ZAHL},
	})
	err, _ = given.validateStructAlias(alias.GetTokens(), []*ast.VarDecl{
		{NameTok: token.Token{Literal: "a"}, Type: ddptypes.ZAHL},
		{NameTok: token.Token{Literal: "b"}, Type: ddptypes.ZAHL},
	})
	assert.Nil(err)

	alias = scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.GenericType{Name: "T"}},
		"b": {Type: ddptypes.GenericType{Name: "R"}},
	})
	err, _ = given.validateStructAlias(alias.GetTokens(), []*ast.VarDecl{
		{NameTok: token.Token{Literal: "a"}, Type: ddptypes.GenericType{Name: "T"}},
		{NameTok: token.Token{Literal: "b"}, Type: ddptypes.GenericType{Name: "R"}},
	})
	assert.Nil(err)

	alias = scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.GenericType{Name: "T"}},
		"b": {Type: ddptypes.GenericType{Name: "R"}},
	})
	err, _ = given.validateStructAlias(alias.GetTokens(), []*ast.VarDecl{
		{NameTok: token.Token{Literal: "a"}, Type: ddptypes.GenericType{Name: "T"}},
		{NameTok: token.Token{Literal: "b"}, Type: ddptypes.GenericType{Name: "R"}},
	})
	assert.NotNil(err)
	assert.Equal(ddperror.SEM_UNABLE_TO_UNIFY_FIELD_TYPES, err.Code)
}
