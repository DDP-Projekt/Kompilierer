/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
)

// implementation of ddpIrType for a ddpstring
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with strings
type ddpIrStringType struct {
	llType                 llvm.Type // the ir struct type
	ptr                    llvm.Type // ptr(typ)
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
	return t.llType
}

func (t *ddpIrStringType) PtrType() llvm.Type {
	return t.ptr
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

	ddpstring.llType = c.llctx.StructType([]llvm.Type{c.i8ptr, c.ddpint}, false)
	ddpstring.ptr = c.ptr(ddpstring.llType)

	// declare all the external functions to work with strings

	// allocates a buffer for ret and copies str into it
	ddpstring.fromConstantsIrFun = c.declareExternalRuntimeFunction("ddp_string_from_constant", false, c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", i8ptr))

	// frees the given string
	ddpstring.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_string", c.voidtyp.IrType(), ir.NewParam("str", ddpstring.ptr))

	// places a copy of str in ret allocating new buffers
	ddpstring.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr))

	// checks wether the two strings are equal
	ddpstring.equalsIrFun = c.declareExternalRuntimeFunction("ddp_string_equal", ddpbool, ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))

	// returns the number of utf8 runes in str
	ddpstring.lengthIrFun = c.declareExternalRuntimeFunction("ddp_string_length", ddpint, ir.NewParam("str", ddpstring.ptr))

	// returns the utf8-char at index
	ddpstring.indexIrFun = c.declareExternalRuntimeFunction("ddp_string_index", ddpchar, ir.NewParam("str", ddpstring.ptr), ir.NewParam("index", ddpint))

	// replaces the utf8-char at the index with ch
	ddpstring.replaceCharIrFun = c.declareExternalRuntimeFunction("ddp_replace_char_in_string", c.voidtyp.IrType(), ir.NewParam("str", ddpstring.ptr), ir.NewParam("ch", ddpchar), ir.NewParam("index", ddpint))

	ddpstring.sliceIrFun = c.declareExternalRuntimeFunction("ddp_string_slice", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("index1", ddpint), ir.NewParam("index2", ddpint))

	ddpstring.str_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_string_verkettet", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))
	ddpstring.char_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_char_string_verkettet", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar), ir.NewParam("str", ddpstring.ptr))
	ddpstring.str_char_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_char_verkettet", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("c", ddpchar))

	ddpstring.int_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_int_to_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("i", ddpint))
	ddpstring.float_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_float_to_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("f", ddpfloat))
	ddpstring.byte_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_byte_to_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("b", ddpbyte))
	ddpstring.bool_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_bool_to_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("b", ddpbool))
	ddpstring.char_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_char_to_string", c.voidtyp.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar))

	// see equivalent in runtime/include/ddptypes.h
	vtable_type := c.mod.NewTypeDef(ddpstring.Name()+"_vtable_type", types.NewStruct(
		ddpint, // ddpint type_size
		ptr(types.NewFunc(c.voidtyp.IrType(), ddpstring.ptr)),                   // free_func_ptr free_func
		ptr(types.NewFunc(c.voidtyp.IrType(), ddpstring.ptr, ddpstring.ptr)),    // deep_copy_func_ptr deep_copy_func
		ptr(types.NewFunc(c.ddpbooltyp.IrType(), ddpstring.ptr, ddpstring.ptr)), // equal_func_ptr equal_func
	))

	var vtable *ir.Global
	if declarationOnly {
		vtable = c.mod.NewGlobal(ddpstring.Name()+"_vtable", ptr(vtable_type))
		vtable.Linkage = enum.LinkageExternal
		vtable.Visibility = enum.VisibilityDefault
	} else {
		vtable = c.mod.NewGlobalDef(ddpstring.Name()+"_vtable", constant.NewStruct(vtable_type.(*types.StructType),
			newInt(int64(c.getTypeSize(ddpstring))),
			ddpstring.freeIrFun,
			ddpstring.deepCopyIrFun,
			ddpstring.equalsIrFun,
		))
	}

	ddpstring.vtable = vtable

	return ddpstring
}
