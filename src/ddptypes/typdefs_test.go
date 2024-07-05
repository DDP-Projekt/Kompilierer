package ddptypes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrueUnderlying(t *testing.T) {
	assert := assert.New(t)
	typedefs := []TypeDef{
		{Underlying: ZAHL},
		{Underlying: &TypeAlias{Underlying: ZAHL}},
		{Underlying: &TypeAlias{Underlying: &TypeAlias{Underlying: ZAHL}}},
		{Underlying: &TypeDef{Underlying: ZAHL}},
		{Underlying: &TypeDef{Underlying: &TypeDef{Underlying: ZAHL}}},
		{Underlying: &TypeDef{Underlying: &TypeAlias{Underlying: ZAHL}}},
		{Underlying: &TypeAlias{Underlying: &TypeDef{Underlying: ZAHL}}},
	}

	for i := range typedefs {
		assert.Equal(ZAHL, TrueUnderlying(&typedefs[i]))
	}
}
