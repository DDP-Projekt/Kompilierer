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
	fromConstantsIrFun          *ir.Func           // the fromConstans ir func
	freeIrFun                   *ir.Func           // the free ir func
	deepCopyIrFun               *ir.Func           // the deepCopy ir func
	equalsIrFun                 *ir.Func           // the equals ir func
	sliceIrFun                  *ir.Func           // the clice ir func
	list_list_concat_IrFunc     *ir.Func           // the list_list_verkettet ir func
	list_scalar_concat_IrFunc   *ir.Func           // the list_scalar_verkettet ir func
	scalar_scalar_concat_IrFunc *ir.Func           // the scalar_scalar_verkettet ir func
	scalar_list_concat_IrFunc   *ir.Func           // the scalar_list_verkettet ir func
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
		constant.NewNull(t.elementType.PtrType()),
		zero,
		zero,
	)
}

func (t *ddpIrListType) FreeFunc() *ir.Func {
	return t.freeIrFun
}

func (t *ddpIrListType) DeepCopyFunc() *ir.Func {
	return t.deepCopyIrFun
}

func (t *ddpIrListType) EqualsFunc() *ir.Func {
	return t.equalsIrFun
}

// defines the struct of a list type
// and all the necessery functions
// from the given elementType and name
// C-Equivalent:
/*
	struct listtypename {
		ElementType* arr;
		ddpint len;
		ddpint cap;
	}
*/
func (c *Compiler) defineListType(name string, elementType ddpIrType) *ddpIrListType {
	list := &ddpIrListType{}
	list.elementType = elementType
	list.typ = c.mod.NewTypeDef(name, types.NewStruct(
		list.elementType.PtrType(), // underlying array
		ddpint,                     // length
		ddpint,                     // capacity
	))
	list.ptr = ptr(list.typ)

	list.fromConstantsIrFun = c.defineFromConstants(list)
	list.freeIrFun = c.defineFree(list)
	list.deepCopyIrFun = c.defineDeepCopy(list)
	list.equalsIrFun = c.defineEquals(list)
	list.sliceIrFun = c.defineSlice(list)

	list.list_list_concat_IrFunc,
		list.list_scalar_concat_IrFunc,
		list.scalar_scalar_concat_IrFunc,
		list.scalar_list_concat_IrFunc = c.defineConcats(list)

	return list
}

const (
	arr_field_index = 0
	len_field_index = 1
	cap_field_index = 2
)

/*
defines the ddp_x_from_constants function for a listType

ddp_x_from_constants allocates an array big enough to hold count
elements and stores it inside ret->arr.
It also sets ret->len and ret->cap accordingly

signature:
c.void.IrType() ddp_x_from_constants(x* ret, ddpint count)
*/
func (c *Compiler) defineFromConstants(listType *ddpIrListType) *ir.Func {
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
	retArr, retLen, retCap := c.indexStruct(ret, arr_field_index), c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)

	// assign the fields
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
func (c *Compiler) defineFree(listType *ddpIrListType) *ir.Func {
	list := ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_free_%s", listType.typ.Name()),
		c.void.IrType(),
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	if !listType.elementType.IsPrimitive() {
		/*
			for (int i = 0; i < list->len; i++) {
				free(list->arr[i]);
			}
		*/
		listArr, listLen := c.loadStructField(list, arr_field_index), c.loadStructField(list, len_field_index)
		c.createFor(zero, c.forDefaultCond(listLen),
			func(index value.Value) {
				val := c.indexArray(listArr, index)
				c.cbb.NewCall(listType.elementType.FreeFunc(), val)
			},
		)
	}

	listArr, listCap := c.loadStructField(list, arr_field_index), c.loadStructField(list, cap_field_index)
	c.freeArr(listArr, listCap)

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
func (c *Compiler) defineDeepCopy(listType *ddpIrListType) *ir.Func {
	ret, list := ir.NewParam("ret", listType.ptr), ir.NewParam("list", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_deep_copy_%s", listType.typ.Name()),
		c.void.IrType(),
		ret,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(ret, arr_field_index), c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)
	origArr, origLen, origCap := c.loadStructField(list, arr_field_index), c.loadStructField(list, len_field_index), c.loadStructField(list, cap_field_index)

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

	c.cbb.NewStore(arr, arrFieldPtr)     // ret->arr = new_arr
	c.cbb.NewStore(origLen, lenFieldPtr) // ret->len = list->len
	c.cbb.NewStore(origCap, capFieldPtr) // ret->cap = list->cap

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
func (c *Compiler) defineEquals(listType *ddpIrListType) *ir.Func {
	list1, list2 := ir.NewParam("list1", listType.ptr), ir.NewParam("list2", listType.ptr)

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("ddp_%s_equal", listType.typ.Name()),
		ddpbool,
		list1,
		list2,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// if (list1 == list2) return true;
	c.cbb.NewPtrToInt(list1, i64)
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, c.cbb.NewPtrToInt(list1, i64), c.cbb.NewPtrToInt(list2, i64))
	c.createIfElese(ptrs_equal, func() {
		c.cbb.NewRet(constant.True)
	},
		nil,
	)

	// if (list1->len != list2->len) return false;
	list1_len := c.loadStructField(list1, len_field_index)
	len_unequal := c.cbb.NewICmp(enum.IPredNE, list1_len, c.loadStructField(list2, len_field_index))
	c.createIfElese(len_unequal, func() {
		c.cbb.NewRet(constant.False)
	},
		nil,
	)

	// compare single elements
	// primitive types can easily be compared
	if listType.elementType.IsPrimitive() {
		// return memcmp(list1->arr, list2->arr, sizeof(T) * list1->len) == 0;
		size := c.cbb.NewMul(c.sizeof(listType.elementType.IrType()), list1_len)
		memcmp := c.memcmp(c.loadStructField(list1, arr_field_index), c.loadStructField(list2, arr_field_index), size)
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
				list1_arr, list2_arr := c.loadStructField(list1, arr_field_index), c.loadStructField(list2, arr_field_index)
				list1_at_count, list2_at_count := c.indexArray(list1_arr, index), c.indexArray(list2_arr, index)
				elements_unequal := c.cbb.NewXor(c.cbb.NewCall(listType.elementType.EqualsFunc(), list1_at_count, list2_at_count), newInt(1))

				c.createIfElese(elements_unequal, func() {
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

var slice_error_string *ir.Global

/*
defines the ddp_x_slice function for a listType

ddp_x_slice slices a list by two indices

signature:
bool ddp_x_slice(x* ret, x* list, ddpint index1, ddpint index2)
*/
func (c *Compiler) defineSlice(listType *ddpIrListType) *ir.Func {
	var (
		ret  *ir.Param = ir.NewParam("ret", listType.ptr)
		list *ir.Param = ir.NewParam("list", listType.ptr)
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

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// empty the ret
	c.cbb.NewStore(constant.NewNull(listType.elementType.PtrType()), c.indexStruct(ret, arr_field_index))
	c.cbb.NewStore(zero, c.indexStruct(ret, len_field_index))
	c.cbb.NewStore(zero, c.indexStruct(ret, cap_field_index))

	listLen := c.loadStructField(list, len_field_index)

	// if (list->len <= 0) return;
	list_empty := c.cbb.NewICmp(enum.IPredSLE, listLen, zero)
	c.createIfElese(list_empty, func() {
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
	c.createIfElese(i2_less_i1,
		func() {
			if slice_error_string == nil {
				slice_error_string = c.mod.NewGlobalDef("", constant.NewCharArrayFromString("Invalide Indexe (Index 1 war %ld, Index 2 war %ld)\n"))
				slice_error_string.Visibility = enum.VisibilityHidden
				slice_error_string.Immutable = true
			}
			c.runtime_error(newInt(1), slice_error_string, index1, index2)
		},
		nil,
	)

	// sub 1 from the indices because ddp-indices start at 1 not 0
	index1 = c.cbb.NewSub(index1, newInt(1))
	index2 = c.cbb.NewSub(index2, newInt(1))

	retArrPtr, retLenPtr, retCapPtr := c.indexStruct(ret, arr_field_index), c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)

	/*
		ret->len = (index2 - index1) + 1; // + 1 if indices are equal
		ret->cap = GROW_CAPACITY(ret->len);
		ret->arr = ALLOCATE(elementType, ret->cap);
	*/
	new_len := c.cbb.NewAdd(c.cbb.NewSub(index2, index1), newInt(1))
	c.cbb.NewStore(new_len, retLenPtr)
	c.cbb.NewStore(c.growCapacity(new_len), retCapPtr)
	c.cbb.NewStore(c.allocateArr(listType.elementType.IrType(), c.loadStructField(ret, cap_field_index)), retArrPtr)

	if listType.elementType.IsPrimitive() {
		// memcpy primitive types
		c.memcpyArr(c.loadStructField(ret, arr_field_index), c.indexArray(c.loadStructField(list, arr_field_index), index1), new_len)
	} else {
		/*
			size_t j = 0;
			for (size_t i = index1; i <= index2 && i < list->len; i++, j++) {
				ddp_deep_copy_x(&ret->arr[j], list->arr[i]);
			}
		*/

		retArr := c.loadStructField(ret, arr_field_index)
		j := c.cbb.NewAlloca(i64) // the j index variable
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
				listArr := c.loadStructField(list, arr_field_index)
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

TODO: create the missing concat functions
*/
func (c *Compiler) defineConcats(listType *ddpIrListType) (*ir.Func, *ir.Func, *ir.Func, *ir.Func) {
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
		list->len = 0;
		list->cap = 0;
		list->arr = NULL;
	*/
	empty_list := func(list value.Value) {
		c.cbb.NewStore(constant.NewNull(listType.elementType.PtrType()), c.indexStruct(list, arr_field_index))
		c.cbb.NewStore(zero, c.indexStruct(list, len_field_index))
		c.cbb.NewStore(zero, c.indexStruct(list, cap_field_index))
	}

	/*
		while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		ret->arr = ddp_reallocate(list->arr, sizeof(ddpchar) * list->cap, sizeof(ddpchar) * ret->cap);
	*/
	grow_capacity_and_set_arr := func(ret, list value.Value) {
		retCapPtr := c.indexStruct(ret, cap_field_index)
		// while (ret->cap < ret->len) ret->cap = GROW_CAPACITY(ret->cap);
		c.createWhile(func() value.Value {
			return c.cbb.NewICmp(enum.IPredSLT, c.loadStructField(ret, cap_field_index), c.loadStructField(ret, len_field_index))
		}, func() {
			c.cbb.NewStore(c.growCapacity(c.loadStructField(ret, cap_field_index)), retCapPtr)
		})

		// ret->arr = ddp_reallocate(list1->arr, sizeof(elementType) * list1->cap, sizeof(elementType) * ret->cap)
		new_arr := c.growArr(c.loadStructField(list, arr_field_index), c.loadStructField(list, cap_field_index), c.loadStructField(ret, cap_field_index))
		c.cbb.NewStore(new_arr, c.indexStruct(ret, arr_field_index))
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

		setup(concListList)

		retLenPtr, retCapPtr := c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)
		// ret->len = list1->len + list2->len
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list1, len_field_index), c.loadStructField(list2, len_field_index)), retLenPtr)
		// ret->cap = list1->cap
		c.cbb.NewStore(c.loadStructField(list1, cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list1)

		new_arr := c.loadStructField(ret, arr_field_index)

		if listType.elementType.IsPrimitive() {
			// memcpy(&ret->arr[list1->len], list2->arr, sizeof(elementType) * list2->len)
			c.memcpyArr(c.indexArray(new_arr, c.loadStructField(list1, len_field_index)), c.loadStructField(list2, arr_field_index), c.loadStructField(list2, len_field_index))
		} else {
			list1Len, list2Len := c.loadStructField(list1, len_field_index), c.loadStructField(list2, len_field_index)
			list2Arr := c.loadStructField(list2, arr_field_index)
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

		setup(concListScal)

		retLenPtr, retCapPtr := c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list, len_field_index), newInt(1)), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.loadStructField(list, cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list)

		retArr := c.loadStructField(ret, arr_field_index)
		listLen := c.loadStructField(list, len_field_index)
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

		setup(concScalScal)

		retArrPtr, retLenPtr, retCapPtr := c.indexStruct(ret, arr_field_index), c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(newInt(2), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.growCapacity(newInt(2)), retCapPtr)
		// ret->arr = ALLOCATE(elementType, 2)
		c.cbb.NewStore(c.allocateArr(listType.elementType.IrType(), c.cbb.NewLoad(ddpint, retCapPtr)), retArrPtr)

		retArr := c.loadStructField(ret, arr_field_index)
		retArr0Ptr, retArr1Ptr := c.indexArray(retArr, newInt(0)), c.indexArray(retArr, newInt(1))
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

		setup(concScalList)

		retLenPtr, retCapPtr := c.indexStruct(ret, len_field_index), c.indexStruct(ret, cap_field_index)
		// ret->len = list->len + 1
		c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list, len_field_index), newInt(1)), retLenPtr)
		// ret->cap = list->cap
		c.cbb.NewStore(c.loadStructField(list, cap_field_index), retCapPtr)

		grow_capacity_and_set_arr(ret, list)

		// memmove(&ret->arr[1], ret->arr, sizeof(elementType) * list->len);
		retArr := c.loadStructField(ret, arr_field_index)
		c.memmoveArr(c.indexArray(retArr, newInt(1)), retArr, c.loadStructField(list, len_field_index))

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
