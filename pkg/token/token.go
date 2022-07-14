package token

import "fmt"

type TokenType int

// a single ddp token
type Token struct {
	Type      TokenType // type of the token
	Literal   string    // the literal from which it was scanned
	Indent    int       // how many levels it is indented
	File      string    // the file from which it was scanned
	Line      int       // the line on which it appeared
	Column    int       // the column it started
	AliasInfo *DDPType  // only present in ALIAS_PARAMETERs, holds type information, nil otherwise
}

func (t Token) String() string {
	return t.Type.String()
}

func (t Token) StringVerbose() string {
	return fmt.Sprintf("[F: %s L: %d C: %d I: %d Lit: \"%s\"] Type: %s", t.File, t.Line, t.Column, t.Indent, t.Literal, t.Type.String())
}
