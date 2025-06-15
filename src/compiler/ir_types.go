/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// interface for the ir-representation of a ddptype
// it exposes some information that all types share
type ddpIrType interface {
	LLType() llvm.Type        // returns the llvm type
	Name() string             // name of the type
	IsPrimitive() bool        // wether the type is a primitive (ddpint, ddpfloat, ddpbool, ddpchar)
	DefaultValue() llvm.Value // returns a default value for the type
	VTable() llvm.Value       // returns a pointer to the vtable of this type
	FreeFunc() llvm.Value     // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() llvm.Value // returns the irFunc used to create a deepCopy this type, nil if IsPrimitive == true
	EqualsFunc() llvm.Value   // returns the irFunc used to compare this type for equality, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	llType       llvm.Type
	defaultValue llvm.Value
	vtable       llvm.Value
	funcNull     llvm.Value
	name         string
}

var _ ddpIrType = (*ddpIrPrimitiveType)(nil)

func (t *ddpIrPrimitiveType) LLType() llvm.Type {
	return t.llType
}

func (t *ddpIrPrimitiveType) Name() string {
	return t.name
}

func (*ddpIrPrimitiveType) IsPrimitive() bool {
	return true
}

func (t *ddpIrPrimitiveType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrPrimitiveType) VTable() llvm.Value {
	return t.vtable
}

func (t *ddpIrPrimitiveType) FreeFunc() llvm.Value {
	return t.funcNull
}

func (t *ddpIrPrimitiveType) DeepCopyFunc() llvm.Value {
	return t.funcNull
}

func (t *ddpIrPrimitiveType) EqualsFunc() llvm.Value {
	return t.funcNull
}

func (c *compiler) definePrimitiveType(typ llvm.Type, defaultValue llvm.Value, name string, declarationOnly bool) *ddpIrPrimitiveType {
	primitive := &ddpIrPrimitiveType{
		llType:       typ,
		defaultValue: defaultValue,
		funcNull:     llvm.ConstNull(c.ptr),
		name:         name,
	}

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetGlobalConstant(true)
		vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(primitive), false),
			llvm.ConstNull(c.ptr),
			llvm.ConstNull(c.ptr),
			llvm.ConstNull(c.ptr),
		}))
	}

	primitive.vtable = vtable

	return primitive
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrVoidType struct {
	rawType llvm.Type
}

var _ ddpIrType = (*ddpIrVoidType)(nil)

func (t *ddpIrVoidType) LLType() llvm.Type {
	return t.rawType
}

func (t *ddpIrVoidType) PtrType() llvm.Type {
	return llvm.Type{}
}

func (t *ddpIrVoidType) Name() string {
	return "void"
}

func (*ddpIrVoidType) IsPrimitive() bool {
	return true
}

func (t *ddpIrVoidType) DefaultValue() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrVoidType) VTable() llvm.Value {
	return llvm.Value{}
}

func (*ddpIrVoidType) FreeFunc() llvm.Value {
	return llvm.Value{}
}

func (*ddpIrVoidType) DeepCopyFunc() llvm.Value {
	return llvm.Value{}
}

func (*ddpIrVoidType) EqualsFunc() llvm.Value {
	return llvm.Value{}
}

func (c *compiler) defineVoidType() *ddpIrVoidType {
	return &ddpIrVoidType{rawType: c.void}
}
