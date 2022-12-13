/*
	This file defines functions to generate llvm-ir
	that interacts with the ddp-runtime
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// declares an external function on c.mod using
// the specified parameters, returnType and the C Calling Convention
func (c *Compiler) declareExternalRuntimeFunction(name string, returnType types.Type, params ...*ir.Param) *ir.Func {
	fun := c.mod.NewFunc(name, returnType, params...)
	fun.CallingConv = enum.CallingConvC
	fun.Linkage = enum.LinkageExternal
	return fun
}

var (
	_ddp_reallocate_irfun *ir.Func
)

// initializes external functions defined in the ddp-runtime
func (c *Compiler) initRuntimeFunctions() {
	_ddp_reallocate_irfun = c.declareExternalRuntimeFunction("_ddp_reallocate", ptr(i8), ir.NewParam("pointer", ptr(i8)), ir.NewParam("oldSize", i64), ir.NewParam("newSize", i64))
}

// helper functions to use the runtime-bindings

// calls _ddp_reallocate from the runtime
func (c *Compiler) _ddp_reallocate(pointer, oldSize, newSize value.Value) value.Value {
	return c.cbb.NewCall(_ddp_reallocate_irfun, pointer, oldSize, newSize)
}

// dynamically allocates a single value of type typ
func (c *Compiler) allocate(typ types.Type) value.Value {
	return c._ddp_reallocate(constant.NewNull(ptr(typ)), zero, c.sizeof(typ))
}

// allocates n elements of typ
func (c *Compiler) allocateArr(typ types.Type, n value.Value) value.Value {
	size := c.cbb.NewMul(n, c.sizeof(typ))
	return c._ddp_reallocate(constant.NewNull(ptr(typ)), zero, size)
}

// reallocates the pointer val which points to an array
// of oldCount elements of type typ to the newCount
func (c *Compiler) growArr(val, oldCount, newCount value.Value, typ types.Type) value.Value {
	elementSize := c.sizeof(typ)
	oldSize := c.cbb.NewMul(oldCount, elementSize)
	newSize := c.cbb.NewMul(newCount, elementSize)
	return c._ddp_reallocate(val, oldSize, newSize)
}

// calls free on the passed pointer which points to an element
// of type typ
func (c *Compiler) free(val value.Value, typ types.Type) {
	c._ddp_reallocate(val, c.sizeof(typ), zero)
}

// frees the pointer val which points to n elements of type typ
func (c *Compiler) freeArr(val, n value.Value, typ types.Type) {
	size := c.cbb.NewMul(n, c.sizeof(typ))
	c._ddp_reallocate(val, size, zero)
}
