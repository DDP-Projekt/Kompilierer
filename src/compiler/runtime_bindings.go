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

	// reference functions
	ddp_allocate_gc_ref_irfun llvm.Value
	ddp_register_gc_root      llvm.Value

	ddp_do_nothing_ptr_gc llvm.Value
	ddp_do_nothing_ptr    llvm.Value
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

	_libc_memmove_irfun = c.declareExternalRuntimeFunction(
		"memmove",
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

	ddp_allocate_gc_ref_irfun = c.declareExternalRuntimeFunction(
		"ddp_allocate_gc_ref",
		false,
		true,
		c.ptr,
		c.ptr,    // vtable
		c.ddpint, // arrlen
	)

	ddp_register_gc_root = c.declareExternalRuntimeFunction(
		"ddp_register_gc_root",
		false,
		false,
		c.void,
		c.ptr,
	)

	ddp_do_nothing_ptr_gc_builder := c.createBuilder("ddp_do_nothing_ptr_gc", llvm.FunctionType(c.void, []llvm.Type{c.ptr}, false), nil, []string{"arg"}, nil, nil, true, false)
	ddp_do_nothing_ptr_gc_builder.CreateRet(llvm.Value{})

	ddp_do_nothing_ptr_gc = ddp_do_nothing_ptr_gc_builder.llFn
	ddp_do_nothing_ptr_gc.SetLinkage(llvm.LinkOnceAnyLinkage)

	ddp_do_nothing_ptr_builder := c.createBuilder("ddp_do_nothing_ptr", llvm.FunctionType(c.void, []llvm.Type{c.ptr}, false), nil, []string{"arg"}, nil, nil, true, false)
	ddp_do_nothing_ptr_builder.CreateRet(llvm.Value{})

	ddp_do_nothing_ptr = ddp_do_nothing_ptr_builder.llFn
	ddp_do_nothing_ptr.SetLinkage(llvm.LinkOnceAnyLinkage)
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

// wraps the memmoveGC function from libc
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
	return c.builder().createCall(ddp_allocate_gc_ref_irfun, vtable, c.one)
}

func (c *compiler) allocateGCRefArray(vtable llvm.Value, n llvm.Value) llvm.Value {
	return c.builder().createCall(ddp_allocate_gc_ref_irfun, vtable, n)
}
