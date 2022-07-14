package compiler

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// often used types declared here to shorten their names
var (
	void = types.Void

	i8  = types.I8
	i32 = types.I32
	i64 = types.I64

	ddpint    = i64
	ddpfloat  = types.Double
	ddpbool   = types.I1
	ddpchar   = i32
	ddpstring = types.NewStruct() // defined in setupStringType
	ddpstrptr = types.NewPointer(ddpstring)

	ptr = types.NewPointer

	zero = constant.NewInt(ddpint, 0)

	VK_STRING = constant.NewInt(i8, 0)
)

const (
	all_ones = ^0 // int with all bits set to 1
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

// turn a tokenType into the corresponding llvm type
func toIRType(ddpType token.DDPType) types.Type {
	if ddpType.IsList {
		notimplemented()
	} else {
		switch ddpType.PrimitiveType {
		case token.NICHTS:
			return void
		case token.ZAHL:
			return ddpint
		case token.KOMMAZAHL:
			return ddpfloat
		case token.BOOLEAN:
			return ddpbool
		case token.BUCHSTABE:
			return ddpchar
		case token.TEXT:
			return ddpstrptr
		}
	}
	panic(fmt.Errorf("illegal ddp type to ir type conversion (%s)", ddpType.String()))
}

// returns the default constant for global variables
func getDefaultValue(ddpType token.DDPType) constant.Constant {
	if ddpType.IsPrimitive() {
		switch ddpType.PrimitiveType {
		case token.ZAHL:
			return constant.NewInt(ddpint, 0)
		case token.KOMMAZAHL:
			return constant.NewFloat(ddpfloat, 0.0)
		case token.BOOLEAN:
			return constant.NewInt(ddpbool, 0)
		case token.BUCHSTABE:
			return constant.NewInt(ddpchar, 0)
		case token.TEXT:
			return constant.NewNull(ddpstrptr)
		}
	}
	panic(fmt.Errorf("illegal ddp type to ir type conversion (%s)", ddpType.String()))
}
