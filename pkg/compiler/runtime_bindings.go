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
	c.insertFunction(name, nil, fun)
	return fun
}

var (
	_ddp_reallocate_irfun *ir.Func
	_libc_memcpy_irfun    *ir.Func
)

// initializes external functions defined in the ddp-runtime
func (c *Compiler) initRuntimeFunctions() {
	_ddp_reallocate_irfun = c.declareExternalRuntimeFunction(
		"_ddp_reallocate",
		ptr(i8),
		ir.NewParam("pointer", ptr(i8)),
		ir.NewParam("oldSize", i64),
		ir.NewParam("newSize", i64),
	)

	_libc_memcpy_irfun = c.declareExternalRuntimeFunction(
		"memcpy",
		ptr(i8),
		ir.NewParam("dest", ptr(i8)),
		ir.NewParam("src", ptr(i8)),
		ir.NewParam("n", i64),
	)
}

// helper functions to use the runtime-bindings

// calls _ddp_reallocate from the runtime
func (c *Compiler) _ddp_reallocate(pointer, oldSize, newSize value.Value) value.Value {
	return c.cbb.NewBitCast(c.cbb.NewCall(_ddp_reallocate_irfun, pointer, oldSize, newSize), pointer.Type())
}

// dynamically allocates a single value of type typ
func (c *Compiler) allocate(typ types.Type) value.Value {
	return c._ddp_reallocate(constant.NewNull(ptr(typ)), zero, c.sizeof(typ))
}

// allocates n elements of elementType
func (c *Compiler) allocateArr(elementType types.Type, n value.Value) value.Value {
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c._ddp_reallocate(constant.NewNull(ptr(elementType)), zero, size)
}

// reallocates the pointer val which points to an array
// of oldCount elements of type typ to the newCount
func (c *Compiler) growArr(ptr, oldCount, newCount value.Value) value.Value {
	elementType := getPointeeType(ptr)
	elementSize := c.sizeof(elementType)
	oldSize := c.cbb.NewMul(oldCount, elementSize)
	newSize := c.cbb.NewMul(newCount, elementSize)
	return c._ddp_reallocate(ptr, oldSize, newSize)
}

// calls free on the passed pointer
func (c *Compiler) free(ptr value.Value) {
	elementType := getPointeeType(ptr)
	c._ddp_reallocate(ptr, c.sizeof(elementType), zero)
}

// frees the pointer val which points to n elements
func (c *Compiler) freeArr(ptr, n value.Value) {
	elementType := getPointeeType(ptr)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	c._ddp_reallocate(ptr, size, zero)
}

// wraps the memcpy function from libc
// dest and src must be pointer types, n is the size to copy in bytes
func (c *Compiler) memcpy(dest, src, n value.Value) value.Value {
	return c.cbb.NewCall(_libc_memcpy_irfun, dest, src, n)
}

// wraps memcpy for a array, where n is the length of the array in src
func (c *Compiler) memcpyArr(dest, src, n value.Value) value.Value {
	elementType := getPointeeType(src)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c.memcpy(dest, src, size)
}
