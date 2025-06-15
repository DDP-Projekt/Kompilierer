/*
This file defines functions to generate llvm-ir
that interacts with the ddp-runtime
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// declares an external function on c.mod using
// the specified parameters, returnType and the C Calling Convention
func (c *compiler) declareExternalRuntimeFunction(name string, variadic bool, returnType llvm.Type, params ...llvm.Type) llvm.Value {
	fnType := llvm.FunctionType(returnType, params, variadic)
	llFn := llvm.AddFunction(c.llmod, name, fnType)
	llFn.SetFunctionCallConv(llvm.CCallConv)
	llFn.SetLinkage(llvm.ExternalLinkage)

	return c.insertFunction(name, nil, llFn, nil)
}

var (
	ddp_reallocate_irfun      llvm.Value
	ddp_runtime_error_irfun   llvm.Value
	utf8_string_to_char_irfun llvm.Value
	_libc_memcpy_irfun        llvm.Value
	_libc_memcmp_irfun        llvm.Value
	_libc_memmove_irfun       llvm.Value
)

// initializes external functions defined in the ddp-runtime
func (c *compiler) initRuntimeFunctions() {
	ddp_reallocate_irfun = c.declareExternalRuntimeFunction(
		"ddp_reallocate",
		false,
		c.ptr,
		c.ptr,
		c.i64,
		c.i64,
	)

	ddp_runtime_error_irfun = c.declareExternalRuntimeFunction(
		"ddp_runtime_error",
		true,
		c.void,
		c.ddpint,
		c.ptr,
	)

	utf8_string_to_char_irfun = c.declareExternalRuntimeFunction(
		"utf8_string_to_char",
		false,
		c.i64,
		c.ptr,
		c.ptr,
	)

	_libc_memcpy_irfun = c.declareExternalRuntimeFunction(
		"memcpy",
		false,
		c.ptr,
		c.ptr,
		c.ptr,
		c.i64,
	)

	_libc_memcmp_irfun = c.declareExternalRuntimeFunction(
		"memcmp",
		false,
		c.ddpbool,
		c.ptr,
		c.ptr,
		c.i64,
	)

	_libc_memmove_irfun = c.declareExternalRuntimeFunction(
		"memmove",
		false,
		c.ptr,
		c.ptr,
		c.ptr,
		c.i64,
	)
}

// helper functions to use the runtime-bindings

func (c *compiler) runtime_error(exit_code int, fmt llvm.Value, args ...llvm.Value) {
	strPtr := llvm.ConstInBoundsGEP(fmt.GlobalValueType(), fmt, []llvm.Value{c.zero, c.zero})
	args = append([]llvm.Value{c.newInt(int64(exit_code)), c.builder().CreateBitCast(strPtr, c.ptr, "")}, args...)
	c.builder().createCall(ddp_runtime_error_irfun, args...)
	c.builder().CreateUnreachable()
}

func (c *compiler) out_of_bounds_error(line, column, index, len llvm.Value) {
	c.runtime_error(1, c.out_of_bounds_error_string, line, column, index, len)
}

// calls ddp_reallocate from the runtime
func (c *compiler) ddp_reallocate(pointer, oldSize, newSize llvm.Value) llvm.Value {
	return c.builder().createCall(ddp_reallocate_irfun, pointer, oldSize, newSize)
}

// dynamically allocates a single value of type typ
func (c *compiler) allocate(typ llvm.Type) llvm.Value {
	return c.ddp_reallocate(c.Null, c.zero, c.sizeof(typ))
}

// allocates n elements of elementType
func (c *compiler) allocateArr(elementType llvm.Type, n llvm.Value) llvm.Value {
	size := c.builder().CreateMul(n, c.sizeof(elementType), "")
	return c.ddp_reallocate(c.Null, c.zero, size)
}

// reallocates the pointer val which points to an array
// of oldCount elements of type typ to the newCount
func (c *compiler) growArr(elementType llvm.Type, ptr, oldCount, newCount llvm.Value) llvm.Value {
	elementSize := c.sizeof(elementType)
	oldSize := c.builder().CreateMul(oldCount, elementSize, "")
	newSize := c.builder().CreateMul(newCount, elementSize, "")
	return c.ddp_reallocate(ptr, oldSize, newSize)
}

// calls free on the passed pointer
func (c *compiler) free(typ llvm.Type, ptr llvm.Value) {
	c.ddp_reallocate(ptr, c.sizeof(typ), c.zero)
}

// frees the pointer val which points to n elements
func (c *compiler) freeArr(elementType llvm.Type, ptr, n llvm.Value) {
	size := c.builder().CreateMul(n, c.sizeof(elementType), "")
	c.ddp_reallocate(ptr, size, c.zero)
}

// wraps the memcpy function from libc
// dest and src must be pointer types, n is the size to copy in bytes
func (c *compiler) memcpy(dest, src, n llvm.Value) llvm.Value {
	return c.builder().createCall(_libc_memcpy_irfun, dest, src, n)
}

// wraps memcpy for a array, where n is the length of the array in src
func (c *compiler) memcpyArr(elementType llvm.Type, dest, src, n llvm.Value) llvm.Value {
	size := c.builder().CreateMul(n, c.sizeof(elementType), "")
	return c.memcpy(dest, src, size)
}

// wraps the memmove function from libc
// dest and src must be pointer types, n is the size to copy in bytes
func (c *compiler) memmove(dest, src, n llvm.Value) llvm.Value {
	return c.builder().createCall(_libc_memmove_irfun, dest, src, n)
}

// wraps memmove for a array, where n is the length of the array in src
func (c *compiler) memmoveArr(elementType llvm.Type, dest, src, n llvm.Value) llvm.Value {
	size := c.builder().CreateMul(n, c.sizeof(elementType), "")
	return c.memmove(dest, src, size)
}

func (c *compiler) memcmp(buf1, buf2, size llvm.Value) llvm.Value {
	return c.builder().createCall(_libc_memcmp_irfun, buf1, buf2, size)
}
