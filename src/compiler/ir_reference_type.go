package compiler

import (
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

// implementation of ddpIrType for a ddpstring
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with strings
type ddpIrReferenceType struct {
	typ          llvm.Type // the ir struct type
	defaultValue llvm.Value
	vtable       llvm.Value
	ddpType      ddptypes.ReferenceType
	underlying   ddpIrType
	name         string
}

var _ ddpIrType = (*ddpIrReferenceType)(nil)

func (t *ddpIrReferenceType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrReferenceType) DDPType() ddptypes.Type {
	return t.ddpType
}

func (t *ddpIrReferenceType) Name() string {
	return t.name
}

func (*ddpIrReferenceType) TriviallyCopyable() bool {
	return true
}

func (t *ddpIrReferenceType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrReferenceType) VTable() llvm.Value {
	return t.vtable
}

func (t *ddpIrReferenceType) FreeFunc() llvm.Value {
	return ddp_free_ref_type_irfun
}

func (t *ddpIrReferenceType) DeepCopyFunc() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrReferenceType) EqualsFunc() llvm.Value {
	return llvm.Value{}
}

func (c *compiler) defineReferenceType(t ddptypes.ReferenceType, underlying ddpIrType, declarationOnly bool) *ddpIrReferenceType {
	if r, ok := c.refTypes[t]; ok {
		return r
	}

	refType := &ddpIrReferenceType{
		typ:          c.ptr,
		defaultValue: c.Null,
		ddpType:      t,
		underlying:   underlying,
		name:         strings.ReplaceAll(t.String(), " ", "_"),
	}

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, refType.name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetGlobalConstant(true)
		vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(refType), false),
			ddp_free_ref_type_irfun, // TODO: this should probably not be null
			llvm.ConstNull(c.ptr),
			llvm.ConstNull(c.ptr),
		}))
	}

	refType.vtable = vtable

	c.refTypes[t] = refType
	return refType
}
