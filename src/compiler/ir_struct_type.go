package compiler

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrStructType struct {
	typ           *types.StructType
	ptr           *types.PointerType
	fieldIrTypes  []ddpIrType
	fieldDDPTypes []ddptypes.StructField
	name          string
	freeIrFun     *ir.Func // the free ir func
	deepCopyIrFun *ir.Func // the deepCopy ir func
	equalsIrFun   *ir.Func // the equals ir func
}

var _ ddpIrType = (*ddpIrStructType)(nil)

func (t *ddpIrStructType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrStructType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrStructType) Name() string {
	return t.name
}

func (*ddpIrStructType) IsPrimitive() bool {
	return false
}

func (t *ddpIrStructType) DefaultValue() constant.Constant {
	return constant.NewZeroInitializer(t.typ)
}

func (t *ddpIrStructType) FreeFunc() *ir.Func {
	return t.freeIrFun
}

func (t *ddpIrStructType) DeepCopyFunc() *ir.Func {
	return t.deepCopyIrFun
}

func (t *ddpIrStructType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

func (c *compiler) defineStructType(name string, fields []ddptypes.StructField, declarationOnly bool) *ddpIrStructType {
	structType := &ddpIrStructType{}
	structType.name = name
	structType.fieldIrTypes = mapSlice(fields, func(field ddptypes.StructField) ddpIrType {
		return c.toIrType(field.Type)
	})
	structType.fieldDDPTypes = fields
	structType.typ = c.mod.NewTypeDef(name, types.NewStruct(
		mapSlice(structType.fieldIrTypes, func(t ddpIrType) types.Type { return t.IrType() })...,
	)).(*types.StructType)
	structType.ptr = ptr(structType.typ)

	structType.freeIrFun = c.createStructFree(structType, declarationOnly)
	structType.deepCopyIrFun = c.createStructDeepCopy(structType, declarationOnly)
	structType.equalsIrFun = c.createStructEquals(structType, declarationOnly)
	return structType
}

func (c *compiler) createStructFree(structTyp *ddpIrStructType, declarationOnly bool) *ir.Func {
	structParam := ir.NewParam(structTyp.name+"_p", structTyp.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_free_%s", structTyp.typ.Name()),
		c.void.IrType(),
		structParam,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// free non-primitives
	for i, field := range structTyp.fieldIrTypes {
		c.freeNonPrimitive(c.indexStruct(structParam, int64(i)), field)
	}

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

func (c *compiler) createStructDeepCopy(structTyp *ddpIrStructType, declarationOnly bool) *ir.Func {
	ret, structParam := ir.NewParam("ret", structTyp.ptr), ir.NewParam(structTyp.name+"_p", structTyp.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_deep_copy_%s", structTyp.typ.Name()),
		c.void.IrType(),
		ret,
		structParam,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}
	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// deep-copy non-primitives
	for i, field := range structTyp.fieldIrTypes {
		dstPtr := c.indexStruct(ret, int64(i))
		if !field.IsPrimitive() {
			srcPtr := c.indexStruct(structParam, int64(i))
			c.deepCopyInto(dstPtr, srcPtr, field)
		} else {
			c.cbb.NewStore(c.loadStructField(structParam, int64(i)), dstPtr)
		}
	}

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

func (c *compiler) createStructEquals(structTyp *ddpIrStructType, declarationOnly bool) *ir.Func {
	struct1, struct2 := ir.NewParam(structTyp.name+"_1", structTyp.ptr), ir.NewParam(structTyp.name+"_2", structTyp.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_%s_equal", structTyp.typ.Name()),
		ddpbool,
		struct1,
		struct2,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// if (struct1 == struct2) return true;
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(struct1, i64), c.cbb.NewPtrToInt(struct2, i64))
	c.createIfElese(ptrs_equal, func() {
		c.cbb.NewRet(constant.True)
	},
		nil,
	)

	// compare every single field and return if one is not equal
	for i, field := range structTyp.fieldIrTypes {
		var f1, f2 value.Value
		if field.IsPrimitive() {
			f1, f2 = c.loadStructField(struct1, int64(i)), c.loadStructField(struct2, int64(i))
		} else {
			f1, f2 = c.indexStruct(struct1, int64(i)), c.indexStruct(struct2, int64(i))
		}
		is_equal := c.compare_values(f1, f2, field)
		c.createIfElese(is_equal, func() {
			c.cbb.NewRet(constant.False)
		}, nil)
	}

	c.cbb.NewRet(constant.True)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

func mapSlice[T, U any](s []T, mapper func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}

func getFieldIndex(fieldName string, typ *ddpIrStructType) int64 {
	for i, field := range typ.fieldDDPTypes {
		if field.Name == fieldName {
			return int64(i)
		}
	}
	return -1
}
