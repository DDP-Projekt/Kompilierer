/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// interface for the ir-representation of a ddptype
type ddpIrType interface {
	IrType() types.Type        // retreives the corresponding types.Type
	IsPrimitive() bool         // wether the type is a primitive (ddpint, ddpfloat, ddpbool, ddpchar)
	DefaultValue() value.Value // returns a default value for the type
	FreeFunc() *ir.Func        // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() *ir.Func    // returns the irFunc used to create a deepCopy this type, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	typ          types.Type
	defaultValue value.Value
}

func (t *ddpIrPrimitiveType) IrType() types.Type {
	return t.typ
}

func (*ddpIrPrimitiveType) IsPrimitive() bool {
	return true
}

func (t *ddpIrPrimitiveType) DefaultValue() value.Value {
	return t.defaultValue
}

func (*ddpIrPrimitiveType) FreeFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) DeepCopyFunc() *ir.Func {
	return nil
}

func (c *Compiler) definePrimitiveType(typ types.Type, defaultValue value.Value) *ddpIrPrimitiveType {
	primitive := &ddpIrPrimitiveType{
		typ:          typ,
		defaultValue: defaultValue,
	}
	return primitive
}
