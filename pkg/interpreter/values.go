package interpreter

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

type value interface {
	Type() token.TokenType
}

type ddpint int64
type ddpfloat float64
type ddpbool bool
type ddpchar rune
type ddpstring string

func (ddpint) Type() token.TokenType {
	return token.ZAHL
}
func (ddpfloat) Type() token.TokenType {
	return token.KOMMAZAHL
}
func (ddpbool) Type() token.TokenType {
	return token.BOOLEAN
}
func (ddpchar) Type() token.TokenType {
	return token.BUCHSTABE
}
func (ddpstring) Type() token.TokenType {
	return token.TEXT
}
