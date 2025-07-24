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
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{ElementType: ddptypes.KOMMAZAHL}}},
	)
	runTest(`Zahl-(Kommazahlen Liste)-Vektor`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{ElementType: ddptypes.KOMMAZAHL}}},
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
		assert.Equal([]ddptypes.StructField{{Type: ddptypes.ZAHL}}, typ.(ddptypes.ListType).ElementType.(*ddptypes.StructType).Fields)
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

		typ := given.parseType(isGeneric)
		assert.Equal(shouldError, mockHandler.DidError())
		assert.Equal(expectedType, typ)
		assert.Equal(shouldBeRef, ddptypes.IsReference(typ))
	}

	runTest(`Zahl`, false, false, false, ddptypes.ZAHL)
	runTest(`Zahlen Liste`, false, false, false, ddptypes.ListType{ElementType: ddptypes.ZAHL})
	runTest(`Zahlen Referenz`, false, false, true, ddptypes.ReferenceType{Type: ddptypes.ZAHL})

	runTest(`T`, false, true, false, nil)

	runTest(`T`, true, false, false, ddptypes.GenericType{Name: "T"})
	runTest(`T Liste`, true, false, false, ddptypes.ListType{ElementType: ddptypes.GenericType{Name: "T"}})
	runTest(`T Referenz`, true, false, true, ddptypes.ReferenceType{Type: ddptypes.GenericType{Name: "T"}})

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

		typ := given.parseType(false)
		if !success {
			assert.True(mockHandler.DidError())
			return
		}

		assert.False(mockHandler.DidError())
		assert.NotNil(typ)
		if isRef {
			assert.IsType(&ddptypes.StructType{}, typ.(ddptypes.ReferenceType).Type)
			assert.Equal(resultFields, typ.(ddptypes.ReferenceType).Type.(*ddptypes.StructType).Fields)
		} else {
			assert.IsType(&ddptypes.StructType{}, typ)
			assert.Equal(resultFields, typ.(*ddptypes.StructType).Fields)
		}
		assert.Equal(isRef, ddptypes.IsReference(typ))
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
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{ElementType: ddptypes.KOMMAZAHL}}},
		true,
		true,
	)
	runGenericTest(`Zahl-(Kommazahlen Liste)-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}, {Type: ddptypes.GenericType{Name: "R"}}},
		[]ddptypes.GenericType{{Name: "T"}, {Name: "R"}},
		[]ddptypes.StructField{{Type: ddptypes.ZAHL}, {Type: ddptypes.ListType{ElementType: ddptypes.KOMMAZAHL}}},
		true,
		true,
	)
	runGenericTest(`(Zahlen Referenz)-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ReferenceType{Type: ddptypes.ZAHL}}},
		true,
		true,
	)
	runGenericTest(`Zahlen Referenz-Vektor Referenz`,
		"Vektor",
		[]ddptypes.StructField{{Type: ddptypes.GenericType{Name: "T"}}},
		[]ddptypes.GenericType{{Name: "T"}},
		[]ddptypes.StructField{{Type: ddptypes.ReferenceType{Type: ddptypes.ZAHL}}},
		true,
		true,
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
		assert.False(ddptypes.IsReference(typ))
		assert.NotNil(typ)
		assert.True(ddptypes.IsList(typ))
		assert.Equal([]ddptypes.StructField{{Type: ddptypes.ZAHL}}, typ.(ddptypes.ListType).ElementType.(*ddptypes.StructType).Fields)
	}
}
