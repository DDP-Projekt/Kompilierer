package ddptypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func fields(types ...Type) *StructType {
	result := &StructType{}
	for _, t := range types {
		result.Fields = append(result.Fields, StructField{Type: t})
	}
	return result
}

func TestStructurallyEqual(t *testing.T) {
	assert := assert.New(t)
	testCases := []struct {
		t1, t2   *StructType
		expected bool
	}{
		{fields(ZAHL), fields(ZAHL), true},
		{fields(TEXT), fields(ZAHL), false},
		{fields(&TypeAlias{Underlying: ZAHL}), fields(ZAHL), true},
		{fields(&TypeAlias{Underlying: TEXT}), fields(ZAHL), false},
		{fields(&TypeDef{Underlying: ZAHL}), fields(ZAHL), true},
		{fields(&TypeDef{Underlying: TEXT}), fields(ZAHL), false},
		{fields(ZAHL, TEXT), fields(ZAHL, TEXT), true},
		{fields(TEXT, ZAHL), fields(ZAHL, TEXT), false},
		{fields(ZAHL, fields(ZAHL, TEXT)), fields(ZAHL, fields(ZAHL, TEXT)), true},
		{fields(ZAHL, fields(TEXT, ZAHL)), fields(ZAHL, fields(ZAHL, TEXT)), false},
		{fields(ReferenceType{Type: ZAHL}), fields(ReferenceType{Type: ZAHL}), true},
		{fields(ReferenceType{Type: TEXT}), fields(ReferenceType{Type: ZAHL}), false},
	}

	for i, testCase := range testCases {
		if !assert.Equal(testCase.expected, StructurallyEqual(testCase.t1, testCase.t2)) {
			t.Log("Failed Test:", i)
		}
	}
}

func TestCastDeeplyNestedGeneric(t *testing.T) {
	assert := assert.New(t)

	generic := GenericType{Name: "generic"}
	types, ok := CastDeeplyNestedGenerics(generic)
	assert.True(ok)
	assert.Equal([]GenericType{generic}, types)

	types, ok = CastDeeplyNestedGenerics(ZAHL)
	assert.False(ok)
	assert.Equal(0, len(types))

	types, ok = CastDeeplyNestedGenerics(ListType{ElementType: generic})
	assert.True(ok)
	assert.Equal([]GenericType{generic}, types)

	types, ok = CastDeeplyNestedGenerics(ReferenceType{Type: generic})
	assert.True(ok)
	assert.Equal([]GenericType{generic}, types)

	types, ok = CastDeeplyNestedGenerics(ListType{ElementType: ListType{ElementType: generic}})
	assert.True(ok)
	assert.Equal([]GenericType{generic}, types)

	types, ok = CastDeeplyNestedGenerics(ReferenceType{Type: ReferenceType{Type: generic}})
	assert.True(ok)
	assert.Equal([]GenericType{generic}, types)

	types, ok = CastDeeplyNestedGenerics(
		&StructType{
			Fields: []StructField{
				{Type: ZAHL},
				{Type: GenericType{Name: "T"}},
				{Type: GenericType{Name: "R"}},
				{Type: GenericType{Name: "R"}},
				{Type: ListType{ElementType: GenericType{Name: "R"}}},
				{Type: ListType{ElementType: GenericType{Name: "Z"}}},
			},
		},
	)
	assert.True(ok)
	assert.Equal([]GenericType{{Name: "T"}, {Name: "R"}, {Name: "Z"}}, types)

	types, ok = CastDeeplyNestedGenerics(
		&StructType{
			Fields: []StructField{
				{Type: ZAHL},
				{Type: GenericType{Name: "T"}},
				{Type: GenericType{Name: "R"}},
				{Type: GenericType{Name: "R"}},
				{Type: ReferenceType{Type: GenericType{Name: "R"}}},
				{Type: ListType{ElementType: GenericType{Name: "Z"}}},
			},
		},
	)
	assert.True(ok)
	assert.Equal([]GenericType{{Name: "T"}, {Name: "R"}, {Name: "Z"}}, types)
}

func TestGetInstantiatedType(t *testing.T) {
	assert := assert.New(t)

	instantiated := GetInstantiatedType(ZAHL, nil)
	assert.Equal(ZAHL, instantiated)

	instantiated = GetInstantiatedType(ListType{ElementType: ZAHL}, nil)
	assert.Equal(ListType{ElementType: ZAHL}, instantiated)

	instantiated = GetInstantiatedType(ReferenceType{Type: ZAHL}, nil)
	assert.Equal(ReferenceType{Type: ZAHL}, instantiated)

	instantiated = GetInstantiatedType(GenericType{Name: "T"}, map[string]Type{"T": ZAHL})
	assert.Equal(ZAHL, instantiated)

	instantiated = GetInstantiatedType(ListType{ElementType: GenericType{Name: "T"}}, map[string]Type{"T": ZAHL})
	assert.Equal(ListType{ElementType: ZAHL}, instantiated)

	instantiated = GetInstantiatedType(ReferenceType{Type: GenericType{Name: "T"}}, map[string]Type{"T": ZAHL})
	assert.Equal(ReferenceType{Type: ZAHL}, instantiated)

	instantiated = GetInstantiatedType(ReferenceType{Type: ListType{ElementType: GenericType{Name: "T"}}}, map[string]Type{"T": ZAHL})
	assert.Equal(ReferenceType{Type: ListType{ElementType: ZAHL}}, instantiated)

	genericType := &GenericStructType{
		StructType: StructType{
			Fields: []StructField{
				{Type: GenericType{Name: "T"}},
				{Type: GenericType{Name: "R"}},
			},
		},
		GenericTypes: []GenericType{{Name: "T"}, {Name: "R"}},
	}

	instantiated = GetInstantiatedType(
		&StructType{
			Fields: []StructField{
				{Type: ZAHL},
				{Type: GenericType{Name: "T"}},
			},
			genericType:      genericType,
			instantiatedWith: []Type{ZAHL, GenericType{Name: "T"}},
		},
		map[string]Type{"T": ZAHL},
	)
	assert.Equal(
		[]StructField{
			{Type: ZAHL},
			{Type: ZAHL},
		},
		instantiated.(*StructType).Fields,
	)

	genericType = &GenericStructType{
		StructType: StructType{
			Fields: []StructField{
				{Type: GenericType{Name: "T"}},
				{Type: ListType{ElementType: GenericType{Name: "R"}}},
				{Type: ReferenceType{Type: GenericType{Name: "Z"}}},
			},
		},
		GenericTypes: []GenericType{{Name: "T"}, {Name: "R"}, {Name: "Z"}},
	}

	instantiated = GetInstantiatedType(
		&StructType{
			Fields: []StructField{
				{Type: ZAHL},
				{Type: ListType{ElementType: GenericType{Name: "T"}}},
				{Type: ReferenceType{Type: GenericType{Name: "T"}}},
			},
			genericType:      genericType,
			instantiatedWith: []Type{ZAHL, GenericType{Name: "T"}, GenericType{Name: "T"}},
		},
		map[string]Type{"T": ZAHL},
	)
	assert.Equal(
		[]StructField{
			{Type: ZAHL},
			{Type: ListType{ElementType: ZAHL}},
			{Type: ReferenceType{Type: ZAHL}},
		},
		instantiated.(*StructType).Fields,
	)
}

func TestUnifyGenericType(t *testing.T) {
	assert := assert.New(t)

	typ := UnifyGenericType(ZAHL, ZAHL, nil)
	assert.Equal(ZAHL, typ)

	genericTypes := map[string]Type{}
	typ = UnifyGenericType(ZAHL, GenericType{Name: "T"}, genericTypes)
	assert.Equal(ZAHL, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{"T": ZAHL}
	typ = UnifyGenericType(ZAHL, GenericType{Name: "T"}, genericTypes)
	assert.Equal(ZAHL, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{"T": TEXT}
	typ = UnifyGenericType(ZAHL, GenericType{Name: "T"}, genericTypes)
	assert.Equal(TEXT, typ)
	assert.Equal(map[string]Type{"T": TEXT}, genericTypes)

	// with lists

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{ElementType: ZAHL}, ListType{ElementType: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ListType{ElementType: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{ElementType: ZAHL}, GenericType{Name: "T"}, genericTypes)
	assert.Equal(ListType{ElementType: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ListType{ElementType: ZAHL}}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ZAHL, ListType{ElementType: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(nil, typ)
	assert.NotContains(genericTypes, "T")

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{ElementType: ListType{ElementType: ZAHL}}, ListType{ElementType: ListType{ElementType: GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(ListType{ElementType: ListType{ElementType: ZAHL}}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// with references

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ZAHL}, ReferenceType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ReferenceType{Type: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// genericTypes = map[string]Type{}
	// typ = UnifyGenericType(ReferenceType{Type: ZAHL}, GenericType{Name: "T"}, genericTypes)
	// assert.Equal(ReferenceType{Type: ZAHL}, typ)
	// assert.Equal(map[string]Type{"T": ReferenceType{Type: ZAHL}}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ZAHL, ReferenceType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(nil, typ)
	assert.NotContains(genericTypes, "T")

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ReferenceType{Type: ZAHL}}, ReferenceType{Type: ReferenceType{Type: GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(ReferenceType{Type: ReferenceType{Type: ZAHL}}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// special case where one level of reference indirection is removed
	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ReferenceType{Type: ZAHL}}, ReferenceType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ReferenceType{Type: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ZAHL}, GenericType{Name: "T"}, genericTypes)
	assert.Equal(ZAHL, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// higher level of references

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ReferenceType{Type: ZAHL}}, ReferenceType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ReferenceType{Type: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// mixed

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ReferenceType{Type: ListType{ElementType: ZAHL}}, ReferenceType{Type: ListType{ElementType: GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(ReferenceType{Type: ListType{ElementType: ZAHL}}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	// with structs

	genericType := &GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: GenericType{Name: "A"}},
				{Type: GenericType{Name: "B"}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "A"},
			{Name: "B"},
		},
		Instantiations: []*StructType{nil, nil},
	}

	genericType.Instantiations[0] = &StructType{
		Name: "Generic",
		Fields: []StructField{
			{Type: GenericType{Name: "T"}},
			{Type: GenericType{Name: "R"}},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			GenericType{Name: "T"},
			GenericType{Name: "R"},
		},
	}

	genericType.Instantiations[1] = &StructType{
		Fields: []StructField{
			{Type: ZAHL},
			{Type: TEXT},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			ZAHL,
			TEXT,
		},
	}

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(
		genericType.Instantiations[1],
		genericType.Instantiations[0],
		genericTypes,
	)
	if assert.NotNil(typ) {
		assert.Equal([]StructField{{Type: ZAHL}, {Type: TEXT}}, typ.(*StructType).Fields)
		assert.Equal(map[string]Type{"T": ZAHL, "R": TEXT}, genericTypes)
		assert.Len(genericType.Instantiations, 2)
		assert.Contains(genericType.Instantiations, typ)
		assert.Same(genericType.Instantiations[1], typ)
	}

	genericType = &GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: GenericType{Name: "A"}},
				{Type: GenericType{Name: "B"}},
				{Type: GenericType{Name: "C"}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "A"},
			{Name: "B"},
			{Name: "C"},
		},
		Instantiations: []*StructType{nil, nil},
	}

	genericType.Instantiations[0] = &StructType{
		Name: "Generic",
		Fields: []StructField{
			{Type: GenericType{Name: "T"}},
			{Type: GenericType{Name: "R"}},
			{Type: TEXT},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			GenericType{Name: "T"},
			GenericType{Name: "R"},
			TEXT,
		},
	}

	genericType.Instantiations[1] = &StructType{
		Fields: []StructField{
			{Type: ZAHL},
			{Type: TEXT},
			{Type: KOMMAZAHL},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			ZAHL,
			TEXT,
			KOMMAZAHL,
		},
	}

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(
		genericType.Instantiations[1],
		genericType.Instantiations[0],
		genericTypes,
	)
	assert.Nil(typ)

	// structs and multilevel references

	genericType = &GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: GenericType{Name: "A"}},
				{Type: GenericType{Name: "B"}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "A"},
			{Name: "B"},
		},
		Instantiations: []*StructType{nil, nil},
	}

	genericType.Instantiations[0] = &StructType{
		Name: "Generic",
		Fields: []StructField{
			{Type: GenericType{Name: "T"}},
			{Type: GenericType{Name: "R"}},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			GenericType{Name: "T"},
			GenericType{Name: "R"},
		},
	}

	genericType.Instantiations[1] = &StructType{
		Name: "Generic",
		Fields: []StructField{
			{Type: ZAHL},
			{Type: TEXT},
		},
		genericType: genericType,
		instantiatedWith: []Type{
			ZAHL,
			TEXT,
		},
	}

	// a function that takes &T should be instantiatable with an argument of type &&T, as &&T can be dereferenced to &T
	genericTypes = map[string]Type{}
	typ = UnifyGenericType(
		ReferenceType{Type: ReferenceType{genericType.Instantiations[1]}},
		ReferenceType{genericType.Instantiations[0]},
		genericTypes,
	)
	if assert.NotNil(typ) {
		assert.Equal([]StructField{{Type: ZAHL}, {Type: TEXT}}, typ.(ReferenceType).Type.(*StructType).Fields)
		assert.Equal(map[string]Type{"T": ZAHL, "R": TEXT}, genericTypes)
		assert.Len(genericType.Instantiations, 2)
		assert.Contains(genericType.Instantiations, typ.(ReferenceType).Type)
		assert.Same(genericType.Instantiations[1], typ.(ReferenceType).Type)
	}

	// same for &&&T
	genericTypes = map[string]Type{}
	typ = UnifyGenericType(
		ReferenceType{Type: ReferenceType{Type: ReferenceType{genericType.Instantiations[1]}}},
		ReferenceType{genericType.Instantiations[0]},
		genericTypes,
	)
	if assert.NotNil(typ) {
		assert.Equal([]StructField{{Type: ZAHL}, {Type: TEXT}}, typ.(ReferenceType).Type.(*StructType).Fields)
		assert.Equal(map[string]Type{"T": ZAHL, "R": TEXT}, genericTypes)
		assert.Len(genericType.Instantiations, 2)
		assert.Contains(genericType.Instantiations, typ.(ReferenceType).Type)
		assert.Same(genericType.Instantiations[1], typ.(ReferenceType).Type)
	}
}

func TestGetInstantiatedStructType(t *testing.T) {
	assert := assert.New(t)

	genericStruct := &GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: ZAHL},
				{Type: GenericType{Name: "T"}},
				{Type: GenericType{Name: "R"}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "T"},
			{Name: "R"},
		},
	}

	instantiated_original := GetInstantiatedStructType(genericStruct, []Type{ZAHL, KOMMAZAHL})
	instantiated2 := GetInstantiatedStructType(genericStruct, []Type{ZAHL, KOMMAZAHL})
	assert.Equal([]StructField{{Type: ZAHL}, {Type: ZAHL}, {Type: KOMMAZAHL}}, instantiated_original.Fields)
	assert.Equal([]StructField{{Type: ZAHL}, {Type: ZAHL}, {Type: KOMMAZAHL}}, instantiated2.Fields)
	assert.Same(instantiated_original, instantiated2)

	instantiated := GetInstantiatedStructType(&GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: ZAHL},
				{Type: ListType{ElementType: GenericType{Name: "R"}}},
				{Type: ReferenceType{Type: GenericType{Name: "R"}}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "R"},
		},
	}, []Type{KOMMAZAHL})

	assert.Equal([]StructField{{Type: ZAHL}, {Type: ListType{ElementType: KOMMAZAHL}}, {Type: ReferenceType{Type: KOMMAZAHL}}}, instantiated.Fields)

	instantiatedGeneric := GetInstantiatedStructType(genericStruct, []Type{ZAHL, GenericType{Name: "R"}})

	instantiated = GetInstantiatedStructType(&GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: ZAHL},
				{Type: instantiatedGeneric},
			},
		},
		GenericTypes: []GenericType{
			{Name: "R"},
		},
	}, []Type{KOMMAZAHL})

	assert.True(Equal(instantiated_original, instantiated.Fields[1].Type))
	assert.Equal(instantiated_original, instantiated.Fields[1].Type)

	genericStruct = &GenericStructType{
		StructType: StructType{
			Name: "Generic",
			Fields: []StructField{
				{Type: ZAHL},
				{Type: GenericType{Name: "T"}},
				{Type: GenericType{Name: "R"}},
			},
		},
		GenericTypes: []GenericType{
			{Name: "T"},
			{Name: "R"},
		},
	}

	instantiated = GetInstantiatedStructType(genericStruct, []Type{ZAHL})
	assert.Nil(instantiated)
	assert.Len(genericStruct.Instantiations, 0)
}

func TestEqualDeref(t *testing.T) {
	assert := assert.New(t)

	assert.False(EqualDeref(ZAHL, TEXT))
	assert.False(EqualDeref(ReferenceType{Type: TEXT}, ZAHL))
	assert.False(EqualDeref(ZAHL, ReferenceType{Type: TEXT}))

	assert.True(EqualDeref(ZAHL, ZAHL))
	assert.True(EqualDeref(ZAHL, ReferenceType{Type: ZAHL}))
	assert.True(EqualDeref(ReferenceType{Type: ZAHL}, ZAHL))
}

func TestIsDereferencableTo(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsDereferencableTo(ZAHL, ZAHL))
	assert.True(IsDereferencableTo(ReferenceType{ZAHL}, ZAHL))
	assert.True(IsDereferencableTo(ReferenceType{ReferenceType{ZAHL}}, ZAHL))

	assert.False(IsDereferencableTo(ZAHL, BUCHSTABE))
	assert.False(IsDereferencableTo(ReferenceType{ZAHL}, BUCHSTABE))
	assert.False(IsDereferencableTo(ReferenceType{ReferenceType{ZAHL}}, BUCHSTABE))

	assert.False(IsDereferencableTo(BUCHSTABE, ZAHL))
	assert.False(IsDereferencableTo(ReferenceType{BUCHSTABE}, ZAHL))
	assert.False(IsDereferencableTo(ReferenceType{ReferenceType{BUCHSTABE}}, ZAHL))
}

func TestRefDepth(t *testing.T) {
	assert := assert.New(t)

	assert.Equal(RefDepth(ZAHL), uint(0))
	assert.Equal(RefDepth(ReferenceType{Type: ZAHL}), uint(1))
	assert.Equal(RefDepth(ReferenceType{ReferenceType{Type: ZAHL}}), uint(2))
	assert.Equal(RefDepth(ReferenceType{ReferenceType{Type: ListType{ElementType: ReferenceType{Type: ZAHL}}}}), uint(2))
	assert.Equal(RefDepth(ReferenceType{&TypeDef{Underlying: ReferenceType{Type: ListType{ElementType: ReferenceType{Type: ZAHL}}}}}), uint(2))
}

func TestIsReferenceTo(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsReferenceTo(ReferenceType{Type: ZAHL}, ZAHL))
	assert.True(IsReferenceTo(ReferenceType{Type: ReferenceType{Type: ZAHL}}, ZAHL))
	assert.True(IsReferenceTo(ReferenceType{Type: ReferenceType{Type: ZAHL}}, ReferenceType{Type: ZAHL}))

	assert.False(IsReferenceTo(ReferenceType{Type: ZAHL}, TEXT))
	assert.False(IsReferenceTo(ReferenceType{Type: ReferenceType{Type: TEXT}}, ZAHL))
	assert.False(IsReferenceTo(ReferenceType{Type: ReferenceType{Type: TEXT}}, ReferenceType{Type: ZAHL}))
}

func TestIsAssigneableTo(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsAssigneableTo(ZAHL, ZAHL))
	assert.True(IsAssigneableTo(ZAHL, ReferenceType{Type: ZAHL}))
	assert.True(IsAssigneableTo(ReferenceType{ZAHL}, ReferenceType{Type: ZAHL}))
	assert.True(IsAssigneableTo(ReferenceType{ZAHL}, ZAHL))
	assert.True(IsAssigneableTo(ZAHL, VARIABLE))
	assert.True(IsAssigneableTo(ZAHL, ReferenceType{Type: VARIABLE}))

	// numeric casts
	assert.True(IsAssigneableTo(ZAHL, KOMMAZAHL))
	assert.True(IsAssigneableTo(ZAHL, BYTE))
	assert.True(IsAssigneableTo(KOMMAZAHL, BYTE))

	// errors
	assert.False(IsAssigneableTo(ZAHL, TEXT))
	assert.False(IsAssigneableTo(ZAHL, ReferenceType{Type: TEXT}))
	assert.False(IsAssigneableTo(VARIABLE, ZAHL))
}

func TestIsPasseableAsParam(t *testing.T) {
	assert := assert.New(t)

	assert.True(IsPasseableAsParam(ZAHL, ZAHL, false))
	assert.True(IsPasseableAsParam(ReferenceType{Type: ZAHL}, ZAHL, false))
	assert.True(IsPasseableAsParam(ReferenceType{ZAHL}, ReferenceType{Type: ZAHL}, false))
	assert.True(IsPasseableAsParam(ZAHL, VARIABLE, false))
	assert.True(IsPasseableAsParam(VARIABLE, VARIABLE, false))
	assert.True(IsPasseableAsParam(ZAHL, ReferenceType{Type: VARIABLE}, false))
	// assert.True(IsPasseableAsParam(ReferenceType{Type: VARIABLE}, ZAHL))

	// TODO: commented out as there are matching issues with getting the most fitting function in alias()
	// numeric casts
	// assert.True(IsPasseableAsParam(ZAHL, KOMMAZAHL))
	// assert.True(IsPasseableAsParam(ZAHL, BYTE))
	// assert.True(IsPasseableAsParam(KOMMAZAHL, BYTE))
	// assert.True(IsPasseableAsParam(KOMMAZAHL, ZAHL))
	// assert.True(IsPasseableAsParam(BYTE, KOMMAZAHL))
	// assert.True(IsPasseableAsParam(BYTE, ZAHL))

	assert.False(IsPasseableAsParam(ZAHL, ReferenceType{Type: ZAHL}, false))
	assert.True(IsPasseableAsParam(ZAHL, ReferenceType{Type: ZAHL}, true))

	// errors
	assert.False(IsPasseableAsParam(ZAHL, ReferenceType{Type: ZAHL}, false))
	assert.False(IsPasseableAsParam(ZAHL, TEXT, false))
	assert.False(IsPasseableAsParam(ZAHL, ReferenceType{Type: TEXT}, false))
	assert.False(IsPasseableAsParam(VARIABLE, ZAHL, false))
}
