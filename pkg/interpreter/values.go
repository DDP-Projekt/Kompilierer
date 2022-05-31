package interpreter

import (
	"KDDP/pkg/token"
)

type value interface {
	Type() token.TokenType
}

type ddpint int64
type ddpfloat float64
type ddpbool bool
type ddpchar rune
type ddpstring string

func (v ddpint) Type() token.TokenType {
	return token.ZAHL
}
func (v ddpfloat) Type() token.TokenType {
	return token.KOMMAZAHL
}
func (v ddpbool) Type() token.TokenType {
	return token.BOOLEAN
}
func (v ddpchar) Type() token.TokenType {
	return token.BUCHSTABE
}
func (v ddpstring) Type() token.TokenType {
	return token.TEXT
}
