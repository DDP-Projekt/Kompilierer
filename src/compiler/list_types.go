/*
This file defines functions to define
and work with lists.

The "calling convention" for all those functions inside the llir
is a bit special for non-primitive types (strings, lists, structs):

All functions that should return a non-primitive type, take a pointer
to it as first argument and fill that argument.
This means that the caller must allocate the return values properly
beforehand.
*/
package compiler

import (
	"fmt"

	"github.com/bafto/Go-LLVM-Bindings/llvm"
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// holds the ir-definitions of a ddp-list-type
// for each list-type such a struct
// should be instantiated by a Compiler
// exactly once
type ddpIrListType struct {
	typ                         types.Type         // the typedef
	ptr                         *types.PointerType // ptr(typ)
	elementType                 ddpIrType          // the ir-type of the list elements
	vtable                      *ir.Global
	llType                      llvm.Type
	fromConstantsIrFun          *ir.Func // the fromConstans ir func
	freeIrFun                   *ir.Func // the free ir func
	deepCopyIrFun               *ir.Func // the deepCopy ir func
	shallowCopyIrFun            *ir.Func // the shallowCopy ir func
	performCowIrFun             *ir.Func // the performCow ir func
	equalsIrFun                 *ir.Func // the equals ir func
	sliceIrFun                  *ir.Func // the clice ir func
	list_list_concat_IrFunc     *ir.Func // the list_list_verkettet ir func
	list_scalar_concat_IrFunc   *ir.Func // the list_scalar_verkettet ir func
	scalar_scalar_concat_IrFunc *ir.Func // the scalar_scalar_verkettet ir func
	scalar_list_concat_IrFunc   *ir.Func // the scalar_list_verkettet ir func
}

var _ ddpIrType = (*ddpIrListType)(nil)

func (t *ddpIrListType) IrType() types.Type {
	return t.typ
}

func (t *ddpIrListType) PtrType() *types.PointerType {
	return t.ptr
}

func (t *ddpIrListType) Name() string {
	return t.typ.Name()
}

func (*ddpIrListType) IsPrimitive() bool {
	return false
}

func (t *ddpIrListType) DefaultValue() constant.Constant {
	return constant.NewStruct(t.typ.(*types.StructType),
		constant.NewNull(ptr(ddpint)),
		constant.NewNull(t.elementType.PtrType()),
		zero,
		zero,
	)
}

func (t *ddpIrListType) VTable() constant.Constant {
	return t.vtable
}

func (t *ddpIrListType) LLVMType() llvm.Type {
	return t.llType
}

func (t *ddpIrListType) FreeFunc() *ir.Func {
	return t.freeIrFun
}

func (t *ddpIrListType) DeepCopyFunc() *ir.Func {
	return t.deepCopyIrFun
}

func (t *ddpIrListType) ShallowCopyFunc() *ir.Func {
	return t.shallowCopyIrFun
}

func (t *ddpIrListType) PerformCowFunc() *ir.Func {
	return t.performCowIrFun
}

func (t *ddpIrListType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

// defines the struct of a list type
// and all the necessery functions
// from the given elementType and name
// if declarationOnly is true, all functions are only extern declarations
// C-Equivalent:
/*
	struct listtypename {
		ElementType* arr;
		ddpint len;
		ddpint cap;
	}
*/
func (c *compiler) createListType(name string, elementType ddpIrType, declarationOnly bool) *ddpIrListType {
	list := &ddpIrListType{}
	list.elementType = elementType
	list.typ = c.mod.NewTypeDef(name, types.NewStruct(
		ptr(ddpint),                // refc
		list.elementType.PtrType(), // underlying array
		ddpint,                     // length
		ddpint,                     // capacity
	))
	list.ptr = ptr(list.typ)

	list.llType = llvm.StructType([]llvm.Type{
		llvm.PointerType(llvm.Int64Type(), 0),
		llvm.PointerType(elementType.LLVMType(), 0),
		c.ddpinttyp.LLVMType(),
		c.ddpinttyp.LLVMType(),
	}, false)

	list.fromConstantsIrFun = c.createListFromConstants(list, declarationOnly)
	list.freeIrFun = c.createListFree(list, declarationOnly)
	list.deepCopyIrFun = c.createListDeepCopy(list, declarationOnly)
	list.shallowCopyIrFun = c.createListShallowCopy(list, declarationOnly)
	list.performCowIrFun = c.createListPerformCow(list, declarationOnly)
	list.equalsIrFun = c.createListEquals(list, declarationOnly)
	list.sliceIrFun = c.createListSlice(list, declarationOnly)

	list.list_list_concat_IrFunc,
		list.list_scalar_concat_IrFunc,
		list.scalar_scalar_concat_IrFunc,
		list.scalar_list_concat_IrFunc = c.createListConcats(list, declarationOnly)

	// see equivalent in runtime/include/ddptypes.h
	vtable_type := c.mod.NewTypeDef(name+"_vtable_type", types.NewStruct(
		ddpint, // ddpint type_size
		ptr(types.NewFunc(c.void.IrType(), list.ptr)),                 // free_func_ptr free_func
		ptr(types.NewFunc(c.void.IrType(), list.ptr, list.ptr)),       // deep_copy_func_ptr deep_copy_func
		ptr(types.NewFunc(c.void.IrType(), list.ptr, list.ptr)),       // shallow_copy_func_ptr shallow_copy_func
		ptr(types.NewFunc(c.ddpbooltyp.IrType(), list.ptr, list.ptr)), // equal_func_ptr equal_func
	))

	var vtable *ir.Global
	if declarationOnly {
		vtable = c.mod.NewGlobal(name+"_vtable", ptr(vtable_type))
		vtable.Linkage = enum.LinkageExternal
		vtable.Visibility = enum.VisibilityDefault
	} else {
		vtable = c.mod.NewGlobalDef(name+"_vtable", constant.NewStruct(vtable_type.(*types.StructType),
			newInt(32),
			list.freeIrFun,
			list.deepCopyIrFun,
			list.shallowCopyIrFun,
			list.equalsIrFun,
		))
	}

	list.vtable = vtable

	return list
}

const (
	list_refc_field_index = 0
	list_arr_field_index  = 1
	list_len_field_index  = 2
	list_cap_field_index  = 3
)

/*
defines the ddp_x_from_constants function for a listType

ddp_x_from_constants allocates an array big enough to hold count
elements and stores it inside ret->arr.
It also sets ret->len and ret->cap accordingly

signature:
c.void.IrType() ddp_x_from_constants(x* ret, ddpint count)
*/
func (c *compiler) createListFromConstants(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	// declare the parameters to use them as values
	ret, count := ir.NewParam("ret", listType.ptr), ir.NewParam("count", ddpint)
	// declare the function
	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_%s_from_constants", listType.typ.Name()),
		c.void.IrType(),
		ret,
		count,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	arrType := listType.elementType.PtrType()

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	// start block
	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	cond := c.cbb.NewICmp(enum.IPredSGT, count, zero) // count > 0

	// count > 0 ? allocate(sizeof(t) * count) : NULL
	result := c.createTernary(cond,
		func() value.Value { return c.allocateArr(listType.elementType.IrType(), count) },
		func() value.Value { return constant.NewNull(arrType) },
	)

	// get pointers to the struct fields
	retArr, retLen, retCap := c.indexStruct(ret, list_arr_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)

	// assign the fields
	c.cbb.NewStore(constant.NewNull(ptr(ddpint)), c.indexStruct(ret, list_refc_field_index))
	c.cbb.NewStore(result, retArr)
	c.cbb.NewStore(count, retLen)
	c.cbb.NewStore(count, retCap)

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_free_x function for a listType

ddp_free_x frees the array in list->arr.
It does not change list->len or list->cap, as list
should not be used after being freed.

signature:
c.void.IrType() ddp_free_x(x* list)
*/
func (c *compiler) createListFree(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	list := ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_free_%s", listType.typ.Name()),
		c.void.IrType(),
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	free_arr := func() {
		if !listType.elementType.IsPrimitive() {
			/*
				for (int i = 0; i < list->len; i++) {
					free(list->arr[i]);
				}
			*/
			listArr, listLen := c.loadStructField(list, list_arr_field_index), c.loadStructField(list, list_len_field_index)
			c.createFor(zero, c.forDefaultCond(listLen),
				func(index value.Value) {
					val := c.indexArray(listArr, index)
					c.cbb.NewCall(listType.elementType.FreeFunc(), val)
				},
			)
		}

		listArr, listCap := c.loadStructField(list, list_arr_field_index), c.loadStructField(list, list_cap_field_index)
		c.freeArr(listArr, listCap)
	}

	refc_ptr := c.loadStructField(list, list_refc_field_index)
	refc_is_null := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(refc_ptr, ddpint), zero)

	c.createIfElse(refc_is_null, func() {
		free_arr()
		c.cbb.NewRet(nil)
	}, nil)

	refc := c.loadArrayElement(refc_ptr, zero)
	refc_decr := c.cbb.NewSub(refc, newInt(1))
	c.cbb.NewStore(refc_decr, refc_ptr)
	is_refc_zero := c.cbb.NewICmp(enum.IPredEQ, refc_decr, zero)
	c.createIfElse(is_refc_zero, func() {
		c.free(refc_ptr)
		free_arr()
	}, nil)

	c.cbb.NewRet(nil)
	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_deep_copy_x function for a listType

ddp_deep_copy_x allocates a new array of the same capacity
as list->cap and copies list->arr in there.
For non-primitive types deepCopies are created

signature:
c.void.IrType() ddp_deep_copy_x(x* ret, x* list)
*/
func (c *compiler) createListDeepCopy(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	ret, list := ir.NewParam("ret", listType.ptr), ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_deep_copy_%s", listType.typ.Name()),
		c.void.IrType(),
		ret,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(ret, list_arr_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)
	origArr, origLen, origCap := c.loadStructField(list, list_arr_field_index), c.loadStructField(list, list_len_field_index), c.loadStructField(list, list_cap_field_index)

	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(ret, i64), c.cbb.NewPtrToInt(list, i64))
	c.createIfElse(ptrs_equal, func() {
		c.cbb.NewRet(nil)
	},
		nil,
	)

	// *ret = DDP_EMPTY_LIST
	c.cbb.NewStore(listType.DefaultValue(), ret)

	// allocate(sizeof(t) * list->cap)
	arr := c.allocateArr(getPointeeTypeT(getPointeeType(arrFieldPtr)), origCap)

	// primitive types can easily be copied
	if listType.elementType.IsPrimitive() {
		// memcpy(arr, list->arr, list->len)
		c.memcpyArr(arr, origArr, origLen)
	} else { // non-primitive types need to be seperately deep-copied
		/*
			for (int i = 0; i < list->len; i++) {
				ddp_deep_copy(&ret->arr[i], &list->arr[i])
			}
		*/
		c.createFor(zero, c.forDefaultCond(origLen),
			func(index value.Value) {
				elementPtr := c.indexArray(arr, index)
				c.cbb.NewCall(listType.elementType.DeepCopyFunc(), elementPtr, c.indexArray(origArr, index))
			},
		)
	}

	c.cbb.NewStore(constant.NewNull(ptr(ddpint)), c.indexStruct(ret, list_refc_field_index)) // ret->refc = NULL
	c.cbb.NewStore(arr, arrFieldPtr)                                                         // ret->arr = new_arr
	c.cbb.NewStore(origLen, lenFieldPtr)                                                     // ret->len = list->len
	c.cbb.NewStore(origCap, capFieldPtr)                                                     // ret->cap = list->cap

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_shallow_copy_x function for a listType

ddp_shallow_copy_x creates a shallow copy of a list and sets the refc as needed

signature:
c.void.IrType() ddp_shallow_copy_x(x* ret, x* list)
*/
func (c *compiler) createListShallowCopy(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	ret, list := ir.NewParam("ret", listType.ptr), ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_shallow_copy_%s", listType.typ.Name()),
		c.void.IrType(),
		ret,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	origRefc := c.loadStructField(list, list_refc_field_index)
	// if (ret == list) return;
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(ret, i64), c.cbb.NewPtrToInt(list, i64))
	c.createIfElse(ptrs_equal, func() {
		c.cbb.NewRet(nil)
	},
		nil,
	)

	// *ret = DDP_EMPTY_LIST
	c.cbb.NewStore(listType.DefaultValue(), ret)

	/*
		if (list->refc == null) {
			list->refc = DDP_ALLOCATE(ddpint, 1)
			*list->refc = 1;
		}
	*/
	refc_is_null := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(origRefc, ddpint), zero)
	c.createIfElse(refc_is_null, func() {
		new_refc := c.allocate(ddpint)
		c.cbb.NewStore(newInt(1), new_refc)
		c.cbb.NewStore(new_refc, c.indexStruct(list, list_refc_field_index))
	}, nil)

	// *list->refc += 1
	origRefc = c.loadStructField(list, list_refc_field_index)
	refc_val := c.cbb.NewLoad(ddpint, origRefc)
	refc_incr := c.cbb.NewAdd(refc_val, newInt(1))
	c.cbb.NewStore(refc_incr, c.loadStructField(list, list_refc_field_index))

	// *ret = *list
	c.cbb.NewStore(c.cbb.NewLoad(listType.IrType(), list), ret)

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_perform_cow_x function for a listType

ddp_perform_cow_x "copies the list into itself" in case it is a cow

signature:
c.void.IrType() ddp_perform_cow_x(x* list)
*/
func (c *compiler) createListPerformCow(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	list := ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_perform_cow_%s", listType.typ.Name()),
		c.void.IrType(),
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	refc := c.loadStructField(list, list_refc_field_index)
	// if (list->refc == NULL) return;
	refc_is_null := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(refc, i64), zero)
	c.createIfElse(refc_is_null, func() {
		c.cbb.NewRet(nil)
	},
		nil,
	)

	// if(*list->refc == 1) return;
	refc_val := c.cbb.NewLoad(ddpint, refc)
	refc_is_one := c.cbb.NewICmp(enum.IPredEQ, refc_val, newInt(1))
	c.createIfElse(refc_is_one, func() {
		c.free(refc)
		c.cbb.NewStore(constant.NewNull(ptr(ddpint)), c.indexStruct(list, list_refc_field_index))
		c.cbb.NewRet(nil)
	}, nil)

	// if (list->arr == NULL) return;
	arr := c.loadStructField(list, list_arr_field_index)
	arr_is_null := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(arr, i64), zero)
	c.createIfElse(arr_is_null, func() {
		c.cbb.NewRet(nil)
	},
		nil,
	)

	// allocate(sizeof(t) * list->len)
	len := c.loadStructField(list, list_len_field_index)
	arr_copy := c.allocateArr(listType.elementType.IrType(), len)

	// primitive types can easily be copied
	if listType.elementType.IsPrimitive() {
		// memcpy(arr, list->arr, list->len)
		c.memcpyArr(arr_copy, arr, len)
	} else { // non-primitive types need to be seperately copied
		/*
			for (int i = 0; i < list->len; i++) {
				ddp_shallow_copy(&ret->arr[i], &list->arr[i])
			}
		*/
		c.createFor(zero, c.forDefaultCond(len),
			func(index value.Value) {
				c.cbb.NewCall(listType.elementType.ShallowCopyFunc(), c.indexArray(arr_copy, index), c.indexArray(arr, index))
			},
		)
	}

	// *list->refc -= 1;
	// list->refc = NULL;
	refc_decr := c.cbb.NewSub(refc_val, newInt(1))
	c.cbb.NewStore(refc_decr, refc)
	c.cbb.NewStore(constant.NewNull(ptr(ddpint)), c.indexStruct(list, list_refc_field_index))

	// list->arr = arr_copy;
	c.cbb.NewStore(arr_copy, c.indexStruct(list, list_arr_field_index))
	// list->cap = list->len
	c.cbb.NewStore(len, c.indexStruct(list, list_cap_field_index))

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_x_equal function for a listType

ddp_x_equal checks wether the two lists are equal

signature:
bool ddp_x_equal(x* list1, x* list2)
*/
func (c *compiler) createListEquals(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	list1, list2 := ir.NewParam("list1", listType.ptr), ir.NewParam("list2", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_%s_equal", listType.typ.Name()),
		ddpbool,
		list1,
		list2,
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// if (list1 == list2) return true;
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(list1, i64), c.cbb.NewPtrToInt(list2, i64))
	c.createIfElse(ptrs_equal, func() {
		c.cbb.NewRet(constant.True)
	},
		nil,
	)

	// if (list1->refc != NULL && list1->refc == list2->refc) return true;
	refc1_ptr := c.loadStructField(list1, list_refc_field_index)
	refc1_not_null := c.cbb.NewICmp(enum.IPredNE, c.cbb.NewPtrToInt(refc1_ptr, ddpint), zero)
	refc2_ptr := c.loadStructField(list2, list_refc_field_index)
	refcs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(refc1_ptr, ddpint), c.cbb.NewPtrToInt(refc2_ptr, ddpint))
	refcs_equal_and := c.cbb.NewAnd(refc1_not_null, refcs_equal)
	c.createIfElse(refcs_equal_and, func() {
		c.cbb.NewRet(constant.True)
	}, nil)

	// if (list1->len != list2->len) return false;
	list1_len := c.loadStructField(list1, list_len_field_index)
	len_unequal := c.cbb.NewICmp(enum.IPredNE, list1_len, c.loadStructField(list2, list_len_field_index))
	c.createIfElse(len_unequal, func() {
		c.cbb.NewRet(constant.False)
	},
		nil,
	)

	// compare single elements
	// primitive types can easily be compared
	if listType.elementType.IsPrimitive() {
		// return memcmp(list1->arr, list2->arr, sizeof(T) * list1->len) == 0;
		size := c.cbb.NewMul(c.sizeof(listType.elementType.IrType()), list1_len)
		memcmp := c.memcmp(c.loadStructField(list1, list_arr_field_index), c.loadStructField(list2, list_arr_field_index), size)
		c.cbb.NewRet(c.cbb.NewICmp(enum.IPredEQ, memcmp, zero))
	} else { // non-primitive types need to be seperately compared
		/*
			for (int i = 0; i < list1->len; i++) {
				if (!ddp_element_equal(&list1->arr[i], &list2->arr[i]))
					return false;
			}
		*/
		c.createFor(zero, c.forDefaultCond(list1_len),
			func(index value.Value) {
				list1_arr, list2_arr := c.loadStructField(list1, list_arr_field_index), c.loadStructField(list2, list_arr_field_index)
				list1_at_count, list2_at_count := c.indexArray(list1_arr, index), c.indexArray(list2_arr, index)
				elements_unequal := c.cbb.NewXor(c.cbb.NewCall(listType.elementType.EqualsFunc(), list1_at_count, list2_at_count), newInt(1))

				c.createIfElse(elements_unequal, func() {
					c.cbb.NewRet(constant.False)
				},
					nil,
				)
			},
		)

		c.cbb.NewRet(constant.True)
	}

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_x_slice function for a listType

ddp_x_slice slices a list by two indices

signature:
void ddp_x_slice(x* ret, x* list, ddpint index1, ddpint index2)
*/
func (c *compiler) createListSlice(listType *ddpIrListType, declarationOnly bool) *ir.Func {
	var (
		ret  = ir.NewParam("ret", listType.ptr)
		list = ir.NewParam("list", listType.ptr)
		// index1/2 are values for later reassignement of the variables
		// for readybility
		index1 value.Value = ir.NewParam("index1", ddpint)
		index2 value.Value = ir.NewParam("index2", ddpint)
	)
	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_%s_slice", listType.typ.Name()),
		c.void.IrType(),
		ret,
		list,
		index1.(*ir.Param),
		index2.(*ir.Param),
	)
	irFunc.CallingConv = enum.CallingConvC

	if declarationOnly {
		irFunc.Linkage = enum.LinkageExternal
		return irFunc
	}

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// empty the ret
	c.cbb.NewStore(listType.DefaultValue(), ret)

	listLen := c.loadStructField(list, list_len_field_index)

	// if (list->len <= 0) return;
	list_empty := c.cbb.NewICmp(enum.IPredSLE, listLen, zero)
	c.createIfElse(list_empty, func() {
		c.cbb.NewRet(nil)
	},
		nil,
	)

	// helper for the clamp function (does what its name suggests)
	clamp := func(val, min, max value.Value) value.Value {
		temp := c.createTernary(
			c.cbb.NewICmp(enum.IPredSLT, val, min),
			func() value.Value { return min },
			func() value.Value { return val },
		)
		return c.createTernary(
			c.cbb.NewICmp(enum.IPredSGT, temp, max),
			func() value.Value { return max },
			func() value.Value { return temp },
		)
	}

	// clamp the indices to the list bounds
	index1 = clamp(index1, newInt(1), listLen)
	index2 = clamp(index2, newInt(1), listLen)

	// validate that the indices are valid
	i2_less_i1 := c.cbb.NewICmp(enum.IPredSLT, index2, index1)
	c.createIfElse(i2_less_i1,
		func() {
			c.runtime_error(1, c.slice_error_string, index1, index2)
		},
		nil,
	)

	// sub 1 from the indices because ddp-indices start at 1 not 0
	index1 = c.cbb.NewSub(index1, newInt(1))
	index2 = c.cbb.NewSub(index2, newInt(1))

	retArrPtr, retLenPtr, retCapPtr := c.indexStruct(ret, list_arr_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)

	/*
		ret->len = (index2 - index1) + 1; // + 1 if indices are equal
		ret->cap = GROW_CAPACITY(ret->len);
		ret->arr = ALLOCATE(elementType, ret->cap);
	*/
	new_len := c.cbb.NewAdd(c.cbb.NewSub(index2, index1), newInt(1))
	c.cbb.NewStore(new_len, retLenPtr)
	c.cbb.NewStore(c.growCapacity(new_len), retCapPtr)
	c.cbb.NewStore(c.allocateArr(listType.elementType.IrType(), c.loadStructField(ret, list_cap_field_index)), retArrPtr)

	if listType.elementType.IsPrimitive() {
		// memcpy primitive types
		c.memcpyArr(c.loadStructField(ret, list_arr_field_index), c.indexArray(c.loadStructField(list, list_arr_field_index), index1), new_len)
	} else {
		/*
			size_t j = 0;
			for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
				ddp_deep_copy_x(&ret->arr[j], list->arr[i]);
			}
		*/

		retArr := c.loadStructField(ret, list_arr_field_index)
		j := c.NewAlloca(i64) // the j index variable
		c.cbb.NewStore(zero, j)
		c.createFor(index1,
			// i <= index2 && i < list->len
			func(i value.Value) value.Value {
				cond1 := c.cbb.NewICmp(enum.IPredSLE, i, index2)
				cond2 := c.cbb.NewICmp(enum.IPredSLT, i, listLen)
				return c.cbb.NewAnd(cond1, cond2)
			},
			// _deep_copy_x(&ret->arr[j], &list->arr[i]); j++
			func(index value.Value) {
				listArr := c.loadStructField(list, list_arr_field_index)
				jval := c.cbb.NewLoad(i64, j)
				elementPtr := c.indexArray(retArr, jval)
				listElementPtr := c.indexArray(listArr, index)
				c.cbb.NewCall(listType.elementType.DeepCopyFunc(), elementPtr, listElementPtr)
				c.cbb.NewStore(c.cbb.NewAdd(jval, newInt(1)), j)
			},
		)
	}

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the ddp_x_y_verkettet functions for a listType
and returns them in the order:
list_list, list_scalar, scalar_scalar, scalar_list
*/
func (c *compiler) createListConcats(listType *ddpIrListType, declarationOnly bool) (*ir.Func, *ir.Func, *ir.Func, *ir.Func) {
	// reusable parts for all 4 functions

	// non-primitive types are passed as pointers
	// until I can pass structs correctly
	scal_param_type := listType.elementType.IrType()
	if !listType.elementType.IsPrimitive() {
		scal_param_type = ptr(scal_param_type)
	}

	var (
		cbb *ir.Block
		cf  *ir.Func
	)
	setup := func(irfun *ir.Func) {
		cbb, cf = c.cbb, c.cf // save the current basic block and ir function
		c.cf = irfun
		c.cbb = c.cf.NewBlock("")
	}

	/*
		list->refc = 0;
		list->len = 0;
		list->cap = 0;
		list->arr = NULL;
	*/
	empty_list := func(list value.Value) {
		c.cbb.NewStore(listType.DefaultValue(), list)
		// c.cbb.NewStore(constant.NewNull(ptr(ddpint)), c.indexStruct(list, list_refc_field_index))
		// c.cbb.NewStore(constant.NewNull(listType.elementType.PtrType()), c.indexStruct(list, list_arr_field_index))
		// c.cbb.NewStore(zero, c.indexStruct(list, list_len_field_index))
		// c.cbb.NewStore(zero, c.indexStruct(list, list_cap_field_index))
	}

	/*
		while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		ret->arr = ddp_reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * ret->cap);
	*/
	grow_capacity_and_set_arr := func(ret, list value.Value) {
		retCapPtr := c.indexStruct(ret, list_cap_field_index)
		// while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		c.createWhile(func() value.Value {
			return c.cbb.NewICmp(enum.IPredSLT, c.loadStructField(ret, list_cap_field_index), c.loadStructField(ret, list_len_field_index))
		}, func() {
			c.cbb.NewStore(c.growCapacity(c.loadStructField(ret, list_cap_field_index)), retCapPtr)
		})

		// ret->arr = ddp_reallocate(list1->arr, sizeof(elementType) * list1->cap, sizeof(elementType) * ret->cap)
		new_arr := c.growArr(c.loadStructField(list, list_arr_field_index), c.loadStructField(list, list_cap_field_index), c.loadStructField(ret, list_cap_field_index))
		c.cbb.NewStore(new_arr, c.indexStruct(ret, list_arr_field_index))
	}

	finish := func(irfun *ir.Func) *ir.Func {
		c.cbb.NewRet(nil)

		c.cbb, c.cf = cbb, cf // restore the basic block and ir function

		c.insertFunction(irfun.Name(), nil, irfun)
		return irfun
	}

	// defines the list_list_verkettet function
	list_list_concat := func() *ir.Func {
		ret, list1, list2 := ir.NewParam("ret", listType.ptr), ir.NewParam("list1", listType.ptr), ir.NewParam("list2", listType.ptr)
		concListList := c.mod.NewFunc(
			fmt.Sprintf("ddp_%s_%s_verkettet", listType.typ.Name(), listType.typ.Name()),
			c.void.IrType(),
			ret, list1, list2,
		)
		concListList.CallingConv = enum.CallingConvC

		if declarationOnly {
			concListList.Linkage = enum.LinkageExternal
			return concListList
		}

		setup(concListList)

		retRefcPtr, retLenPtr, retCapPtr := c.indexStruct(ret, list_refc_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)
		// ret->len = list1->len + list2->len
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list1, list_len_field_index), c.loadStructField(list2, list_len_field_index)), retLenPtr)
		// ret->cap = list1->cap
		c.cbb.NewStore(c.loadStructField(list1, list_cap_field_index), retCapPtr)
		// ret->refc = list1->refc
		c.cbb.NewStore(c.loadStructField(list1, list_refc_field_index), retRefcPtr)

		grow_capacity_and_set_arr(ret, list1)

		new_arr := c.loadStructField(ret, list_arr_field_index)

		if listType.elementType.IsPrimitive() {
			// memcpy(&ret->arr[list1->len], list2->arr, sizeof(elementType) * list2->len)
			c.memcpyArr(c.indexArray(new_arr, c.loadStructField(list1, list_len_field_index)), c.loadStructField(list2, list_arr_field_index), c.loadStructField(list2, list_len_field_index))
		} else {
			list1Len, list2Len := c.loadStructField(list1, list_len_field_index), c.loadStructField(list2, list_len_field_index)
			list2Arr := c.loadStructField(list2, list_arr_field_index)
			/*
				for (int i = 0; i < list->len; i++) {
					ddp_deep_copy(&ret->arr[i+list1->len], &list2->arr[i])
				}
			*/
			c.createFor(zero, c.forDefaultCond(list2Len),
				func(index value.Value) {
					elementPtr := c.indexArray(new_arr, c.cbb.NewAdd(list1Len, index))
					c.cbb.NewCall(listType.elementType.DeepCopyFunc(), elementPtr, c.indexArray(list2Arr, index))
				},
			)
		}

		empty_list(list1)

		return finish(concListList)
	}

	list_scalar_concat := func() *ir.Func {
		ret, list, scal := ir.NewParam("ret", listType.ptr), ir.NewParam("list", listType.ptr), ir.NewParam("scal", scal_param_type)
		concListScal := c.mod.NewFunc(
			fmt.Sprintf("ddp_%s_%s_verkettet", listType.typ.Name(), listType.elementType.Name()),
			c.void.IrType(),
			ret, list, scal,
		)
		concListScal.CallingConv = enum.CallingConvC

		if declarationOnly {
			concListScal.Linkage = enum.LinkageExternal
			return concListScal
		}

		setup(concListScal)

		retRefcPtr, retLenPtr, retCapPtr := c.indexStruct(ret, list_refc_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list, list_len_field_index), newInt(1)), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.loadStructField(list, list_cap_field_index), retCapPtr)
		// ret->refc = list1->refc
		c.cbb.NewStore(c.loadStructField(list, list_refc_field_index), retRefcPtr)

		grow_capacity_and_set_arr(ret, list)

		retArr := c.loadStructField(ret, list_arr_field_index)
		listLen := c.loadStructField(list, list_len_field_index)
		if listType.elementType.IsPrimitive() {
			// ret->arr[list->len] = scal
			c.cbb.NewStore(scal, c.indexArray(retArr, listLen))
		} else {
			// ddp_deep_copy_scal(&ret->arr[list->len], scal)
			dst := c.indexArray(retArr, listLen)
			c.cbb.NewCall(listType.elementType.DeepCopyFunc(), dst, scal)
		}

		empty_list(list)

		return finish(concListScal)
	}

	scalar_scalar_concat := func() *ir.Func {
		// string concatenations result in a new string
		if listType.elementType == c.ddpstring {
			return nil
		}

		ret, scal1, scal2 := ir.NewParam("ret", listType.ptr), ir.NewParam("scal1", scal_param_type), ir.NewParam("scal2", scal_param_type)
		concScalScal := c.mod.NewFunc(
			fmt.Sprintf("ddp_%s_%s_verkettet", listType.elementType.Name(), listType.elementType.Name()),
			c.void.IrType(),
			ret, scal1, scal2,
		)
		concScalScal.CallingConv = enum.CallingConvC

		if declarationOnly {
			concScalScal.Linkage = enum.LinkageExternal
			return concScalScal
		}

		setup(concScalScal)

		retRefcPtr, retArrPtr, retLenPtr, retCapPtr := c.indexStruct(ret, list_refc_field_index), c.indexStruct(ret, list_arr_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(newInt(2), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.growCapacity(newInt(2)), retCapPtr)
		// ret->arr = ALLOCATE(elementType, 2)
		c.cbb.NewStore(c.allocateArr(listType.elementType.IrType(), c.cbb.NewLoad(ddpint, retCapPtr)), retArrPtr)
		// ret->refc = NULL
		c.cbb.NewStore(constant.NewNull(ptr(ddpint)), retRefcPtr)

		retArr := c.loadStructField(ret, list_arr_field_index)
		retArr0Ptr, retArr1Ptr := c.indexArray(retArr, zero), c.indexArray(retArr, newInt(1))
		if listType.elementType.IsPrimitive() {
			// ret->arr[0] = scal1;
			// ret->arr[1] = scal1;
			c.cbb.NewStore(scal1, retArr0Ptr)
			c.cbb.NewStore(scal2, retArr1Ptr)
		} else {
			// ddp_deep_copy_scalar(&ret->arr[0], scal1);
			// ddp_deep_copy_scalar(&ret->arr[0], scal1);
			c.cbb.NewCall(listType.elementType.DeepCopyFunc(), retArr0Ptr, scal1)
			c.cbb.NewCall(listType.elementType.DeepCopyFunc(), retArr0Ptr, scal2)
		}

		return finish(concScalScal)
	}

	scalar_list_concat := func() *ir.Func {
		ret, scal, list := ir.NewParam("ret", listType.ptr), ir.NewParam("scal", scal_param_type), ir.NewParam("list", listType.ptr)
		concScalList := c.mod.NewFunc(
			fmt.Sprintf("ddp_%s_%s_verkettet", listType.elementType.Name(), listType.typ.Name()),
			c.void.IrType(),
			ret, scal, list,
		)
		concScalList.CallingConv = enum.CallingConvC

		if declarationOnly {
			concScalList.Linkage = enum.LinkageExternal
			return concScalList
		}

		setup(concScalList)

		retRefcPtr, retLenPtr, retCapPtr := c.indexStruct(ret, list_refc_field_index), c.indexStruct(ret, list_len_field_index), c.indexStruct(ret, list_cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list, list_len_field_index), newInt(1)), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.loadStructField(list, list_cap_field_index), retCapPtr)
		// ret->refc = list->refc
		c.cbb.NewStore(c.loadStructField(list, list_refc_field_index), retRefcPtr)

		grow_capacity_and_set_arr(ret, list)

		// memmove(&ret->arr[1], ret->arr, sizeof(elementType) * list->len);
		retArr := c.loadStructField(ret, list_arr_field_index)
		c.memmoveArr(c.indexArray(retArr, newInt(1)), retArr, c.loadStructField(list, list_len_field_index))

		if listType.elementType.IsPrimitive() {
			// ret->arr[0] = scal;
			c.cbb.NewStore(scal, retArr)
		} else {
			// ddp_deep_copy(&ret->arr[0], scal)
			c.cbb.NewCall(listType.elementType.DeepCopyFunc(), retArr, scal)
		}

		empty_list(list)

		return finish(concScalList)
	}

	return list_list_concat(), list_scalar_concat(), scalar_scalar_concat(), scalar_list_concat()
}
