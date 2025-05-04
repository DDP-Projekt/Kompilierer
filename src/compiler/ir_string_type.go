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
	typ                    types.Type         // the ir struct type
	ptr                    *types.PointerType // ptr(typ)
	vtable                 *ir.Global
	llType                 llvm.Type
	fromConstantsIrFun     *ir.Func // the fromConstans ir func
	freeIrFun              *ir.Func // the free ir func
	deepCopyIrFun          *ir.Func // the deepCopy ir func
	equalsIrFun            *ir.Func // the equals ir func
	lengthIrFun            *ir.Func // the string_length ir func
	indexIrFun             *ir.Func // the string_index ir func
	replaceCharIrFun       *ir.Func // the replace_char_in_string ir func
	sliceIrFun             *ir.Func // the clice ir func
	str_str_concat_IrFunc  *ir.Func // the str_str_verkettet ir func
	str_char_concat_IrFunc *ir.Func // the str_char_verkettet ir func
	char_str_concat_IrFunc *ir.Func // the char_str_verkettet ir func
	int_to_string_IrFun    *ir.Func // the int_to_string ir func
	float_to_string_IrFun  *ir.Func // the float_to_string ir func
	byte_to_string_IrFun   *ir.Func // the byte_to_string ir func
	bool_to_string_IrFun   *ir.Func // the bool_to_string ir func
	char_to_string_IrFun   *ir.Func // the char_to_string ir func
}

var _ ddpIrType = (*ddpIrStringType)(nil)

func (t *ddpIrStringType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrStringType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrStringType) Name() string {
	return "ddpstring"
}

func (*ddpIrStringType) IsPrimitive() bool {
	return false
}

func (t *ddpIrStringType) DefaultValue() constant.Constant {
	return constant.NewStruct(t.typ.(*types.StructType),
		constant.NewNull(i8ptr),
		zero,
	)
}

func (t *ddpIrStringType) VTable() constant.Constant {
	return t.vtable
}

func (t *ddpIrStringType) LLVMType() llvm.Type {
	return t.llType
}

func (t *ddpIrStringType) FreeFunc() *ir.Func {
	return t.freeIrFun
}

func (t *ddpIrStringType) DeepCopyFunc() *ir.Func {
	return t.deepCopyIrFun
}

func (t *ddpIrStringType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

const (
	string_str_field_index = 0
	string_cap_field_index = 1
)

func (c *compiler) defineStringType(declarationOnly bool) *ddpIrStringType {
	ddpstring := &ddpIrStringType{}
	ddpstring.typ = c.mod.NewTypeDef("ddpstring", types.NewStruct(
		i8ptr,  // char* str
		ddpint, // ddpint cap;
	))
	ddpstring.ptr = ptr(ddpstring.typ)

	ddpstring.llType = llvm.StructType([]llvm.Type{
		llvm.PointerType(llvm.Int8Type(), 0),
		llvm.Int64Type(),
	}, false)

	// declare all the external functions to work with strings

	// allocates a buffer for ret and copies str into it
	ddpstring.fromConstantsIrFun = c.declareExternalRuntimeFunction("ddp_string_from_constant", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", i8ptr))

	// frees the given string
	ddpstring.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_string", c.void.IrType(), ir.NewParam("str", ddpstring.ptr))

	// places a copy of str in ret allocating new buffers
	ddpstring.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr))

	// checks wether the two strings are equal
	ddpstring.equalsIrFun = c.declareExternalRuntimeFunction("ddp_string_equal", ddpbool, ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))

	// returns the number of utf8 runes in str
	ddpstring.lengthIrFun = c.declareExternalRuntimeFunction("ddp_string_length", ddpint, ir.NewParam("str", ddpstring.ptr))

	// returns the utf8-char at index
	ddpstring.indexIrFun = c.declareExternalRuntimeFunction("ddp_string_index", ddpchar, ir.NewParam("str", ddpstring.ptr), ir.NewParam("index", ddpint))

	// replaces the utf8-char at the index with ch
	ddpstring.replaceCharIrFun = c.declareExternalRuntimeFunction("ddp_replace_char_in_string", c.void.IrType(), ir.NewParam("str", ddpstring.ptr), ir.NewParam("ch", ddpchar), ir.NewParam("index", ddpint))

	ddpstring.sliceIrFun = c.declareExternalRuntimeFunction("ddp_string_slice", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("index1", ddpint), ir.NewParam("index2", ddpint))

	ddpstring.str_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_string_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))
	ddpstring.char_str_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_char_string_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar), ir.NewParam("str", ddpstring.ptr))
	ddpstring.str_char_concat_IrFunc = c.declareExternalRuntimeFunction("ddp_string_char_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("c", ddpchar))

	ddpstring.int_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_int_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("i", ddpint))
	ddpstring.float_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_float_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("f", ddpfloat))
	ddpstring.byte_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_byte_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("b", ddpbyte))
	ddpstring.bool_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_bool_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("b", ddpbool))
	ddpstring.char_to_string_IrFun = c.declareExternalRuntimeFunction("ddp_char_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar))

	// see equivalent in runtime/include/ddptypes.h
	vtable_type := c.mod.NewTypeDef(ddpstring.Name()+"_vtable_type", types.NewStruct(
		ddpint, // ddpint type_size
		ptr(types.NewFunc(c.void.IrType(), ddpstring.ptr)),                      // free_func_ptr free_func
		ptr(types.NewFunc(c.void.IrType(), ddpstring.ptr, ddpstring.ptr)),       // deep_copy_func_ptr deep_copy_func
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
