package compiler

import (
	"github.com/bafto/Go-LLVM-Bindings/llvm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// implementation of ddpIrType for a ddpany
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with anys
type ddpIrAnyType struct {
	typ              types.Type         // the ir struct type
	ptr              *types.PointerType // ptr(typ)
	llType           llvm.Type
	freeIrFun        *ir.Func // the free ir func
	deepCopyIrFun    *ir.Func // the deepCopy ir func
	shallowCopyIrFun *ir.Func // the shallowCopy ir func
	performCowIrFun  *ir.Func // the performCow ir func
	equalsIrFun      *ir.Func // the equals ir func
}

var _ ddpIrType = (*ddpIrAnyType)(nil)

func (t *ddpIrAnyType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrAnyType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrAnyType) Name() string {
	return "ddpany"
}

func (*ddpIrAnyType) IsPrimitive() bool {
	return false
}

var i8_zero = newIntT(i8, 0)

func (t *ddpIrAnyType) DefaultValue() constant.Constant {
	return constant.NewStruct(t.typ.(*types.StructType),
		constant.NewNull(ptr(ddpint)),
		constant.NewNull(i8ptr),
		constant.NewArray(any_value_type,
			i8_zero, i8_zero, i8_zero, i8_zero,
			i8_zero, i8_zero, i8_zero, i8_zero,
			i8_zero, i8_zero, i8_zero, i8_zero,
			i8_zero, i8_zero, i8_zero, i8_zero),
	)
}

func (t *ddpIrAnyType) VTable() constant.Constant {
	return nil
}

func (t *ddpIrAnyType) LLVMType() llvm.Type {
	return t.llType
}

func (t *ddpIrAnyType) FreeFunc() *ir.Func {
	return t.freeIrFun
}

func (t *ddpIrAnyType) DeepCopyFunc() *ir.Func {
	return t.deepCopyIrFun
}

func (t *ddpIrAnyType) ShallowCopyFunc() *ir.Func {
	return t.shallowCopyIrFun
}

func (t *ddpIrAnyType) PerformCowFunc() *ir.Func {
	return t.performCowIrFun
}

func (t *ddpIrAnyType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

var any_value_type = types.NewArray(16, i8)

func (c *compiler) defineAnyType() *ddpIrAnyType {
	ddpany := &ddpIrAnyType{}
	// see equivalent in runtime/include/ddptypes.h
	ddpany.typ = c.mod.NewTypeDef("ddpany", types.NewStruct(
		ptr(ddpint),    // ddpint *refc;
		i8ptr,          // vtable *vtable_ptr
		any_value_type, // uint8_t value[16]
	))
	ddpany.ptr = ptr(ddpany.typ)

	ddpany.llType = llvm.StructType([]llvm.Type{
		llvm.PointerType(llvm.Int64Type(), 0),
		llvm.PointerType(llvm.Int8Type(), 0),
		llvm.ArrayType(llvm.Int8Type(), 16),
	}, false)

	// frees the given any and it's value
	ddpany.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_any", c.void.IrType(), ir.NewParam("any", ddpany.ptr))

	// places a copy of any in ret
	ddpany.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_any", c.void.IrType(), ir.NewParam("ret", ddpany.ptr), ir.NewParam("any", ddpany.ptr))
	// places a shallow copy of any in ret
	ddpany.shallowCopyIrFun = c.declareExternalRuntimeFunction("ddp_shallow_copy_any", c.void.IrType(), ir.NewParam("ret", ddpany.ptr), ir.NewParam("any", ddpany.ptr))

	ddpany.performCowIrFun = c.declareExternalRuntimeFunction("ddp_perform_cow_any", c.void.IrType(), ir.NewParam("any", ddpany.ptr))

	// compares two any
	ddpany.equalsIrFun = c.declareExternalRuntimeFunction("ddp_any_equal", ddpbool, ir.NewParam("any1", ddpany.ptr), ir.NewParam("any2", ddpany.ptr))

	return ddpany
}

// helper functions for working with big vs small anys

const (
	any_refc_index       = 0
	any_vtable_ptr_index = 1
	any_value_index      = 2
)

// loads the size from the any's vtable
func (c *compiler) loadAnyTypeSize(val value.Value) value.Value {
	vtable_ptr := c.loadStructField(val, any_vtable_ptr_index)
	// the size is the first field of the vtable, so this is valid
	size_ptr := c.cbb.NewBitCast(vtable_ptr, ptr(ddpint))
	return c.cbb.NewLoad(ddpint, size_ptr)
}

func (c *compiler) isSmallAny(val value.Value) value.Value {
	return c.cbb.NewICmp(enum.IPredSLE, c.loadAnyTypeSize(val), newInt(16))
}

// returns a pointer to the value_ptr field assuming it is a big any
func (c *compiler) indexBigAnyValuePtr(val value.Value) value.Value {
	val_arr_field_ptr := c.indexStruct(val, any_value_index)
	return c.cbb.NewBitCast(val_arr_field_ptr, ptr(i8ptr))
}

// loads value_ptr assuming any holds a big type
// returns an i8ptr
func (c *compiler) loadBigAnyValuePtr(val value.Value) value.Value {
	return c.cbb.NewLoad(i8ptr, c.indexBigAnyValuePtr(val))
}

func (c *compiler) loadSmallAnyValuePtr(val value.Value, typ types.Type) value.Value {
	return c.cbb.NewBitCast(c.indexStruct(val, any_value_index), ptr(typ))
}

// loads value assuming any holds a small type
// returns the given type
func (c *compiler) loadSmallAnyValue(val value.Value, typ types.Type) value.Value {
	val_arr_field_ptr := c.indexStruct(val, any_value_index)
	val_ptr_field_ptr := c.cbb.NewBitCast(val_arr_field_ptr, ptr(typ))
	return c.cbb.NewLoad(typ, val_ptr_field_ptr)
}

// loads a pointer to the value of the given any
// returns either value_ptr or a pointer to the value field,
// depending on wether it is a big or small any
func (c *compiler) loadAnyValuePtr(val value.Value, typ types.Type) value.Value {
	return c.createTernary(c.isSmallAny(val), func() value.Value {
		return c.loadSmallAnyValuePtr(val, typ)
	}, func() value.Value {
		return c.cbb.NewBitCast(c.loadBigAnyValuePtr(val), ptr(typ))
	})
}

// creates a new any from the given non-any
func (c *compiler) castNonAnyToAny(val value.Value, typ ddpIrType, isTemp bool, vtable value.Value) (value.Value, ddpIrType, bool) {
	if typ == c.ddpany {
		c.err("c.ddpany passed to castNonAnyToAny")
	}

	var result value.Value = c.NewAlloca(c.ddpany.IrType())
	c.cbb.NewStore(c.ddpany.DefaultValue(), result)

	result_vtable_ptr_ptr := c.indexStruct(result, any_vtable_ptr_index)

	c.cbb.NewStore(c.cbb.NewBitCast(vtable, i8ptr), result_vtable_ptr_ptr)

	is_small_any := c.isSmallAny(result)

	c.createIfElse(is_small_any, func() {
	}, func() {
		type_size := newInt(int64(c.getTypeSize(typ)))
		value_ptr := c.ddp_reallocate(constant.NewNull(i8ptr), zero, type_size)
		c.cbb.NewStore(value_ptr, c.indexBigAnyValuePtr(result))
	})

	// copy the value
	c.claimOrCopy(c.loadAnyValuePtr(result, typ.IrType()), val, typ, isTemp)
	result, _ = c.scp.addTemporary(result, c.ddpany)

	return result, c.ddpany, true
}

// checks if val (c.ddpany) holds a targetType
func (c *compiler) compareAnyType(val value.Value, vtable value.Value) value.Value {
	return c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(c.loadStructField(val, any_vtable_ptr_index), ddpint), c.cbb.NewPtrToInt(vtable, ddpint))
}
