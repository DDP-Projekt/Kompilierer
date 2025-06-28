package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

type ddpIrGenericListType struct {
	typ          llvm.Type
	defaultValue llvm.Value
}

var _ ddpIrType = (*ddpIrGenericListType)(nil)

func (t *ddpIrGenericListType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrGenericListType) Name() string {
	return "ddpgenericlist"
}

func (*ddpIrGenericListType) IsPrimitive() bool {
	return false
}

func (t *ddpIrGenericListType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrGenericListType) VTable() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrGenericListType) FreeFunc() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrGenericListType) DeepCopyFunc() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrGenericListType) EqualsFunc() llvm.Value {
	return llvm.Value{}
}

func (c *compiler) createGenericListType() *ddpIrGenericListType {
	list := &ddpIrGenericListType{}
	list.typ = c.llctx.StructType([]llvm.Type{c.ptr, c.ddpint, c.ddpint}, false)

	return list
}
