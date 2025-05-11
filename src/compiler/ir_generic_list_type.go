package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

type ddpIrGenericListType struct {
	typ    types.Type
	ptr    *types.PointerType
	llType llvm.Type
}

var _ ddpIrType = (*ddpIrGenericListType)(nil)

func (t *ddpIrGenericListType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrGenericListType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrGenericListType) Name() string {
	return t.typ.Name()
}

func (*ddpIrGenericListType) IsPrimitive() bool {
	return false
}

func (t *ddpIrGenericListType) DefaultValue() constant.Constant {
	return constant.NewStruct(t.typ.(*types.StructType),
		constant.NewNull(i8ptr),
		zero,
		zero,
	)
}

func (t *ddpIrGenericListType) VTable() constant.Constant {
	return nil
}

func (t *ddpIrGenericListType) LLVMType() llvm.Type {
	return t.llType
}

func (t *ddpIrGenericListType) FreeFunc() *ir.Func {
	return nil
}

func (t *ddpIrGenericListType) DeepCopyFunc() *ir.Func {
	return nil
}

func (t *ddpIrGenericListType) EqualsFunc() *ir.Func {
	return nil
}

func (c *compiler) createGenericListType() *ddpIrGenericListType {
	list := &ddpIrGenericListType{}
	list.typ = c.mod.NewTypeDef("ddpgenericlist", types.NewStruct(
		i8ptr,  // underlying array
		ddpint, // length
		ddpint, // capacity
	))
	list.ptr = ptr(list.typ)

	list.llType = llvm.StructType([]llvm.Type{
		llvm.PointerType(llvm.Int8Type(), 0),
		c.ddpinttyp.LLVMType(),
		c.ddpinttyp.LLVMType(),
	}, false)

	return list
}
