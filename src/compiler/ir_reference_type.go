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
	return llvm.Value{}
}

func (t *ddpIrReferenceType) DeepCopyFunc() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrReferenceType) EqualsFunc() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrReferenceType) PtrMask() [32]uint8 {
	return [32]uint8{1}
}

func (t *ddpIrReferenceType) LoadLivesAndRestores(c *compiler, v llvm.Value) ([]llvm.Value, []llvm.Value) {
	ref := c.builder().CreateLoad(c.ptr, v, "")
	// in case this is a stack reference, load the lives of the referenced value as well, as it couldn't be traced by the gc otherwise
	lives, restores := t.underlying.LoadLivesAndRestores(c, ref)
	lives, restores = append(lives, ref), append(restores, llvm.Value{})
	return lives, restores
}

func (c *compiler) defineReferenceType(t ddptypes.ReferenceType, underlying ddpIrType) *ddpIrReferenceType {
	if r, ok := c.refTypes[t]; ok {
		return r
	}

	refType := &ddpIrReferenceType{
		typ:          c.ptr,
		defaultValue: c.Null,
		ddpType:      t,
		underlying:   underlying,
		name:         strings.ReplaceAll(underlying.Name()+"_Referenz", " ", "_"),
	}

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, refType.name+"_vtable")
	vtable.SetLinkage(llvm.WeakODRLinkage) // weak_odr to combine vtables, which are equivalent in all modules, see https://llvm.org/docs/LangRef.html#linkage
	vtable.SetVisibility(llvm.DefaultVisibility)

	vtable.SetGlobalConstant(true)
	vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
		llvm.ConstInt(c.ddpint, c.getTypeSize(refType), false),
		llvm.ConstNull(c.ptr),
		llvm.ConstNull(c.ptr),
		llvm.ConstNull(c.ptr),
		c.refPtrMask,
	}))

	refType.vtable = vtable

	c.refTypes[t] = refType
	return refType
}
