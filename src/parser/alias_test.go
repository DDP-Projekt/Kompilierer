package parser

import (
	"strings"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
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
		Tokens:   result,
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
	args, errs := given.checkAlias(f, true, 0, cached_args)
	assert.Empty(errs)
	assert.IsType(&ast.IntLit{}, args["a"])
	assert.IsType(&ast.IntLit{}, args["b"])

	// generic test

	genericFunc := &ast.FuncDecl{
		NameTok: token.Token{Literal: "foo"},
		Parameters: []ast.ParameterInfo{
			{
				Name: token.Token{Literal: "a"},
				Type: ddptypes.ParameterType{Type: ddptypes.ZAHL, IsReference: false},
			},
		},
	}
	_ = genericFunc

	g := scanAlias(t, `foo <a> <b>`, map[string]ddptypes.ParameterType{
		"a": {Type: ddptypes.ZAHL, IsReference: false},
		"b": {Type: &ddptypes.GenericType{Name: "T"}, IsReference: false},
	})
	cached_args = make(map[cachedArgKey]*cachedArg, 4)
	args, errs = given.checkAlias(g, true, 0, cached_args)
	assert.Empty(errs)
	assert.NotEmpty(args)
	assert.IsType(&ast.IntLit{}, args["a"])
	assert.IsType(&ast.IntLit{}, args["b"])
}

func TestInstantiateGenericFunction(t *testing.T) {
	assert := assert.New(t)
	_ = assert

	t.FailNow()
}

func TestCreateGenericContext(t *testing.T) {
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

	declContext := ast.GenericContext{
		Symbols: createSymbols(
			"z1", ddptypes.ZAHL,
			"bar", &ast.FuncDecl{},
			"baz", baz_context,
			"Bar", &ast.StructDecl{
				Type: &ddptypes.StructType{Name: "Bar"},
			}),
		Aliases: declContextAliases,
	}

	context := given.generateGenericContext(declContext, nil)

	assert.NotNil(context.Symbols)
	assert.NotNil(context.Aliases)

	_, has_z, _ := context.Symbols.LookupDecl("z")
	_, has_foo, _ := context.Symbols.LookupDecl("foo")
	_, has_z1, _ := context.Symbols.LookupDecl("z1")
	_, has_bar, _ := context.Symbols.LookupDecl("bar")
	_, has_Foo, _ := context.Symbols.LookupDecl("Foo")
	_, has_Bar, _ := context.Symbols.LookupDecl("Bar")

	// variables from the parser should not be used
	assert.False(has_z)
	// everything else from both tables should be present
	assert.True(has_foo)
	assert.True(has_z1)
	assert.True(has_bar)
	assert.True(has_Foo)
	assert.True(has_Bar)

	// the original context should be prefered when the parser and context both contain a name
	bazDecl, has_baz, _ := context.Symbols.LookupDecl("baz")
	assert.True(has_baz)
	assert.Same(baz_context, bazDecl)
	assert.NotSame(baz_parser, bazDecl)

	// types as well
	FooType, has_Foo_type := context.Symbols.LookupType("Foo")
	BarType, has_Bar_type := context.Symbols.LookupType("Bar")
	assert.True(has_Foo_type)
	assert.True(has_Bar_type)
	assert.Equal("Foo", FooType.String())
	assert.Equal("Bar", BarType.String())

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
