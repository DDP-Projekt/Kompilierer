package scanner

import (
	"testing"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"github.com/stretchr/testify/assert"
)

func testHandler(t *testing.T) ddperror.Handler {
	return func(err ddperror.Message) {
		t.Error(err.Error())
	}
}

func TestScanLiterals(t *testing.T) {
	assert := assert.New(t)
	errHndl := testHandler(t)

	s, err := New("", []byte(`1 1,5 "test" "" 'H' 'Ü'`), errHndl, ModeStrictCapitalization)
	assert.Nil(err, err)

	assert.Equal(token.INT, s.NextToken().Type)
	assert.Equal(token.FLOAT, s.NextToken().Type)
	assert.Equal(token.STRING, s.NextToken().Type)
	assert.Equal(token.STRING, s.NextToken().Type)
	assert.Equal(token.CHAR, s.NextToken().Type)
	assert.Equal(token.CHAR, s.NextToken().Type)
	assert.Equal(token.EOF, s.NextToken().Type)
	assert.Equal(token.EOF, s.NextToken().Type)
	assert.Equal(token.EOF, s.NextToken().Type)
}
