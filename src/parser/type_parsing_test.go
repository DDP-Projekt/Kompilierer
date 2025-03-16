package parser

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/stretchr/testify/assert"
)

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
}
