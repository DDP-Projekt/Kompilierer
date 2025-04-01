package parser

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func TestParseTypeGeneric(t *testing.T) {
	assert := assert.New(t)

	runTest := func(src, declName string, genericFields []ddptypes.StructField, genericTypes []ddptypes.GenericType, resultFields []ddptypes.StructField) {
		mockHandler := ddperror.Collector{}
		symbols := ast.NewSymbolTable(nil)
		decl := &ast.StructDecl{
			NameTok: token.Token{Literal: declName},
			Type: &ddptypes.GenericStructType{
				StructType: ddptypes.StructType{
					Name:   declName,
					Fields: genericFields,
				},
				GenericTypes: genericTypes,
			},
		}
		symbols.InsertDecl(declName, decl)
		given := createParser(t, parser{
			tokens:       scanTokens(t, src),
			errorHandler: mockHandler.GetHandler(),
		})
		given.setScope(symbols)

		typ := given.parseType(false)
		assert.False(mockHandler.DidError())
		assert.NotNil(typ)
		assert.IsType(&ddptypes.StructType{}, typ)
		assert.Equal(resultFields, typ.(*ddptypes.StructType).Fields)
	}

	runTest(`Zahl-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}},
	)
	runTest(`Zahl-Kommazahl-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.KOMMAZAHL}},
	)
	runTest(`Zahl-Kommazahlen Liste-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}}},
	)
	runTest(`Zahl-(Kommazahlen Liste)-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}}},
	)

	// lists

	mockHandler := ddperror.Collector{}
	symbols := ast.NewSymbolTable(nil)
	decl := &ast.StructDecl{
		NameTok: token.Token{Literal: "Vektor"},
		Type: &ddptypes.GenericStructType{
			StructType: ddptypes.StructType{
				Name:   "Vektor",
				Fields: []ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
			},
			GenericTypes: []ddptypes.GenericType{{Name: "T"}},
		},
	}
	symbols.InsertDecl("Vektor", decl)
	given := createParser(t, parser{
		tokens:       scanTokens(t, `Zahl-Vektor Liste`),
		errorHandler: mockHandler.GetHandler(),
	})
	given.setScope(symbols)

	typ := given.parseType(false)
	if assert.False(mockHandler.DidError()) {
		assert.NotNil(typ)
		assert.True(ddptypes.IsList(typ))
		assert.Equal([]ddptypes.StructField{{Type: ddptypes.ZAHL}}, typ.(ddptypes.ListType).Underlying.(*ddptypes.StructType).Fields)
	}
}

func TestParseReferenceType(t *testing.T) {
	assert := assert.New(t)

	runTest := func(src string, isGeneric, shouldError, shouldBeRef bool, expectedType ddptypes.Type) {
		mockHandler := ddperror.Collector{}
		given := createParser(t, parser{
			tokens:       scanTokens(t, src),
			errorHandler: mockHandler.GetHandler(),
		})

		typ, isRef := given.parseReferenceType(isGeneric)
		assert.Equal(shouldError, mockHandler.DidError())
		assert.Equal(expectedType, typ)
		assert.Equal(shouldBeRef, isRef)
	}

	runTest(`Zahl`, false, false, false, ddptypes.ZAHL)
	runTest(`Zahlen Liste`, false, false, false, ddptypes.ListType{Underlying: ddptypes.ZAHL})
	runTest(`Zahlen Referenz`, false, false, true, ddptypes.ZAHL)

	runTest(`T`, false, true, false, nil)

	runTest(`T`, true, false, false, ddptypes.GenericType{Name: "T"})
	runTest(`T Liste`, true, false, false, ddptypes.ListType{Underlying: ddptypes.GenericType{Name: "T"}})
	runTest(`T Referenz`, true, false, true, ddptypes.GenericType{Name: "T"})

	runGenericTest := func(src, declName string, genericFields []ddptypes.StructField, genericTypes []ddptypes.GenericType, resultFields []ddptypes.StructField, isRef, success bool) {
		mockHandler := ddperror.Collector{}
		symbols := ast.NewSymbolTable(nil)
		decl := &ast.StructDecl{
			NameTok: token.Token{Literal: declName},
			Type: &ddptypes.GenericStructType{
				StructType: ddptypes.StructType{
					Name:   declName,
					Fields: genericFields,
				},
				GenericTypes: genericTypes,
			},
		}
		symbols.InsertDecl(declName, decl)
		given := createParser(t, parser{
			tokens:       scanTokens(t, src),
			errorHandler: mockHandler.GetHandler(),
		})
		given.setScope(symbols)

		typ, isReference := given.parseReferenceType(false)
		if !success {
			assert.True(mockHandler.DidError())
			return
		}

		assert.False(mockHandler.DidError())
		assert.NotNil(typ)
		assert.IsType(&ddptypes.StructType{}, typ)
		assert.Equal(resultFields, typ.(*ddptypes.StructType).Fields)
		assert.Equal(isRef, isReference)
	}

	runGenericTest(`Zahl-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}},
		false,
		true,
	)
	runGenericTest(`Zahl-Kommazahl-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.KOMMAZAHL}},
		false,
		true,
	)
	runGenericTest(`Zahl-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}},
		true,
		true,
	)
	runGenericTest(`Zahl-Kommazahlen Liste-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}}},
		true,
		true,
	)
	runGenericTest(`Zahl-(Kommazahlen Liste)-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}}},
		true,
		true,
	)
	runGenericTest(`(Zahlen Referenz)-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}},
		true,
		false,
	)
	runGenericTest(`Zahlen Referenz-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}},
		true,
		false,
	)

	// lists

	mockHandler := ddperror.Collector{}
	symbols := ast.NewSymbolTable(nil)
	decl := &ast.StructDecl{
		NameTok: token.Token{Literal: "Vektor"},
		Type: &ddptypes.GenericStructType{
			StructType: ddptypes.StructType{
				Name:   "Vektor",
				Fields: []ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
			},
			GenericTypes: []ddptypes.GenericType{{Name: "T"}},
		},
	}
	symbols.InsertDecl("Vektor", decl)
	given := createParser(t, parser{
		tokens:       scanTokens(t, `Zahl-Vektor Liste`),
		errorHandler: mockHandler.GetHandler(),
	})
	given.setScope(symbols)

	typ, isReference := given.parseReferenceType(false)
	if assert.False(mockHandler.DidError()) {
		assert.False(isReference)
		assert.NotNil(typ)
		assert.True(ddptypes.IsList(typ))
		assert.Equal([]ddptypes.StructField{{Type: ddptypes.ZAHL}}, typ.(ddptypes.ListType).Underlying.(*ddptypes.StructType).Fields)
	}
}
