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

func (val ddpint) Type() token.TokenType {
	return token.ZAHL
}
func (val ddpfloat) Type() token.TokenType {
	return token.KOMMAZAHL
}
func (val ddpbool) Type() token.TokenType {
	return token.BOOLEAN
}
func (val ddpchar) Type() token.TokenType {
	return token.BUCHSTABE
}
func (val ddpstring) Type() token.TokenType {
	return token.TEXT
}
