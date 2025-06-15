/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// implementation of ddpIrType for a ddpstring
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with strings
type ddpIrStringType struct {
	typ                    llvm.Type // the ir struct type
	defaultValue           llvm.Value
	vtable                 llvm.Value
	fromConstantsIrFun     llvm.Value // the fromConstans ir func
	freeIrFun              llvm.Value // the free ir func
	deepCopyIrFun          llvm.Value // the deepCopy ir func
	equalsIrFun            llvm.Value // the equals ir func
	lengthIrFun            llvm.Value // the string_length ir func
	indexIrFun             llvm.Value // the string_index ir func
	replaceCharIrFun       llvm.Value // the replace_char_in_string ir func
	sliceIrFun             llvm.Value // the clice ir func
	str_str_concat_IrFunc  llvm.Value // the str_str_verkettet ir func
	str_char_concat_IrFunc llvm.Value // the str_char_verkettet ir func
	char_str_concat_IrFunc llvm.Value // the char_str_verkettet ir func
	int_to_string_IrFun    llvm.Value // the int_to_string ir func
	float_to_string_IrFun  llvm.Value // the float_to_string ir func
	byte_to_string_IrFun   llvm.Value // the byte_to_string ir func
	bool_to_string_IrFun   llvm.Value // the bool_to_string ir func
	char_to_string_IrFun   llvm.Value // the char_to_string ir func
}

var _ ddpIrType = (*ddpIrStringType)(nil)

func (t *ddpIrStringType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrStringType) Name() string {
	return "ddpstring"
}

func (*ddpIrStringType) IsPrimitive() bool {
	return false
}

func (t *ddpIrStringType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrStringType) VTable() llvm.Value {
	return t.vtable
}

func (t *ddpIrStringType) FreeFunc() llvm.Value {
	return t.freeIrFun
}

func (t *ddpIrStringType) DeepCopyFunc() llvm.Value {
	return t.deepCopyIrFun
}

func (t *ddpIrStringType) EqualsFunc() llvm.Value {
	return t.equalsIrFun
}

const (
	string_str_field_index = 0
	string_cap_field_index = 1
)

func (c *compiler) defineStringType(declarationOnly bool) *ddpIrStringType {
	ddpstring := &ddpIrStringType{}

	ddpstring.typ = c.llctx.StructType([]llvm.Type{c.ptr, c.ddpint}, false)

	// declare all the external functions to work with strings

	// allocates a buffer for ret and copies str into it
	ddpstring.fromConstantsIrFun = c.declareExternalRuntimeFunction("ddp_string_from_constant", false, c.void, c.ptr, c.ptr)

	// frees the given string
	ddpstring.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_string", false, c.void, c.ptr)

	// places a copy of str in ret allocating new buffers
	ddpstring.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_string", false, c.void, c.ptr, c.ptr)

	// checks wether the two strings are equal
	ddpstring.equalsIrFun = c.declareExternalRuntimeFunction("ddp_string_equal", false, c.ddpbool, c.ptr, c.ptr)

	// returns the number of utf8 runes in str
	ddpstring.lengthIrFun = c.declareExternalRuntimeFunction("ddp_string_length", false, c.ddpint, c.ptr)

	// returns the utf8-char at index
	ddpstring.indexIrFun = c.declareExternalRuntimeFunction("ddp_string_index", false, c.ddpchar, c.ptr, c.ddpint)

	// replaces the utf8-char at the index with ch
	ddpstring.replaceCharIrFun = c.declareExternalRuntimeFunction("ddp_replace_char_in_string", false, c.void, c.ptr, c.ddpchar, c.ddpint)

	ddpstring.sliceIrFun = c.declareExternalRuntimeFunction("ddp_string_slice", false, c.void, c.ptr, c.ptr, c.ddpint, c.ddpint)

	ddpstring.str_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_string_verkettet", false, c.void, c.ptr, c.ptr, c.ptr)
	ddpstring.char_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_char_string_verkettet", false, c.void, c.ptr, c.ddpchar, c.ptr)
	ddpstring.str_char_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_char_verkettet", false, c.void, c.ptr, c.ptr, c.ddpchar)

	ddpstring.int_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_int_to_string", false, c.void, c.ptr, c.ddpint)
	ddpstring.float_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_float_to_string", false, c.void, c.ptr, c.ddpfloat)
	ddpstring.byte_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_byte_to_string", false, c.void, c.ptr, c.ddpbyte)
	ddpstring.bool_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_bool_to_string", false, c.void, c.ptr, c.ddpbool)
	ddpstring.char_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_char_to_string", false, c.void, c.ptr, c.ddpchar)

	vtable := llvm.AddGlobal(c.llmod, c.vtable_type, "ddpstring_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetGlobalConstant(true)
		vtable.SetInitializer(llvm.ConstNamedStruct(c.vtable_type, []llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(ddpstring), false),
			ddpstring.freeIrFun,
			ddpstring.deepCopyIrFun,
			ddpstring.equalsIrFun,
		}))
	}

	ddpstring.vtable = vtable
	ddpstring.defaultValue = llvm.ConstNull(ddpstring.typ)

	return ddpstring
}
