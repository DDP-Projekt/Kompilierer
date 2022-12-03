package token

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"
)

type TokenType int

// a single ddp token
type Token struct {
	Type      TokenType               // type of the token
	Literal   string                  // the literal from which it was scanned
	Indent    uint                    // how many levels it is indented
	File      string                  // the file from which it was scanned
	Range     Range                   // the range the token spans
	AliasInfo *ddptypes.ParameterType // only present in ALIAS_PARAMETERs, holds type information, nil otherwise
}

func (t Token) String() string {
	return t.Literal
}

func (t Token) StringVerbose() string {
	return fmt.Sprintf("[F: %s L: %d C: %d I: %d Lit: \"%s\"] Type: %s", t.File, t.Range.Start.Line, t.Range.Start.Column, t.Indent, t.Literal, t.Type)
}

// t.Range.Start.Line
func (t Token) Line() uint {
	return t.Range.Start.Line
}

// t.Range.Start.Column
func (t Token) Column() uint {
	return t.Range.Start.Column
}
