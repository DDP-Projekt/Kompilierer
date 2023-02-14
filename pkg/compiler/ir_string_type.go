/*
This file defines types and functions to work
with the llir representation of ddptypes
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type ddpIrStringType struct {
	typ                types.Type
	fromConstantsIrFun *ir.Func // the fromConstans ir func
	freeIrFun          *ir.Func // the free ir func
	deepCopyIrFun      *ir.Func // the deepCopy ir func
	equalsIrFun        *ir.Func // the equals ir func
}

var _ ddpIrType = (*ddpIrStringType)(nil)

func (t *ddpIrStringType) IrType() types.Type {
	return t.typ
}

func (*ddpIrStringType) IsPrimitive() bool {
	return false
}

func (t *ddpIrStringType) DefaultValue() value.Value {
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

	// declare all the external functions to work with strings

	// creates a ddpstring from a string literal and returns a pointer to it
	// the caller is responsible for calling increment_ref_count on this pointer
	ddpstring.fromConstantsIrFun = c.declareExternalRuntimeFunction("_ddp_string_from_constant", ddpstrptr, ir.NewParam("str", ptr(i8)))

	// frees the given string
	ddpstring.freeIrFun = c.declareExternalRuntimeFunction("_ddp_free_string", void, ir.NewParam("str", ddpstrptr))

	// returns a copy of the passed string as a new pointer
	// the caller is responsible for calling increment_ref_count on this pointer
	ddpstring.deepCopyIrFun = c.declareExternalRuntimeFunction("_ddp_deep_copy_string", ddpstrptr, ir.NewParam("str", ddpstrptr))

	// checks wether the two strings are equal
	ddpstring.equalsIrFun = c.declareExternalRuntimeFunction("_ddp_string_equal", ddpbool, ir.NewParam("str1", ddpstrptr), ir.NewParam("str2", ddpstrptr))

	return ddpstring
}
