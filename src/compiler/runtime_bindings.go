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
func (c *compiler) declareExternalRuntimeFunction(name string, returnType types.Type, params ...*ir.Param) *ir.Func {
	fun := c.mod.NewFunc(name, returnType, params...)
	fun.CallingConv = enum.CallingConvC
	fun.Linkage = enum.LinkageExternal
	c.insertFunction(name, nil, fun)
	return fun
}

var (
	ddp_reallocate_irfun      *ir.Func
	ddp_runtime_error_irfun   *ir.Func
	utf8_string_to_char_irfun *ir.Func
	_libc_memcpy_irfun        *ir.Func
	_libc_memcmp_irfun        *ir.Func
	_libc_memmove_irfun       *ir.Func
)

// initializes external functions defined in the ddp-runtime
func (c *compiler) initRuntimeFunctions() {
	ddp_reallocate_irfun = c.declareExternalRuntimeFunction(
		"ddp_reallocate",
		i8ptr,
		ir.NewParam("pointer", i8ptr),
		ir.NewParam("oldSize", i64),
		ir.NewParam("newSize", i64),
	)

	ddp_runtime_error_irfun = c.declareExternalRuntimeFunction(
		"ddp_runtime_error",
		c.void.IrType(),
		ir.NewParam("exit_code", ddpint),
		ir.NewParam("fmt", i8ptr),
	)
	ddp_runtime_error_irfun.Sig.Variadic = true

	utf8_string_to_char_irfun = c.declareExternalRuntimeFunction(
		"utf8_string_to_char",
		i64,
		ir.NewParam("str", i8ptr),
		ir.NewParam("out", ptr(i32)),
	)

	_libc_memcpy_irfun = c.declareExternalRuntimeFunction(
		"memcpy",
		i8ptr,
		ir.NewParam("dest", i8ptr),
		ir.NewParam("src", i8ptr),
		ir.NewParam("n", i64),
	)

	_libc_memcmp_irfun = c.declareExternalRuntimeFunction(
		"memcmp",
		ddpbool,
		ir.NewParam("buf1", i8ptr),
		ir.NewParam("buf2", i8ptr),
		ir.NewParam("size", i64),
	)

	_libc_memmove_irfun = c.declareExternalRuntimeFunction(
		"memmove",
		i8ptr,
		ir.NewParam("dest", i8ptr),
		ir.NewParam("src", i8ptr),
		ir.NewParam("n", i64),
	)
}

// helper functions to use the runtime-bindings

func (c *compiler) runtime_error(exit_code int, fmt value.Value, args ...value.Value) {
	args = append([]value.Value{newInt(int64(exit_code)), c.cbb.NewBitCast(fmt, i8ptr)}, args...)
	c.cbb.NewCall(ddp_runtime_error_irfun, args...)
	c.cbb.NewUnreachable()
}

func (c *compiler) out_of_bounds_error(line, column, index, len value.Value) {
	c.runtime_error(1, c.out_of_bounds_error_string, line, column, index, len)
}

// calls ddp_reallocate from the runtime
func (c *compiler) ddp_reallocate(pointer, oldSize, newSize value.Value) value.Value {
	pointer_param := c.cbb.NewBitCast(pointer, i8ptr)
	return c.cbb.NewBitCast(c.cbb.NewCall(ddp_reallocate_irfun, pointer_param, oldSize, newSize), pointer.Type())
}

// dynamically allocates a single value of type typ
func (c *compiler) allocate(typ types.Type) value.Value {
	return c.ddp_reallocate(constant.NewNull(ptr(typ)), zero, c.sizeof(typ))
}

// allocates n elements of elementType
func (c *compiler) allocateArr(elementType types.Type, n value.Value) value.Value {
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c.ddp_reallocate(constant.NewNull(ptr(elementType)), zero, size)
}

// reallocates the pointer val which points to an array
// of oldCount elements of type typ to the newCount
func (c *compiler) growArr(ptr, oldCount, newCount value.Value) value.Value {
	elementType := getPointeeType(ptr)
	elementSize := c.sizeof(elementType)
	oldSize := c.cbb.NewMul(oldCount, elementSize)
	newSize := c.cbb.NewMul(newCount, elementSize)
	return c.ddp_reallocate(ptr, oldSize, newSize)
}

// calls free on the passed pointer
func (c *compiler) free(ptr value.Value) {
	elementType := getPointeeType(ptr)
	c.ddp_reallocate(ptr, c.sizeof(elementType), zero)
}

// frees the pointer val which points to n elements
func (c *compiler) freeArr(ptr, n value.Value) {
	elementType := getPointeeType(ptr)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	c.ddp_reallocate(ptr, size, zero)
}

// wraps the memcpy function from libc
// dest and src must be pointer types, n is the size to copy in bytes
func (c *compiler) memcpy(dest, src, n value.Value) value.Value {
	dest_param, src_param := c.cbb.NewBitCast(dest, i8ptr), c.cbb.NewBitCast(src, i8ptr)
	return c.cbb.NewCall(_libc_memcpy_irfun, dest_param, src_param, n)
}

// wraps memcpy for a array, where n is the length of the array in src
func (c *compiler) memcpyArr(dest, src, n value.Value) value.Value {
	elementType := getPointeeType(src)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c.memcpy(dest, src, size)
}

// wraps the memmove function from libc
// dest and src must be pointer types, n is the size to copy in bytes
func (c *compiler) memmove(dest, src, n value.Value) value.Value {
	dest_param, src_param := c.cbb.NewBitCast(dest, i8ptr), c.cbb.NewBitCast(src, i8ptr)
	return c.cbb.NewCall(_libc_memmove_irfun, dest_param, src_param, n)
}

// wraps memmove for a array, where n is the length of the array in src
func (c *compiler) memmoveArr(dest, src, n value.Value) value.Value {
	elementType := getPointeeType(src)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c.memmove(dest, src, size)
}

func (c *compiler) memcmp(buf1, buf2, size value.Value) value.Value {
	buf1_param, buf2_param := c.cbb.NewBitCast(buf1, i8ptr), c.cbb.NewBitCast(buf2, i8ptr)
	return c.cbb.NewCall(_libc_memcmp_irfun, buf1_param, buf2_param, size)
}
