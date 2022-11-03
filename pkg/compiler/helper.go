package compiler

import (
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

	ddpint           = i64
	ddpfloat         = types.Double
	ddpbool          = types.I1
	ddpchar          = i32
	ddpstring        = types.NewStruct() // defined in setupStringType
	ddpintlist       = types.NewStruct() // defined in setupListTypes
	ddpfloatlist     = types.NewStruct() // defined in setupListTypes
	ddpboollist      = types.NewStruct() // defined in setupListTypes
	ddpcharlist      = types.NewStruct() // defined in setupListTypes
	ddpstringlist    = types.NewStruct() // defined in setupListTypes
	ddpstrptr        = types.NewPointer(ddpstring)
	ddpintlistptr    = types.NewPointer(ddpintlist)
	ddpfloatlistptr  = types.NewPointer(ddpfloatlist)
	ddpboollistptr   = types.NewPointer(ddpboollist)
	ddpcharlistptr   = types.NewPointer(ddpcharlist)
	ddpstringlistptr = types.NewPointer(ddpstringlist)

	ptr = types.NewPointer

	zero = constant.NewInt(ddpint, 0)
)

const (
	all_ones = ^0 // int with all bits set to 1
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

func newIntT(typ *types.IntType, value int64) *constant.Int {
	return constant.NewInt(typ, value)
}

// turn a tokenType into the corresponding llvm type
func toIRType(ddpType token.DDPType) types.Type {
	if ddpType.IsList {
		switch ddpType.PrimitiveType {
		case token.ZAHL:
			return ddpintlistptr
		case token.KOMMAZAHL:
			return ddpfloatlistptr
		case token.BOOLEAN:
			return ddpboollistptr
		case token.BUCHSTABE:
			return ddpcharlistptr
		case token.TEXT:
			return ddpstringlistptr
		}
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
	err("illegal ddp type to ir type conversion (%s)", ddpType)
	return i8 // unreachable
}

func toIRTypeRef(ty token.ArgType) types.Type {
	if !ty.IsReference {
		return toIRType(ty.Type)
	}
	return ptr(toIRType(ty.Type))
}

// returns the default constant for global variables
func getDefaultValue(ddpType token.DDPType) constant.Constant {
	if ddpType.IsList {
		switch ddpType.PrimitiveType {
		case token.ZAHL:
			return constant.NewNull(ddpintlistptr)
		case token.KOMMAZAHL:
			return constant.NewNull(ddpfloatlistptr)
		case token.BOOLEAN:
			return constant.NewNull(ddpboollistptr)
		case token.BUCHSTABE:
			return constant.NewNull(ddpcharlistptr)
		case token.TEXT:
			return constant.NewNull(ddpstringlistptr)
		}
	} else {
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
	err("illegal ddp type to ir type conversion (%s)", ddpType)
	return zero // unreachable
}

func isDynamic(typ types.Type) bool {
	switch typ {
	case ddpstrptr, ddpintlistptr, ddpfloatlistptr, ddpboollistptr, ddpcharlistptr, ddpstringlistptr:
		return true
	}
	return false
}

func getTypeName(ddpType token.DDPType) string {
	if ddpType.IsList {
		switch ddpType.PrimitiveType {
		case token.ZAHL:
			return "ddpintlist"
		case token.KOMMAZAHL:
			return "ddpfloatlist"
		case token.BOOLEAN:
			return "ddpboollist"
		case token.BUCHSTABE:
			return "ddpcharlist"
		case token.TEXT:
			return "ddpstringlist"
		}
	} else {
		switch ddpType.PrimitiveType {
		case token.ZAHL:
			return "ddpint"
		case token.KOMMAZAHL:
			return "ddpfloat"
		case token.BOOLEAN:
			return "ddpbool"
		case token.BUCHSTABE:
			return "ddpchar"
		case token.TEXT:
			return "ddpstring"
		}
	}
	err("illegal ddp type to ir type conversion (%s)", ddpType)
	return "" // unreachable
}

func derefListPtr(typ types.Type) types.Type {
	switch typ {
	case ddpintlistptr:
		return ddpintlist
	case ddpfloatlistptr:
		return ddpfloatlist
	case ddpboollistptr:
		return ddpboollist
	case ddpcharlistptr:
		return ddpcharlist
	case ddpstringlistptr:
		return ddpstringlist
	}
	err("bad argument")
	return void // unreachable
}

func getElementType(typ types.Type) types.Type {
	switch typ {
	case ddpintlistptr:
		return ddpint
	case ddpfloatlistptr:
		return ddpfloat
	case ddpboollistptr:
		return ddpbool
	case ddpcharlistptr:
		return ddpchar
	case ddpstringlistptr:
		return ddpstrptr
	}
	err("bad argument")
	return void // unreachable
}
