/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// interface for the ir-representation of a ddptype
// it exposes some information that all types share
type ddpIrType interface {
	IrType() types.Type              // retreives the corresponding types.Type
	PtrType() *types.PointerType     // ptr(IrType())
	Name() string                    // name of the type
	IsPrimitive() bool               // wether the type is a primitive (ddpint, ddpfloat, ddpbool, ddpchar)
	DefaultValue() constant.Constant // returns a default value for the type
	FreeFunc() *ir.Func              // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() *ir.Func          // returns the irFunc used to create a deepCopy this type, nil if IsPrimitive == true
	EqualsFunc() *ir.Func            // returns the irFunc used to compare this type for equality, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	typ          types.Type
	ptr          *types.PointerType
	defaultValue constant.Constant
	name         string
}

var _ ddpIrType = (*ddpIrPrimitiveType)(nil)

func (t *ddpIrPrimitiveType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrPrimitiveType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrPrimitiveType) Name() string {
	return t.name
}

func (*ddpIrPrimitiveType) IsPrimitive() bool {
	return true
}

func (t *ddpIrPrimitiveType) DefaultValue() constant.Constant {
	return t.defaultValue
}

func (*ddpIrPrimitiveType) FreeFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) DeepCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) EqualsFunc() *ir.Func {
	return nil
}

func (c *Compiler) definePrimitiveType(typ types.Type, defaultValue constant.Constant, name string) *ddpIrPrimitiveType {
	primitive := &ddpIrPrimitiveType{
		typ:          typ,
		ptr:          ptr(typ),
		defaultValue: defaultValue,
		name:         name,
	}
	return primitive
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrVoidType struct{}

var _ ddpIrType = (*ddpIrPrimitiveType)(nil)

func (t *ddpIrVoidType) IrType() types.Type {
	return types.Void
}

func (t *ddpIrVoidType) PtrType() *types.PointerType {
	return nil
}

func (t *ddpIrVoidType) Name() string {
	return "void"
}

func (*ddpIrVoidType) IsPrimitive() bool {
	return true
}

func (t *ddpIrVoidType) DefaultValue() constant.Constant {
	return nil
}

func (*ddpIrVoidType) FreeFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) DeepCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) EqualsFunc() *ir.Func {
	return nil
}
