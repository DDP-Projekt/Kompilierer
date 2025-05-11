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
	LLType() llvm.Type         // returns the llvm type
	PtrType() llvm.Type        // ptr(LLType())
	Name() string              // name of the type
	IsPrimitive() bool         // wether the type is a primitive (ddpint, ddpfloat, ddpbool, ddpchar)
	DefaultValue() llvm.Value  // returns a default value for the type
	VTable() llvm.Value        // returns a pointer to the vtable of this type
	FreeFunc() *llvm.Value     // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() *llvm.Value // returns the irFunc used to create a deepCopy this type, nil if IsPrimitive == true
	EqualsFunc() *llvm.Value   // returns the irFunc used to compare this type for equality, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	llType       llvm.Type
	ptr          llvm.Type
	defaultValue llvm.Value
	vtable       llvm.Value
	name         string
}

var _ ddpIrType = (*ddpIrPrimitiveType)(nil)

func (t *ddpIrPrimitiveType) LLType() llvm.Type {
	return t.llType
}

func (t *ddpIrPrimitiveType) PtrType() llvm.Type {
	return t.ptr
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

func (*ddpIrPrimitiveType) FreeFunc() *llvm.Value {
	return nil
}

func (*ddpIrPrimitiveType) DeepCopyFunc() *llvm.Value {
	return nil
}

func (*ddpIrPrimitiveType) EqualsFunc() *llvm.Value {
	return nil
}

func (c *compiler) definePrimitiveType(typ llvm.Type, defaultValue llvm.Value, name string, declarationOnly bool) *ddpIrPrimitiveType {
	typ_ptr := llvm.PointerType(typ, 0)

	// the single field is a dummy pointer to make the struct non-zero sized
	vtable_type := c.llctx.StructType([]llvm.Type{
		c.ddpint,
		c.ptr(llvm.FunctionType(c.void, []llvm.Type{typ_ptr}, false)),
		c.ptr(llvm.FunctionType(c.void, []llvm.Type{typ_ptr, typ_ptr}, false)),
		c.ptr(llvm.FunctionType(c.ddpbool, []llvm.Type{typ_ptr, typ_ptr}, false)),
	}, false,
	)

	primitive := &ddpIrPrimitiveType{
		llType:       typ,
		ptr:          typ_ptr,
		defaultValue: defaultValue,
		name:         name,
	}

	vtable := llvm.AddGlobal(c.llmod, c.ptr(vtable_type), name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetInitializer(llvm.ConstStruct([]llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(primitive), false),
			llvm.ConstNull(vtable_type.StructElementTypes()[0]),
			llvm.ConstNull(vtable_type.StructElementTypes()[1]),
			llvm.ConstNull(vtable_type.StructElementTypes()[2]),
		}, false))
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

func (*ddpIrVoidType) FreeFunc() *llvm.Value {
	return nil
}

func (*ddpIrVoidType) DeepCopyFunc() *llvm.Value {
	return nil
}

func (*ddpIrVoidType) EqualsFunc() *llvm.Value {
	return nil
}

func (c *compiler) defineVoidType() *ddpIrVoidType {
	return &ddpIrVoidType{rawType: c.void}
}
