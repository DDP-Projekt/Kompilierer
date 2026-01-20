/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

// interface for the ir-representation of a ddptype
// it exposes some information that all types share
type ddpIrType interface {
	LLType() llvm.Type        // returns the llvm type
	DDPType() ddptypes.Type   // returns the ddp type this represents
	Name() string             // name of the type
	TriviallyCopyable() bool  // wether the type is a primitive (ddpint, ddpfloat, ddpbool, ddpchar)
	DefaultValue() llvm.Value // returns a default value for the type
	VTable() llvm.Value       // returns a pointer to the vtable of this type
	FreeFunc() llvm.Value     // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() llvm.Value // returns the irFunc used to create a deepCopy this type, nil if IsPrimitive == true
	EqualsFunc() llvm.Value   // returns the irFunc used to compare this type for equality, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	llType       llvm.Type
	ddpType      ddptypes.Type
	defaultValue llvm.Value
	vtable       llvm.Value
	funcNull     llvm.Value
	name         string
}

var _ ddpIrType = (*ddpIrPrimitiveType)(nil)

func (t *ddpIrPrimitiveType) LLType() llvm.Type {
	return t.llType
}

func (t *ddpIrPrimitiveType) DDPType() ddptypes.Type {
	return t.ddpType
}

func (t *ddpIrPrimitiveType) Name() string {
	return t.name
}

func (*ddpIrPrimitiveType) TriviallyCopyable() bool {
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

func (c *compiler) definePrimitiveType(ddptyp ddptypes.Type, typ llvm.Type, defaultValue llvm.Value, name string) *ddpIrPrimitiveType {
	primitive := &ddpIrPrimitiveType{
		llType:       typ,
		ddpType:      ddptyp,
		defaultValue: defaultValue,
		funcNull:     llvm.ConstNull(c.ptr),
		name:         name,
	}

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, name+"_vtable")
	vtable.SetLinkage(llvm.WeakODRLinkage) // weak_odr to combine vtables, which are equivalent in all modules, see https://llvm.org/docs/LangRef.html#linkage
	vtable.SetVisibility(llvm.DefaultVisibility)

	vtable.SetGlobalConstant(true)
	vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
		llvm.ConstInt(c.ddpint, c.getTypeSize(primitive), false),
		llvm.ConstNull(c.ptr),
		llvm.ConstNull(c.ptr),
		llvm.ConstNull(c.ptr),
	}))

	primitive.vtable = vtable

	c.defineReferenceType(ddptypes.ReferenceType{Type: ddptyp}, primitive)
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

func (t *ddpIrVoidType) DDPType() ddptypes.Type {
	return ddptypes.VoidType{}
}

func (t *ddpIrVoidType) PtrType() llvm.Type {
	return llvm.Type{}
}

func (t *ddpIrVoidType) Name() string {
	return "void"
}

func (*ddpIrVoidType) TriviallyCopyable() bool {
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
