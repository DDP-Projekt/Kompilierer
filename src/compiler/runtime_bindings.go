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
func (c *compiler) declareExternalRuntimeFunction(name string, variadic bool, gc bool, returnType llvm.Type, params ...llvm.Type) llvm.Value {
	fnType := llvm.FunctionType(returnType, params, variadic)
	llFn := llvm.AddFunction(c.llmod, name, fnType)
	llFn.SetFunctionCallConv(llvm.CCallConv)
	llFn.SetLinkage(llvm.ExternalLinkage)
	if gc {
		llFn.SetGC(DDP_GC_STRATEGY_NAME)
	}

	return c.insertFunction(name, nil, llFn, nil)
}

var (
	ddp_reallocate_irfun      llvm.Value
	ddp_runtime_error_irfun   llvm.Value
	utf8_string_to_char_irfun llvm.Value
	_libc_memcpy_irfun        llvm.Value
	_libc_memcmp_irfun        llvm.Value
	_libc_memmove_irfun       llvm.Value

	// llvm intrinsics
	llvm_statepoint_p0 llvm.Value
	llvm_result_p1     llvm.Value

	// reference functions
	ddp_free_gc_ref_irfun     llvm.Value
	ddp_allocate_gc_ref_irfun llvm.Value
	ddp_register_gc_root      llvm.Value
)

// initializes external functions defined in the ddp-runtime
func (c *compiler) initRuntimeFunctions() {
	ddp_reallocate_irfun = c.declareExternalRuntimeFunction(
		"ddp_reallocate",
		false,
		true,
		c.ptr,
		c.ptr,
		c.i64,
		c.i64,
	)

	ddp_runtime_error_irfun = c.declareExternalRuntimeFunction(
		"ddp_runtime_error",
		true,
		false,
		c.void,
		c.ddpint,
		c.ptr,
	)

	utf8_string_to_char_irfun = c.declareExternalRuntimeFunction(
		"utf8_string_to_char",
		false,
		false,
		c.i64,
		c.ptr,
		c.ptr,
	)

	_libc_memcpy_irfun = c.declareExternalRuntimeFunction(
		"memcpy",
		false,
		false,
		c.ptr,
		c.ptr,
		c.ptr,
		c.i64,
	)

	_libc_memcmp_irfun = c.declareExternalRuntimeFunction(
		"memcmp",
		false,
		false,
		c.ddpbool,
		c.ptr,
		c.ptr,
		c.i64,
	)

	_libc_memmove_irfun = c.declareExternalRuntimeFunction(
		"memmove",
		false,
		false,
		c.ptr,
		c.ptr,
		c.ptr,
		c.i64,
	)

	llvm_statepoint_p0 = c.declareExternalRuntimeFunction(
		"llvm.experimental.gc.statepoint.p0",
		true,
		false,
		c.token,
		c.i64,
		c.i32,
		c.ptr,
		c.i32,
		c.i32,
	)

	llvm_result_p1 = c.declareExternalRuntimeFunction(
		"llvm.experimental.gc.result.p1",
		false,
		false,
		c.ptr_gc,
		c.token,
	)
	llvm_result_p1.AddFunctionAttr(c.attr_nounwind)
	llvm_result_p1.AddFunctionAttr(c.attr_nocallback)
	llvm_result_p1.AddFunctionAttr(c.attr_nofree)
	llvm_result_p1.AddFunctionAttr(c.attr_nosync)
	llvm_result_p1.AddFunctionAttr(c.attr_willreturn)
	llvm_result_p1.AddFunctionAttr(c.attr_memory_none)

	ddp_allocate_gc_ref_irfun = c.declareExternalRuntimeFunction(
		"ddp_allocate_gc_ref",
		false,
		true,
		c.ptr_gc,
		c.ptr, // vtable
	)

	ddp_free_gc_ref_irfun = c.declareExternalRuntimeFunction(
		"ddp_free_gc_ref",
		false,
		true,
		c.void,
		c.ptr_gc,
	)

	ddp_register_gc_root = c.declareExternalRuntimeFunction(
		"ddp_register_gc_root",
		false,
		false,
		c.void,
		c.ptr,
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

func (c *compiler) allocateGCRef(vtable llvm.Value) llvm.Value {
	tok := c.builder().createCall(llvm_statepoint_p0, c.zero, c.zero32, ddp_allocate_gc_ref_irfun, c.one32, c.zero32, vtable, c.zero32, c.zero32)
	// TODO: add attribute correctly
	tok.AddCallSiteAttribute(3, c.attr_elementtype_ptr_gc_ptr)

	return c.builder().createCall(llvm_result_p1, tok)
}
