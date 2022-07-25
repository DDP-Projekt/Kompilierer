package interpreter

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

type value interface {
	Type() token.DDPType
}

type ddpint int64
type ddpfloat float64
type ddpbool bool
type ddpchar rune
type ddpstring string

type ddpintlist []ddpint
type ddpfloatlist []ddpfloat
type ddpboollist []ddpbool
type ddpcharlist []ddpchar
type ddpstringlist []ddpstring

func (ddpint) Type() token.DDPType {
	return token.DDPIntType()
}
func (ddpfloat) Type() token.DDPType {
	return token.DDPFloatType()
}
func (ddpbool) Type() token.DDPType {
	return token.DDPBoolType()
}
func (ddpchar) Type() token.DDPType {
	return token.DDPCharType()
}
func (ddpstring) Type() token.DDPType {
	return token.DDPStringType()
}

func (ddpintlist) Type() token.DDPType {
	return token.NewListType(token.ZAHL)
}
func (ddpfloatlist) Type() token.DDPType {
	return token.NewListType(token.KOMMAZAHL)
}
func (ddpboollist) Type() token.DDPType {
	return token.NewListType(token.BOOLEAN)
}
func (ddpcharlist) Type() token.DDPType {
	return token.NewListType(token.BUCHSTABE)
}
func (ddpstringlist) Type() token.DDPType {
	return token.NewListType(token.TEXT)
}
