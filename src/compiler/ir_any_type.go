package compiler

import (
	"github.com/bafto/Go-LLVM-Bindings/llvm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// implementation of ddpIrType for a ddpany
// this struct is meant to be instantiaed by a Compiler
// exactly once as it declares all the runtime bindings needed
// to work with anys
type ddpIrAnyType struct {
	typ           types.Type         // the ir struct type
	ptr           *types.PointerType // ptr(typ)
	llType        llvm.Type
	freeIrFun     *ir.Func // the free ir func
	deepCopyIrFun *ir.Func // the deepCopy ir func
	equalsIrFun   *ir.Func // the equals ir func
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

func (t *ddpIrAnyType) DefaultValue() constant.Constant {
	return constant.NewStruct(t.typ.(*types.StructType),
		zero,
		constant.NewNull(i8ptr),
		constant.NewNull(i8ptr),
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

func (t *ddpIrAnyType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

func (c *compiler) defineAnyType() *ddpIrAnyType {
	ddpany := &ddpIrAnyType{}
	ddpany.typ = c.mod.NewTypeDef("ddpany", types.NewStruct(
		ddpint, // size
		i8ptr,  // pointer to vtable
		i8ptr,  // pointer to value
	))
	ddpany.ptr = ptr(ddpany.typ)

	ddpany.llType = llvm.StructType([]llvm.Type{
		c.ddpinttyp.LLVMType(),
		llvm.PointerType(llvm.Int8Type(), 0),
		llvm.PointerType(llvm.Int8Type(), 0),
	}, false)

	// frees the given any and it's value
	ddpany.freeIrFun = c.declareExternalRuntimeFunction("ddp_free_any", c.void.IrType(), ir.NewParam("any", ddpany.ptr))

	// places a copy of any in ret
	ddpany.deepCopyIrFun = c.declareExternalRuntimeFunction("ddp_deep_copy_any", c.void.IrType(), ir.NewParam("ret", ddpany.ptr), ir.NewParam("any", ddpany.ptr))

	// compares two any
	ddpany.equalsIrFun = c.declareExternalRuntimeFunction("ddp_any_equal", ddpbool, ir.NewParam("any1", ddpany.ptr), ir.NewParam("any2", ddpany.ptr))

	return ddpany
}
