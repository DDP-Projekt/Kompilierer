package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// often used types declared here to shorten their names
var (
	i8  = types.I8
	i32 = types.I32
	i64 = types.I64

	// convenience declarations for often used types
	ddpint   = i64
	ddpfloat = types.Double
	ddpbool  = types.I1
	ddpchar  = i32

	ptr = types.NewPointer

	zero  = newInt(0) // 0: i64
	zerof = constant.NewFloat(ddpfloat, 0)
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

func newIntT(typ *types.IntType, value int64) *constant.Int {
	return constant.NewInt(typ, value)
}

// turn a ddptypes.Type into the corresponding llvm type
func (c *compiler) toIrType(ddpType ddptypes.Type) ddpIrType {
	if ddpType.IsList {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return c.ddpintlist
		case ddptypes.KOMMAZAHL:
			return c.ddpfloatlist
		case ddptypes.BOOLEAN:
			return c.ddpboollist
		case ddptypes.BUCHSTABE:
			return c.ddpcharlist
		case ddptypes.TEXT:
			return c.ddpstringlist
		}
	} else {
		switch ddpType.Primitive {
		case ddptypes.ZAHL:
			return c.ddpinttyp
		case ddptypes.KOMMAZAHL:
			return c.ddpfloattyp
		case ddptypes.BOOLEAN:
			return c.ddpbooltyp
		case ddptypes.BUCHSTABE:
			return c.ddpchartyp
		case ddptypes.TEXT:
			return c.ddpstring
		case ddptypes.NICHTS:
			return c.void
		}
	}
	err("illegal ddp type to ir type conversion (%s)", ddpType)
	return nil // unreachable
}

// used to handle possible reference parameters
func (c *compiler) toIrParamType(ty ddptypes.ParameterType) types.Type {
	irType := c.toIrType(ty.Type)

	if !ty.IsReference && irType.IsPrimitive() {
		return irType.IrType()
	}

	return irType.PtrType()
}

func (c *compiler) getListType(ty ddpIrType) *ddpIrListType {
	switch ty {
	case c.ddpinttyp:
		return c.ddpintlist
	case c.ddpfloattyp:
		return c.ddpfloatlist
	case c.ddpbooltyp:
		return c.ddpboollist
	case c.ddpchartyp:
		return c.ddpcharlist
	case c.ddpstring:
		return c.ddpstringlist
	}
	err("no list type found for elementType %s", ty.Name())
	return nil // unreachable
}
