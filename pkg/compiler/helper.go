package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"

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

	zero = newInt(0) // 0: i64
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

func newIntT(typ *types.IntType, value int64) *constant.Int {
	return constant.NewInt(typ, value)
}

// turn a tokenType into the corresponding llvm type
func toIRType(ddpType ddptypes.Type) types.Type {
	if ddpType.IsList {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return ddpintlistptr
		case ddptypes.KOMMAZAHL:
			return ddpfloatlistptr
		case ddptypes.BOOLEAN:
			return ddpboollistptr
		case ddptypes.BUCHSTABE:
			return ddpcharlistptr
		case ddptypes.TEXT:
			return ddpstringlistptr
		}
	} else {
		switch ddpType.Primitive {
		case ddptypes.NICHTS:
			return void
		case ddptypes.ZAHL:
			return ddpint
		case ddptypes.KOMMAZAHL:
			return ddpfloat
		case ddptypes.BOOLEAN:
			return ddpbool
		case ddptypes.BUCHSTABE:
			return ddpchar
		case ddptypes.TEXT:
			return ddpstrptr
		}
	}
	err("illegal ddp type to ir type conversion (%s)", ddpType)
	return i8 // unreachable
}

func toIRTypeRef(ty ddptypes.ParameterType) types.Type {
	if !ty.IsReference {
		return toIRType(ty.Type)
	}
	return ptr(toIRType(ty.Type))
}

// returns the default constant for global variables
func getDefaultValue(ddpType ddptypes.Type) constant.Constant {
	if ddpType.IsList {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return constant.NewNull(ddpintlistptr)
		case ddptypes.KOMMAZAHL:
			return constant.NewNull(ddpfloatlistptr)
		case ddptypes.BOOLEAN:
			return constant.NewNull(ddpboollistptr)
		case ddptypes.BUCHSTABE:
			return constant.NewNull(ddpcharlistptr)
		case ddptypes.TEXT:
			return constant.NewNull(ddpstringlistptr)
		}
	} else {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return constant.NewInt(ddpint, 0)
		case ddptypes.KOMMAZAHL:
			return constant.NewFloat(ddpfloat, 0.0)
		case ddptypes.BOOLEAN:
			return constant.NewInt(ddpbool, 0)
		case ddptypes.BUCHSTABE:
			return constant.NewInt(ddpchar, 0)
		case ddptypes.TEXT:
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

func getTypeName(ddpType ddptypes.Type) string {
	if ddpType.IsList {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return "ddpintlist"
		case ddptypes.KOMMAZAHL:
			return "ddpfloatlist"
		case ddptypes.BOOLEAN:
			return "ddpboollist"
		case ddptypes.BUCHSTABE:
			return "ddpcharlist"
		case ddptypes.TEXT:
			return "ddpstringlist"
		}
	} else {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return "ddpint"
		case ddptypes.KOMMAZAHL:
			return "ddpfloat"
		case ddptypes.BOOLEAN:
			return "ddpbool"
		case ddptypes.BUCHSTABE:
			return "ddpchar"
		case ddptypes.TEXT:
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
