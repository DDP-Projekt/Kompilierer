/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// implementation of ddpIrType for a ddpstring
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with strings
type ddpIrStringType struct {
	typ                    types.Type         // the ir struct type
	ptr                    *types.PointerType // ptr(typ)
	fromConstantsIrFun     *ir.Func           // the fromConstans ir func
	freeIrFun              *ir.Func           // the free ir func
	deepCopyIrFun          *ir.Func           // the deepCopy ir func
	equalsIrFun            *ir.Func           // the equals ir func
	lengthIrFun            *ir.Func           // the string_length ir func
	indexIrFun             *ir.Func           // the string_index ir func
	replaceCharIrFun       *ir.Func           // the replace_char_in_string ir func
	sliceIrFun             *ir.Func           // the clice ir func
	str_str_concat_IrFunc  *ir.Func           // the str_str_verkettet ir func
	str_char_concat_IrFunc *ir.Func           // the str_char_verkettet ir func
	char_str_concat_IrFunc *ir.Func           // the char_str_verkettet ir func
	int_to_string_IrFun    *ir.Func           // the int_to_string ir func
	float_to_string_IrFun  *ir.Func           // the float_to_string ir func
	bool_to_string_IrFun   *ir.Func           // the bool_to_string ir func
	char_to_string_IrFun   *ir.Func           // the char_to_string ir func
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
		constant.NewNull(ptr(i8)),
		zero,
	)
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

func (c *Compiler) defineStringType() *ddpIrStringType {
	ddpstring := &ddpIrStringType{}
	ddpstring.typ = c.mod.NewTypeDef("ddpstring", types.NewStruct(
		ptr(i8), // char* str
		ddpint,  // ddpint cap;
	))
	ddpstring.ptr = ptr(ddpstring.typ)

	// declare all the external functions to work with strings

	// allocates a buffer for ret and copies str into it
	ddpstring.fromConstantsIrFun = c.declareExternalRuntimeFunction("_ddp_string_from_constant", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ptr(i8)))

	// frees the given string
	ddpstring.freeIrFun = c.declareExternalRuntimeFunction("_ddp_free_string", c.void.IrType(), ir.NewParam("str", ddpstring.ptr))

	// places a copy of str in ret allocating new buffers
	ddpstring.deepCopyIrFun = c.declareExternalRuntimeFunction("_ddp_deep_copy_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr))

	// checks wether the two strings are equal
	ddpstring.equalsIrFun = c.declareExternalRuntimeFunction("_ddp_string_equal", ddpbool, ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))

	// returns the number of utf8 runes in str
	ddpstring.lengthIrFun = c.declareExternalRuntimeFunction("_ddp_string_length", ddpint, ir.NewParam("str", ddpstring.ptr))

	// returns the utf8-char at index
	ddpstring.indexIrFun = c.declareExternalRuntimeFunction("_ddp_string_index", ddpchar, ir.NewParam("str", ddpstring.ptr), ir.NewParam("index", ddpint))

	// replaces the utf8-char at the index with ch
	ddpstring.replaceCharIrFun = c.declareExternalRuntimeFunction("_ddp_replace_char_in_string", c.void.IrType(), ir.NewParam("str", ddpstring.ptr), ir.NewParam("ch", ddpchar), ir.NewParam("index", ddpint))

	ddpstring.sliceIrFun = c.declareExternalRuntimeFunction("_ddp_string_slice", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("index1", ddpint), ir.NewParam("index2", ddpint))

	ddpstring.str_str_concat_IrFunc = c.declareExternalRuntimeFunction("_ddp_string_string_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str1", ddpstring.ptr), ir.NewParam("str2", ddpstring.ptr))
	ddpstring.char_str_concat_IrFunc = c.declareExternalRuntimeFunction("_ddp_char_string_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar), ir.NewParam("str", ddpstring.ptr))
	ddpstring.str_char_concat_IrFunc = c.declareExternalRuntimeFunction("_ddp_string_char_verkettet", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("str", ddpstring.ptr), ir.NewParam("c", ddpchar))

	ddpstring.int_to_string_IrFun = c.declareExternalRuntimeFunction("_ddp_int_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("i", ddpint))
	ddpstring.float_to_string_IrFun = c.declareExternalRuntimeFunction("_ddp_float_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("f", ddpfloat))
	ddpstring.bool_to_string_IrFun = c.declareExternalRuntimeFunction("_ddp_bool_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("b", ddpbool))
	ddpstring.char_to_string_IrFun = c.declareExternalRuntimeFunction("_ddp_char_to_string", c.void.IrType(), ir.NewParam("ret", ddpstring.ptr), ir.NewParam("c", ddpchar))

	return ddpstring
}
