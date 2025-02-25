package parser

import (
	"strings"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func scanAlias(t *testing.T, alias string, params map[string]ddptypes.ParameterType) ast.Alias {
	orig := token.Token{Literal: "\"" + alias + "\""}
	result, err := scanner.ScanAlias(orig, testHandler(t))
	if err != nil {
		t.Errorf("error scanning alias: %v", err)
	}
	for i := range result {
		if result[i].Type == token.ALIAS_PARAMETER {
			info := params[strings.Trim(result[i].Literal, "<>")]
			result[i].AliasInfo = &info
		}
	}
	return &ast.FuncAlias{
		Tokens:   result[:len(result)-1], // remove EOF
		Original: orig,
		Args:     params,
	}
}

func TestAliasSorter(t *testing.T) {
	assert := assert.New(t)

	a := scanAlias(t, `foo`, nil)
	b := scanAlias(t, `foo <a> test`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
	})
	c := scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
	})
	d := scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: true},
	})
	e := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: ddptypes.ZAHL, IsReference: true},
	})
	f := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: true},
		"b": {Type: ddptypes.ZAHL, IsReference: true},
	})
	g := scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
	})
	h := scanAlias(t, `foo <a>`, map[string]ddptypes.ParameterType{
		"a": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: true},
	})
	i := scanAlias(t, `foo <a> <b> sehr viel länger`, map[string]ddptypes.ParameterType{
		"a": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
		"b": {Type: ddptypes.ZAHL, IsReference: false},
	})
	j := scanAlias(t, `foo <a> <b> sehr viel länger`, map[string]ddptypes.ParameterType{
		"a": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
		"b": {Type: &ddptypes.GenericType{Name: "R"}, IsReference: false},
	})
	k := scanAlias(t, `foo <a> <b> sehr viel länger`, map[string]ddptypes.ParameterType{
		"a": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
		"b": {Type: &ddptypes.GenericType{Name: "R"}, IsReference: true},
	})

	// longer aliase to the top
	aliases := []ast.Alias{a, b, c}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{b, c, a}, aliases)

	// references up
	aliases = []ast.Alias{c, d}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{d, c}, aliases)

	// more references even higher
	aliases = []ast.Alias{e, f}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{f, e}, aliases)

	// longer ones above references
	aliases = []ast.Alias{a, b, c, d, e, f}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{f, e, b, d, c, a}, aliases)

	// generics to the bottom
	aliases = []ast.Alias{g, c}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{c, g}, aliases)

	// generics to the bottom, but then refs
	aliases = []ast.Alias{g, h}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{h, g}, aliases)

	// more generics deeper down
	aliases = []ast.Alias{j, i}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{i, j}, aliases)

	// same number of generics but different number of refs
	aliases = []ast.Alias{j, k}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{k, j}, aliases)

	// everything
	aliases = []ast.Alias{a, b, c, d, e, f, g, h, i, j, k}
	sortAliases(aliases)
	assert.Equal([]ast.Alias{i, k, j, f, e, b, d, c, h, g, a}, aliases)
}

func TestGenerateGenericContext(t *testing.T) {
	assert := assert.New(t)
	_ = assert

	parserAliases := at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)
	foo_a := scanAlias(t, `ein alias foo`, nil)
	parserAliases.Insert(toPointerSlice(foo_a.GetTokens()), foo_a)
	baz_a_parser := scanAlias(t, `ein alias baz`, nil)
	parserAliases.Insert(toPointerSlice(baz_a_parser.GetTokens()), baz_a_parser)

	given := createParser(t, parser{
		aliases: parserAliases,
	})

	baz_context := &ast.FuncDecl{}
	baz_parser := &ast.FuncDecl{}

	given.setScope(createSymbols(
		"a", ddptypes.KOMMAZAHL,
		"z", ddptypes.ZAHL,
		"foo", &ast.FuncDecl{},
		"baz", baz_parser,
		"Foo", &ast.StructDecl{
			Type: &ddptypes.StructType{Name: "Foo"},
		},
	))

	declContextAliases := at.New[*token.Token, ast.Alias](tokenEqual, tokenLess)
	bar_a := scanAlias(t, `ein alias bar`, nil)
	declContextAliases.Insert(toPointerSlice(bar_a.GetTokens()), bar_a)
	baz_a_declContext := scanAlias(t, `ein alias baz`, nil)
	declContextAliases.Insert(toPointerSlice(baz_a_declContext.GetTokens()), baz_a_declContext)

	structDecl := &ast.StructDecl{
		Type: &ddptypes.StructType{Name: "Bar"},
	}

	declContext := ast.GenericContext{
		Symbols: createSymbols(
			"a", ddptypes.KOMMAZAHL,
			"z1", ddptypes.ZAHL,
			"bar", &ast.FuncDecl{},
			"baz", baz_context,
			"Bar", structDecl,
		),
		Aliases: declContextAliases,
	}

	context := given.generateGenericContext(declContext, []ast.ParameterInfo{
		{Name: token.Token{Literal: "a"}, Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false}},
		{Name: token.Token{Literal: "b"}, Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false}},
		{Name: token.Token{Literal: "c"}, Type: ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "T"}, IsReference: false}},
		{Name: token.Token{Literal: "d"}, Type: ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "R"}, IsReference: false}},
		{Name: token.Token{Literal: "e"}, Type: ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: &ddptypes.GenericType{Name: "T"}}, IsReference: false}},
	}, map[string]ddptypes.Type{
		"T": ddptypes.ZAHL,
		"R": structDecl.Type,
	})

	assert.NotNil(context.Symbols)
	assert.NotNil(context.Aliases)

	a_decl, has_a, _ := context.Symbols.LookupDecl("a")
	a_decl_given, _, _ := given.scope().LookupDecl("a")
	a_decl_declContext, _, _ := declContext.Symbols.LookupDecl("a")
	_, has_z, _ := context.Symbols.LookupDecl("z")
	_, has_foo, _ := context.Symbols.LookupDecl("foo")
	_, has_z1, _ := context.Symbols.LookupDecl("z1")
	_, has_bar, _ := context.Symbols.LookupDecl("bar")
	_, has_Foo, _ := context.Symbols.LookupDecl("Foo")
	_, has_Bar, _ := context.Symbols.LookupDecl("Bar")
	_, has_T, _ := context.Symbols.LookupDecl("T")
	RDecl, has_R, _ := context.Symbols.LookupDecl("R")

	// variables from the parser should not be used
	assert.False(has_z)
	// everything else from both tables should be present
	assert.True(has_foo)
	assert.True(has_z1)
	assert.True(has_bar)
	assert.True(has_Foo)
	assert.True(has_Bar)
	// generic types should not have a decl themselves
	assert.False(has_T)
	assert.True(has_R)
	assert.Same(RDecl, structDecl)
	// parameters should override everything
	assert.True(has_a)
	assert.NotSame(a_decl.(*ast.VarDecl), a_decl_given.(*ast.VarDecl))
	assert.NotSame(a_decl.(*ast.VarDecl), a_decl_declContext.(*ast.VarDecl))

	// the original context should be prefered when the parser and context both contain a name
	bazDecl, has_baz, _ := context.Symbols.LookupDecl("baz")
	assert.True(has_baz)
	assert.Same(baz_context, bazDecl)
	assert.NotSame(baz_parser, bazDecl)

	// types as well
	FooType, has_Foo_type := context.Symbols.LookupType("Foo")
	BarType, has_Bar_type := context.Symbols.LookupType("Bar")
	TType, has_T_type := context.Symbols.LookupType("T")
	RType, has_R_type := context.Symbols.LookupType("R")
	assert.True(has_Foo_type)
	assert.True(has_Bar_type)
	assert.True(has_T_type)
	assert.True(has_R_type)
	assert.Equal("Foo", FooType.String())
	assert.Equal("Bar", BarType.String())
	assert.True(ddptypes.Equal(ddptypes.ZAHL, TType))
	assert.True(ddptypes.Equal(structDecl.Type, RType))
	// parameter types should be correctly instantiated
	c_decl_context, _, _ := context.Symbols.LookupDecl("c")
	assert.True(ddptypes.Equal(ddptypes.ZAHL, c_decl_context.(*ast.VarDecl).Type))
	assert.True(ddptypes.Equal(ddptypes.ListType{Underlying: ddptypes.ZAHL}, ddptypes.ListType{Underlying: TType}))
	e_decl_context, e_exists, e_isVar := context.Symbols.LookupDecl("e")
	assert.True(e_exists)
	assert.True(e_isVar)
	assert.True(ddptypes.Equal(ddptypes.ListType{Underlying: ddptypes.ZAHL}, e_decl_context.(*ast.VarDecl).Type))

	assert.Equal("Zahl", TType.String())
	assert.Equal("Bar", RType.String())

	assert.True(ddptypes.MatchesGender(TType, ddptypes.MASKULIN))
	assert.True(ddptypes.MatchesGender(TType, ddptypes.FEMININ))
	assert.True(ddptypes.MatchesGender(TType, ddptypes.NEUTRUM))
	assert.True(ddptypes.MatchesGender(RType, ddptypes.MASKULIN))
	assert.True(ddptypes.MatchesGender(RType, ddptypes.FEMININ))
	assert.True(ddptypes.MatchesGender(RType, ddptypes.NEUTRUM))

	has_foo_a, _ := context.Aliases.Contains(toPointerSlice(foo_a.GetTokens()))
	has_bar_a, _ := context.Aliases.Contains(toPointerSlice(bar_a.GetTokens()))
	has_baz_a, context_baz_a := context.Aliases.Contains(toPointerSlice(baz_a_parser.GetTokens()))
	assert.True(has_foo_a)
	assert.True(has_bar_a)
	assert.True(has_baz_a)
	assert.Same(baz_a_declContext.(*ast.FuncAlias), context_baz_a.(*ast.FuncAlias))

	newAlias := scanAlias(t, `neuer Alias`, nil)
	context.Aliases.Insert(toPointerSlice(newAlias.GetTokens()), newAlias)

	has_new_alias, _ := given.aliases.Contains(toPointerSlice(newAlias.GetTokens()))
	assert.False(has_new_alias)
	has_new_alias, _ = declContext.Aliases.Contains(toPointerSlice(newAlias.GetTokens()))
	assert.False(has_new_alias)
	has_new_alias, _ = context.Aliases.Contains(toPointerSlice(newAlias.GetTokens()))
	assert.True(has_new_alias)

	// inserting into the context should not change the parser or declContext tables
	assert.False(context.Symbols.InsertDecl("new", &ast.FuncDecl{}))
	_, existed, _ := declContext.Symbols.LookupDecl("new")
	assert.False(existed)
	_, existed, _ = given.scope().LookupDecl("new")
	assert.False(existed)
	_, existed, _ = context.Symbols.LookupDecl("new")
	assert.True(existed)
}

func TestInstantiateGenericFunction(t *testing.T) {
	assert := assert.New(t)

	given := createParser(t, parser{
		tokens: scanTokens(t, `
Die generische Funktion foo mit den Parametern a und b vom Typ T und T, gibt ein T zurück, macht:
	Gib a plus b zurück.
Und kann so benutzt werden:
	"foo <a> <b>"`),
	})

	decl_stmt := given.declaration()
	decl := decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)

	instantiation, errors := given.instantiateGenericFunction(decl, map[string]ddptypes.Type{
		"T": ddptypes.ZAHL,
	}, ddptypes.ZAHL)

	assert.Empty(errors)
	assert.NotNil(instantiation)
	assert.Contains(decl.Generic.Instantiations, given.module)
	assert.Contains(decl.Generic.Instantiations[given.module], instantiation)

	instantiation, errors = given.instantiateGenericFunction(decl, map[string]ddptypes.Type{
		"T": ddptypes.KOMMAZAHL,
	}, ddptypes.KOMMAZAHL)

	assert.Empty(errors)
	assert.NotNil(instantiation)
	assert.Contains(decl.Generic.Instantiations, given.module)
	assert.Contains(decl.Generic.Instantiations[given.module], instantiation)

	second_instantiation, errors := given.instantiateGenericFunction(decl, map[string]ddptypes.Type{
		"T": ddptypes.KOMMAZAHL,
	}, ddptypes.KOMMAZAHL)

	assert.Empty(errors)
	assert.NotNil(second_instantiation)
	assert.Contains(decl.Generic.Instantiations, given.module)
	assert.Contains(decl.Generic.Instantiations[given.module], second_instantiation)
	assert.Same(instantiation, second_instantiation)

	_, errors = given.instantiateGenericFunction(decl, map[string]ddptypes.Type{
		"T": ddptypes.BUCHSTABE,
	}, ddptypes.BUCHSTABE)

	assert.Equal(ddperror.TYP_TYPE_MISMATCH, errors[0].Code)
	assert.Equal(2, len(decl.Generic.Instantiations[given.module]))

	given = createParser(t, parser{
		tokens: scanTokens(t, `
Die generische Funktion foo mit den Parametern a und b vom Typ T Referenz und T Referenz, gibt nichts zurück, macht:
	Das T temp ist a.
	Speichere b in a.
	Speichere temp in b.
Und kann so benutzt werden:
	"foo <a> <b>"`),
	})

	decl_stmt = given.declaration()
	decl = decl_stmt.(*ast.DeclStmt).Decl.(*ast.FuncDecl)

	instantiation, errors = given.instantiateGenericFunction(decl, map[string]ddptypes.Type{
		"T": ddptypes.ZAHL,
	}, ddptypes.VoidType{})

	assert.Empty(errors)
	assert.NotNil(instantiation)
	assert.Contains(decl.Generic.Instantiations, given.module)
	assert.Contains(decl.Generic.Instantiations[given.module], instantiation)
}

func TestUnifyGenericType(t *testing.T) {
	assert := assert.New(t)

	typ := unifyGenericType(ddptypes.ZAHL, ddptypes.ParameterType{Type: ddptypes.ZAHL}, nil)
	assert.Equal(ddptypes.ZAHL, typ)

	genericTypes := map[string]ddptypes.Type{}
	typ = unifyGenericType(ddptypes.ZAHL, ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ddptypes.ZAHL, typ)
	assert.Equal(map[string]ddptypes.Type{"T": ddptypes.ZAHL}, genericTypes)

	genericTypes = map[string]ddptypes.Type{"T": ddptypes.ZAHL}
	typ = unifyGenericType(ddptypes.ZAHL, ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ddptypes.ZAHL, typ)
	assert.Equal(map[string]ddptypes.Type{"T": ddptypes.ZAHL}, genericTypes)

	genericTypes = map[string]ddptypes.Type{"T": ddptypes.TEXT}
	typ = unifyGenericType(ddptypes.ZAHL, ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ddptypes.TEXT, typ)
	assert.Equal(map[string]ddptypes.Type{"T": ddptypes.TEXT}, genericTypes)

	// with lists

	genericTypes = map[string]ddptypes.Type{}
	typ = unifyGenericType(ddptypes.ListType{Underlying: ddptypes.ZAHL}, ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: &ddptypes.GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(ddptypes.ListType{Underlying: ddptypes.ZAHL}, typ)
	assert.Equal(map[string]ddptypes.Type{"T": ddptypes.ZAHL}, genericTypes)

	genericTypes = map[string]ddptypes.Type{}
	typ = unifyGenericType(ddptypes.ListType{Underlying: ddptypes.ZAHL}, ddptypes.ParameterType{Type: &ddptypes.GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ddptypes.ListType{Underlying: ddptypes.ZAHL}, typ)
	assert.Equal(map[string]ddptypes.Type{"T": ddptypes.ListType{Underlying: ddptypes.ZAHL}}, genericTypes)

	genericTypes = map[string]ddptypes.Type{}
	typ = unifyGenericType(ddptypes.ZAHL, ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: &ddptypes.GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(nil, typ)
	assert.NotContains(genericTypes, "T")
}

func TestCheckAlias(t *testing.T) {
	assert := assert.New(t)

	given := createParser(t, parser{
		tokens: scanTokens(t, `foo 1 2`),
	})

	f := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: ddptypes.ZAHL, IsReference: false},
	})
	cached_args := make(map[cachedArgKey]*cachedArg, 4)
	args, instantiation, errs := given.checkAlias(f, true, 0, cached_args)
	assert.Nil(instantiation)
	assert.Empty(errs)
	assert.IsType(&ast.IntLit{}, args["a"])
	assert.IsType(&ast.IntLit{}, args["b"])

	// generic test

	genericFunc := &ast.FuncDecl{
		NameTok:    token.Token{Literal: "foo"},
		Mod:        given.module,
		ReturnType: ddptypes.VoidType{},
		Parameters: []ast.ParameterInfo{
			{
				Name: token.Token{Literal: "a"},
				Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false},
			},
		},
		Generic: &ast.GenericFuncInfo{
			Types: map[string]*ddptypes.GenericType{"T": {Name: "T"}},
			Tokens: scanTokens(t, `:
Ende`),
			Context:        ast.GenericContext{Symbols: given.scope(), Aliases: given.aliases},
			Instantiations: make(map[*ast.Module][]*ast.FuncDecl),
		},
	}
	_ = genericFunc

	g := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
	})
	g.(*ast.FuncAlias).Func = genericFunc

	cached_args = make(map[cachedArgKey]*cachedArg, 4)
	args, instantiation, errs = given.checkAlias(g, true, 0, cached_args)
	assert.Empty(errs)
	assert.NotEmpty(args)
	assert.NotNil(instantiation)
	assert.IsType(&ast.IntLit{}, args["a"])
	assert.IsType(&ast.IntLit{}, args["b"])

	_, second_instantiation, _ := given.checkAlias(g, true, 0, cached_args)
	assert.Same(instantiation, second_instantiation)

	// generic test with list types

	given = createParser(t, parser{
		tokens: scanTokens(t, `foo 1 (eine Liste, die aus 1, 2, 3 besteht)`),
	})

	genericType := &ddptypes.GenericType{Name: "T"}
	genericFunc = &ast.FuncDecl{
		NameTok:    token.Token{Literal: "bar"},
		Mod:        given.module,
		ReturnType: ddptypes.VoidType{},
		Parameters: []ast.ParameterInfo{
			{
				Name: token.Token{Literal: "a"},
				Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false},
			},
			{
				Name: token.Token{Literal: "b"},
				Type: ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
			},
		},
		Generic: &ast.GenericFuncInfo{
			Types: map[string]*ddptypes.GenericType{"T": genericType},
			Tokens: scanTokens(t, `:
Ende`),
			Context:        ast.GenericContext{Symbols: given.scope(), Aliases: given.aliases},
			Instantiations: make(map[*ast.Module][]*ast.FuncDecl),
		},
	}

	g = scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
	})
	g.(*ast.FuncAlias).Func = genericFunc

	cached_args = make(map[cachedArgKey]*cachedArg, 4)
	args, instantiation, errs = given.checkAlias(g, true, 0, cached_args)
	assert.Empty(errs)
	assert.NotEmpty(args)
	assert.NotNil(instantiation)
	assert.IsType(&ast.IntLit{}, args["a"])
	assert.IsType(&ast.Grouping{}, args["b"])
	assert.IsType(&ast.ListLit{}, args["b"].(*ast.Grouping).Expr)

	_, second_instantiation, _ = given.checkAlias(g, true, 0, cached_args)
	assert.Same(instantiation, second_instantiation)

	// test it with not-working instantiation

	given = createParser(t, parser{
		tokens: scanTokens(t, `foo 1 (eine Liste, die aus "a", "b", "c" besteht)`),
	})

	genericType = &ddptypes.GenericType{Name: "T"}
	genericFunc = &ast.FuncDecl{
		NameTok:    token.Token{Literal: "bar"},
		Mod:        given.module,
		ReturnType: ddptypes.VoidType{},
		Parameters: []ast.ParameterInfo{
			{
				Name: token.Token{Literal: "a"},
				Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false},
			},
			{
				Name: token.Token{Literal: "b"},
				Type: ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
			},
		},
		Generic: &ast.GenericFuncInfo{
			Types: map[string]*ddptypes.GenericType{"T": genericType},
			Tokens: scanTokens(t, `:
	Speichere a in b an der Stelle 1.
Ende`),
			Context:        ast.GenericContext{Symbols: given.scope(), Aliases: given.aliases},
			Instantiations: make(map[*ast.Module][]*ast.FuncDecl),
		},
	}

	g = scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
	})
	g.(*ast.FuncAlias).Func = genericFunc

	cached_args = make(map[cachedArgKey]*cachedArg, 4)
	args, instantiation, errs = given.checkAlias(g, true, 0, cached_args)
	assert.NotEmpty(errs)

	// test it with not-working instantiation and recusive generic functions

	given = createParser(t, parser{
		tokens: scanTokens(t, `foo 1 (eine Liste, die aus "a", "b", "c" besteht)`),
	})

	genericType = &ddptypes.GenericType{Name: "T"}
	genericFunc = &ast.FuncDecl{
		NameTok:    token.Token{Literal: "bar"},
		Mod:        given.module,
		ReturnType: ddptypes.VoidType{},
		Parameters: []ast.ParameterInfo{
			{
				Name: token.Token{Literal: "a"},
				Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false},
			},
			{
				Name: token.Token{Literal: "b"},
				Type: ddptypes.ParameterType{Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
			},
		},
		Generic: &ast.GenericFuncInfo{
			Types: map[string]*ddptypes.GenericType{"T": genericType},
			Tokens: scanTokens(t, `:
	Speichere a in b an der Stelle 1.
Ende`),
			Context:        ast.GenericContext{Symbols: given.scope(), Aliases: given.aliases},
			Instantiations: make(map[*ast.Module][]*ast.FuncDecl),
		},
	}

	g = scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: ddptypes.ListType{Underlying: genericType}, IsReference: false},
	})
	g.(*ast.FuncAlias).Func = genericFunc

	cached_args = make(map[cachedArgKey]*cachedArg, 4)
	args, instantiation, errs = given.checkAlias(g, true, 0, cached_args)
	assert.NotEmpty(errs)
}
