package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// implementation of ddpIrType for a ddpany
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with anys
type ddpIrAnyType struct {
	typ           llvm.Type // the ir struct type
	defaultValue  llvm.Value
	freeIrFun     llvm.Value // the free ir func
	deepCopyIrFun llvm.Value // the deepCopy ir func
	equalsIrFun   llvm.Value // the equals ir func
}

var _ ddpIrType = (*ddpIrAnyType)(nil)

func (t *ddpIrAnyType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrAnyType) Name() string {
	return "ddpany"
}

func (*ddpIrAnyType) IsPrimitive() bool {
	return false
}

func (t *ddpIrAnyType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrAnyType) VTable() llvm.Value {
	return llvm.Value{}
}

func (t *ddpIrAnyType) FreeFunc() llvm.Value {
	return t.freeIrFun
}

func (t *ddpIrAnyType) DeepCopyFunc() llvm.Value {
	return t.deepCopyIrFun
}

func (t *ddpIrAnyType) EqualsFunc() llvm.Value {
	return t.equalsIrFun
}

func (c *compiler) defineAnyType() *ddpIrAnyType {
	ddpany := &ddpIrAnyType{}
	ddpany.typ = c.llctx.StructType([]llvm.Type{c.ptr, llvm.ArrayType(c.i8, 16)}, false)

	// frees the given any and it's value
	ddpany.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_any", false, c.void, c.ptr)

	// places a copy of any in ret
	ddpany.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_any", false, c.void, c.ptr, c.ptr)

	// compares two any
	ddpany.equalsIrFun = c.declareExternalRuntimeFunction("ddp_any_equal", false, c.ddpbool, c.ptr, c.ptr)

	ddpany.defaultValue = llvm.ConstNull(ddpany.typ)

	return ddpany
}

// helper functions for working with big vs small anys

const (
	any_vtable_ptr_index = 0
	any_value_index      = 1
)

// loads the size from the any's vtable
func (c *compiler) loadAnyTypeSize(val llvm.Value) llvm.Value {
	vtable_ptr := c.loadStructField(c.ddpany.typ, val, any_vtable_ptr_index)
	// the size is the first field of the vtable, so this is valid
	return c.builder().CreateLoad(c.ddpint, vtable_ptr, "")
}

func (c *compiler) isSmallAny(val llvm.Value) llvm.Value {
	return c.builder().CreateICmp(llvm.IntSLE, c.loadAnyTypeSize(val), c.newInt(16), "")
}

// returns a pointer to the value_ptr field assuming it is a big any
func (c *compiler) indexBigAnyValuePtr(val llvm.Value) llvm.Value {
	return c.indexStruct(c.ddpany.typ, val, any_value_index)
}

// loads value_ptr assuming any holds a big type
// returns an i8ptr
func (c *compiler) loadBigAnyValuePtr(val llvm.Value) llvm.Value {
	return c.builder().CreateLoad(c.ptr, c.indexBigAnyValuePtr(val), "")
}

func (c *compiler) loadSmallAnyValuePtr(val llvm.Value) llvm.Value {
	return c.builder().CreateBitCast(c.indexStruct(c.ddpany.typ, val, any_value_index), c.ptr, "")
}

// loads value assuming any holds a small type
// returns the given type
func (c *compiler) loadSmallAnyValue(val llvm.Value, typ llvm.Type) llvm.Value {
	val_arr_field_ptr := c.indexStruct(c.ddpany.typ, val, any_value_index)
	val_ptr_field_ptr := c.builder().CreateBitCast(val_arr_field_ptr, c.ptr, "")
	return c.builder().CreateLoad(typ, val_ptr_field_ptr, "")
}

// loads a pointer to the value of the given any
// returns either value_ptr or a pointer to the value field,
// depending on wether it is a big or small any
func (c *compiler) loadAnyValuePtr(val llvm.Value, typ llvm.Type) llvm.Value {
	return c.createTernary(c.ptr, c.isSmallAny(val), func() llvm.Value {
		return c.loadSmallAnyValuePtr(val)
	}, func() llvm.Value {
		return c.builder().CreateBitCast(c.loadBigAnyValuePtr(val), c.ptr, "")
	})
}

// creates a new any from the given non-any
func (c *compiler) castNonAnyToAny(val llvm.Value, typ ddpIrType, isTemp bool, vtable llvm.Value) (llvm.Value, ddpIrType, bool) {
	if typ == c.ddpany {
		c.err("c.ddpany passed to castNonAnyToAny")
	}

	result := c.NewAlloca(c.ddpany.LLType())

	result_vtable_ptr_ptr := c.indexStruct(c.ddpany.typ, result, any_vtable_ptr_index)

	c.builder().CreateStore(c.builder().CreateBitCast(vtable, c.ptr, ""), result_vtable_ptr_ptr)

	is_small_any := c.isSmallAny(result)

	c.createIfElse(is_small_any, func() {
	}, func() {
		type_size := c.newInt(int64(c.getTypeSize(typ)))
		value_ptr := c.ddp_reallocate(c.Null, c.zero, type_size)
		c.builder().CreateStore(value_ptr, c.indexBigAnyValuePtr(result))
	})

	// copy the value
	c.claimOrCopy(c.loadAnyValuePtr(result, typ.LLType()), val, typ, isTemp)
	result, _ = c.scp.addTemporary(result, c.ddpany)

	return result, c.ddpany, true
}

// checks if val (c.ddpany) holds a targetType
func (c *compiler) compareAnyType(val llvm.Value, vtable llvm.Value) llvm.Value {
	return c.builder().CreateICmp(llvm.IntEQ, c.loadStructField(c.ddpany.typ, val, any_vtable_ptr_index), vtable, "")
}
