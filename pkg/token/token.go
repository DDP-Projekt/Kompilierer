package token

import "fmt"

type TokenType int

type Token struct {
	Type    TokenType
	Literal string
	Indent  int
	File    string
	Line    int
	Column  int
}

func (t Token) String() string {
	return t.Type.String()
}

func (t Token) StringVerbose() string {
	return fmt.Sprintf("[F: %s L: %d C: %d I: %d Lit: \"%s\"] Type: %s", t.File, t.Line, t.Column, t.Indent, t.Literal, t.Type.String())
}
