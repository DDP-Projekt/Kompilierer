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
	ddp_reallocate_irfun    *ir.Func
	ddp_runtime_error_irfun *ir.Func
	_libc_memcpy_irfun      *ir.Func
	_libc_memcmp_irfun      *ir.Func
	_libc_memmove_irfun     *ir.Func
)

// initializes external functions defined in the ddp-runtime
func (c *compiler) initRuntimeFunctions() {
	ddp_reallocate_irfun = c.declareExternalRuntimeFunction(
		"ddp_reallocate",
		ptr(i8),
		ir.NewParam("pointer", ptr(i8)),
		ir.NewParam("oldSize", i64),
		ir.NewParam("newSize", i64),
	)

	ddp_runtime_error_irfun = c.declareExternalRuntimeFunction(
		"ddp_runtime_error",
		c.void.IrType(),
		ir.NewParam("exit_code", ddpint),
		ir.NewParam("fmt", ptr(i8)),
	)
	ddp_runtime_error_irfun.Sig.Variadic = true

	_libc_memcpy_irfun = c.declareExternalRuntimeFunction(
		"memcpy",
		ptr(i8),
		ir.NewParam("dest", ptr(i8)),
		ir.NewParam("src", ptr(i8)),
		ir.NewParam("n", i64),
	)

	_libc_memcmp_irfun = c.declareExternalRuntimeFunction(
		"memcmp",
		ddpbool,
		ir.NewParam("buf1", ptr(i8)),
		ir.NewParam("buf2", ptr(i8)),
		ir.NewParam("size", i64),
	)

	_libc_memmove_irfun = c.declareExternalRuntimeFunction(
		"memmove",
		ptr(i8),
		ir.NewParam("dest", ptr(i8)),
		ir.NewParam("src", ptr(i8)),
		ir.NewParam("n", i64),
	)
}

// helper functions to use the runtime-bindings

func (c *compiler) runtime_error(exit_code, fmt value.Value, args ...value.Value) {
	args = append([]value.Value{exit_code, c.cbb.NewBitCast(fmt, ptr(i8))}, args...)
	c.cbb.NewCall(ddp_runtime_error_irfun, args...)
	c.cbb.NewUnreachable()
}

var out_of_bounds_error_string *ir.Global

func (c *compiler) out_of_bounds_error(index, len value.Value) {
	if out_of_bounds_error_string == nil {
		out_of_bounds_error_string = c.mod.NewGlobalDef("", constant.NewCharArrayFromString("Index außerhalb der Listen Länge (Index war %ld, Listen Länge war %ld)\n"))
		out_of_bounds_error_string.Visibility = enum.VisibilityHidden
		out_of_bounds_error_string.Immutable = true
	}
	c.runtime_error(newInt(1), out_of_bounds_error_string, index, len)
}

// calls ddp_reallocate from the runtime
func (c *compiler) ddp_reallocate(pointer, oldSize, newSize value.Value) value.Value {
	pointer_param := c.cbb.NewBitCast(pointer, ptr(i8))
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
	dest_param, src_param := c.cbb.NewBitCast(dest, ptr(i8)), c.cbb.NewBitCast(src, ptr(i8))
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
	dest_param, src_param := c.cbb.NewBitCast(dest, ptr(i8)), c.cbb.NewBitCast(src, ptr(i8))
	return c.cbb.NewCall(_libc_memmove_irfun, dest_param, src_param, n)
}

// wraps memmove for a array, where n is the length of the array in src
func (c *compiler) memmoveArr(dest, src, n value.Value) value.Value {
	elementType := getPointeeType(src)
	size := c.cbb.NewMul(n, c.sizeof(elementType))
	return c.memmove(dest, src, size)
}

func (c *compiler) memcmp(buf1, buf2, size value.Value) value.Value {
	buf1_param, buf2_param := c.cbb.NewBitCast(buf1, ptr(i8)), c.cbb.NewBitCast(buf2, ptr(i8))
	return c.cbb.NewCall(_libc_memcmp_irfun, buf1_param, buf2_param, size)
}
