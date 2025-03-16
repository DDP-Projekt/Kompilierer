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
	ty, ok := CastDeeplyNestedGeneric(generic)
	assert.True(ok)
	assert.Equal(generic, ty)

	ty, ok = CastDeeplyNestedGeneric(ZAHL)
	assert.False(ok)
	assert.Equal(GenericType{}, ty)

	ty, ok = CastDeeplyNestedGeneric(ListType{Underlying: generic})
	assert.True(ok)
	assert.Equal(generic, ty)

	ty, ok = CastDeeplyNestedGeneric(ListType{Underlying: ListType{Underlying: generic}})
	assert.True(ok)
	assert.Equal(generic, ty)
}

func TestGetInstantiatedType(t *testing.T) {
	assert := assert.New(t)

	instantiated := GetInstantiatedType(ZAHL, nil)
	assert.Equal(ZAHL, instantiated)

	instantiated = GetInstantiatedType(ListType{Underlying: ZAHL}, nil)
	assert.Equal(ListType{Underlying: ZAHL}, instantiated)

	instantiated = GetInstantiatedType(GenericType{Name: "T"}, map[string]Type{"T": ZAHL})
	assert.Equal(ZAHL, instantiated)

	instantiated = GetInstantiatedType(ListType{Underlying: GenericType{Name: "T"}}, map[string]Type{"T": ZAHL})
	assert.Equal(ListType{Underlying: ZAHL}, instantiated)
}

func TestUnifyGenericType(t *testing.T) {
	assert := assert.New(t)

	typ := UnifyGenericType(ZAHL, ParameterType{Type: ZAHL}, nil)
	assert.Equal(ZAHL, typ)

	genericTypes := map[string]Type{}
	typ = UnifyGenericType(ZAHL, ParameterType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ZAHL, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{"T": ZAHL}
	typ = UnifyGenericType(ZAHL, ParameterType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ZAHL, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{"T": TEXT}
	typ = UnifyGenericType(ZAHL, ParameterType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(TEXT, typ)
	assert.Equal(map[string]Type{"T": TEXT}, genericTypes)

	// with lists

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{Underlying: ZAHL}, ParameterType{Type: ListType{Underlying: GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(ListType{Underlying: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{Underlying: ZAHL}, ParameterType{Type: GenericType{Name: "T"}}, genericTypes)
	assert.Equal(ListType{Underlying: ZAHL}, typ)
	assert.Equal(map[string]Type{"T": ListType{Underlying: ZAHL}}, genericTypes)

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ZAHL, ParameterType{Type: ListType{Underlying: GenericType{Name: "T"}}}, genericTypes)
	assert.Equal(nil, typ)
	assert.NotContains(genericTypes, "T")

	genericTypes = map[string]Type{}
	typ = UnifyGenericType(ListType{Underlying: ListType{Underlying: ZAHL}}, ParameterType{Type: ListType{Underlying: ListType{Underlying: GenericType{Name: "T"}}}}, genericTypes)
	assert.Equal(ListType{Underlying: ListType{Underlying: ZAHL}}, typ)
	assert.Equal(map[string]Type{"T": ZAHL}, genericTypes)
}
