/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/bafto/Go-LLVM-Bindings/llvm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
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
	VTable() constant.Constant       // returns a pointer to the vtable of this type
	LLVMType() llvm.Type             // returns the llvm type
	FreeFunc() *ir.Func              // returns the irFunc used to free this type, nil if IsPrimitive == true
	DeepCopyFunc() *ir.Func          // returns the irFunc used to create a deepCopy of this type, nil if IsPrimitive == true
	ShallowCopyFunc() *ir.Func       // returns the irFunc used to create a shallowCopy of this type, nil if IsPrimitive == true
	PerformCowFunc() *ir.Func        // only present for strings, lists and anys
	EqualsFunc() *ir.Func            // returns the irFunc used to compare this type for equality, nil if IsPrimitive == true
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrPrimitiveType struct {
	typ          types.Type
	ptr          *types.PointerType
	defaultValue constant.Constant
	vtable       *ir.Global
	name         string
	llType       llvm.Type
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

func (t *ddpIrPrimitiveType) VTable() constant.Constant {
	return t.vtable
}

func (t *ddpIrPrimitiveType) LLVMType() llvm.Type {
	return t.llType
}

func (*ddpIrPrimitiveType) FreeFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) DeepCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) ShallowCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) PerformCowFunc() *ir.Func {
	return nil
}

func (*ddpIrPrimitiveType) EqualsFunc() *ir.Func {
	return nil
}

func (c *compiler) definePrimitiveType(typ types.Type, defaultValue constant.Constant, llType llvm.Type, name string, declarationOnly bool) *ddpIrPrimitiveType {
	typ_ptr := ptr(typ)

	// the single field is a dummy pointer to make the struct non-zero sized
	vtable_type := c.mod.NewTypeDef(name+"_vtable_type", types.NewStruct(
		ptr(types.NewFunc(types.Void, typ_ptr)),
		ptr(types.NewFunc(types.Void, typ_ptr, typ_ptr)),
		ptr(types.NewFunc(types.Void, typ_ptr, typ_ptr)),
		ptr(types.NewFunc(ddpbool, typ_ptr, typ_ptr)),
	))

	var vtable *ir.Global
	if declarationOnly {
		vtable = c.mod.NewGlobal(name+"_vtable", ptr(vtable_type))
		vtable.Linkage = enum.LinkageExternal
		vtable.Visibility = enum.VisibilityDefault
	} else {
		vtable = c.mod.NewGlobalDef(name+"_vtable", constant.NewStruct(vtable_type.(*types.StructType),
			constant.NewNull(vtable_type.(*types.StructType).Fields[0].(*types.PointerType)),
			constant.NewNull(vtable_type.(*types.StructType).Fields[1].(*types.PointerType)),
			constant.NewNull(vtable_type.(*types.StructType).Fields[2].(*types.PointerType)),
			constant.NewNull(vtable_type.(*types.StructType).Fields[3].(*types.PointerType)),
		))
	}

	primitive := &ddpIrPrimitiveType{
		typ:          typ,
		ptr:          typ_ptr,
		defaultValue: defaultValue,
		vtable:       vtable,
		name:         name,
		llType:       llType,
	}
	return primitive
}

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrVoidType struct{}

var _ ddpIrType = (*ddpIrVoidType)(nil)

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

func (t *ddpIrVoidType) VTable() constant.Constant {
	return nil
}

func (t *ddpIrVoidType) LLVMType() llvm.Type {
	return llvm.VoidType()
}

func (*ddpIrVoidType) FreeFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) DeepCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) ShallowCopyFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) PerformCowFunc() *ir.Func {
	return nil
}

func (*ddpIrVoidType) EqualsFunc() *ir.Func {
	return nil
}
