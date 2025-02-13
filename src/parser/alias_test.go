package parser

import (
	"strings"
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
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
