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
