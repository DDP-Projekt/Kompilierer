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
type ddpIrListType struct {
	typ                         types.Type // the typedef
	elementType                 ddpIrType  // the ir-type of the list elements
	fromConstantsIrFun          *ir.Func   // the fromConstans ir func
	freeIrFun                   *ir.Func   // the free ir func
	deepCopyIrFun               *ir.Func   // the deepCopy ir func
	equalsIrFun                 *ir.Func   // the equals ir func
	list_list_concat_IrFunc     *ir.Func   // the list_list_verkettet ir func
	list_scalar_concat_IrFunc   *ir.Func   // the list_scalar_verkettet ir func
	scalar_scalar_concat_IrFunc *ir.Func   // the scalar_scalar_verkettet ir func
	scalar_list_concat_IrFunc   *ir.Func   // the scalar_list_verkettet ir func
}

var _ ddpIrType = (*ddpIrListType)(nil)

func (t *ddpIrListType) IrType() types.Type {
	return t.typ
}

func (*ddpIrListType) IsPrimitive() bool {
	return false
}

func (t *ddpIrListType) DefaultValue() value.Value {
	return constant.NewStruct(t.typ.(*types.StructType),
		constant.NewNull(ptr(t.elementType.IrType())),
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
func (c *Compiler) defineListType(name string, elementType ddpIrType) *ddpIrListType {
	list := &ddpIrListType{}
	list.elementType = elementType
	list.typ = c.mod.NewTypeDef(name, types.NewStruct(
		ptr(list.elementType.IrType()), // underlying array
		ddpint,                         // length
		ddpint,                         // capacity
	))

	list.fromConstantsIrFun = c.defineFromConstants(list.typ)
	list.freeIrFun = c.defineFree(list)
	list.deepCopyIrFun = c.defineDeepCopy(list)
	list.equalsIrFun = c.defineEquals(list)

	list.list_list_concat_IrFunc,
		list.list_scalar_concat_IrFunc,
		list.scalar_scalar_concat_IrFunc,
		list.scalar_list_concat_IrFunc = c.defineConcats(list)

	return list
}

// returns the elementType of the given listType
// listType must have been declared using defineListType
func getListElementType(listType types.Type) types.Type {
	return listType.(*types.StructType).Fields[0].(*types.PointerType).ElemType
}

/*
defines the _ddp_x_from_constants function for a listType

_ddp_x_from_constans allocates an array big enough to hold count
elements and stores it inside ret->arr.
It also sets ret->len and ret->cap accordingly
*/
func (c *Compiler) defineFromConstants(listType types.Type) *ir.Func {
	// declare the parameters to use them as values
	ret, count := ir.NewParam("ret", ptr(listType)), ir.NewParam("count", ddpint)
	// declare the function
	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_%s_from_constants", listType.Name()),
		void,
		ret,
		count,
	)
	irFunc.CallingConv = enum.CallingConvC

	elementType := getListElementType(listType)
	arrType := ptr(elementType)

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	// start block
	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	cond := c.cbb.NewICmp(enum.IPredSGT, count, zero) // count > 0

	result := c.createTernary(cond,
		func() value.Value { return c.allocateArr(elementType, count) },
		func() value.Value { return constant.NewNull(arrType) },
	)

	// get pointers to the struct fields
	retArr, retLen, retCap := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)

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
defines the _ddp_free_x function for a listType

_ddp_free_x frees the array in list->arr.
It does not change list->len or list->cap, as list
should not be used after being freed.
*/
func (c *Compiler) defineFree(listType *ddpIrListType) *ir.Func {
	list := ir.NewParam("list", ptr(listType.typ))

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_free_%s", listType.typ.Name()),
		void,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	if listType.elementType.IsPrimitive() {
		listArr, listCap := c.loadStructField(list, 0), c.loadStructField(list, 2)
		c.freeArr(listArr, listCap)
	} else {
		listArr, listLen := c.loadStructField(list, 0), c.loadStructField(list, 1)

		c.createFor(func() value.Value { return zero }, func() value.Value { return listLen },
			func(index value.Value) {
				val := c.loadArrayElement(listArr, index)
				c.cbb.NewCall(listType.elementType.FreeFunc(), val)
			},
		)
	}

	c.cbb.NewRet(nil)

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

/*
defines the _ddp_deep_copy_x function for a listType

_ddp_deep_copy_x allocates a new array of the same capacity
as list->cap and copies list->arr in there.
For non-primitive types deepCopies are created
*/
func (c *Compiler) defineDeepCopy(listType *ddpIrListType) *ir.Func {
	ret, list := ir.NewParam("ret", ptr(listType.typ)), ir.NewParam("list", ptr(listType.typ))

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_deep_copy_%s", listType.typ.Name()),
		void,
		ret,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")
	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)
	origArr, origLen, origCap := c.loadStructField(list, 0), c.loadStructField(list, 1), c.loadStructField(list, 2)

	arr := c.allocateArr(getPointeeTypeT(getPointeeType(arrFieldPtr)), origCap)

	// primitive types can easily be copied
	if listType.elementType.IsPrimitive() {
		c.memcpyArr(arr, origArr, origLen)
	} else { // non-primitive types need to be seperately deep-copied
		c.createFor(func() value.Value { return zero }, func() value.Value { return origLen },
			func(index value.Value) {
				oldVal := c.loadArrayElement(origArr, index)
				deepCopy := c.cbb.NewAlloca(listType.elementType.IrType())
				c.cbb.NewCall(listType.elementType.DeepCopyFunc(), deepCopy, oldVal)
				c.cbb.NewStore(c.cbb.NewLoad(deepCopy.ElemType, deepCopy), c.indexArray(arr, index))
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
defines the _ddp_x_equal function for a listType

_ddp_x_equal checks wether the two lists are equal
*/
func (c *Compiler) defineEquals(listType *ddpIrListType) *ir.Func {
	list1, list2 := ir.NewParam("list1", ptr(listType.typ)), ir.NewParam("list2", ptr(listType.typ))

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_%s_equal", listType.typ.Name()),
		ddpbool,
		list1,
		list2,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = irFunc
	c.cbb = c.cf.NewBlock("")

	// if (list1 == list2) return true;
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, list1, list2)
	c.createIfElese(ptrs_equal, func() {
		c.cbb.NewRet(constant.True)
	},
		nil,
	)

	// if (list1->len != list2->len) return false;
	list1_len := c.loadStructField(list1, 1)
	len_unequal := c.cbb.NewICmp(enum.IPredNE, list1_len, c.loadStructField(list2, 1))

	c.createIfElese(len_unequal, func() {
		c.cbb.NewRet(constant.False)
	},
		nil,
	)

	// compare single elements
	// primitive types can easily be compared
	if listType.elementType.IsPrimitive() {
		// return memcmp(list1->arr, list2->arr, sizeof(T) * list1->len);
		size := c.cbb.NewMul(c.sizeof(listType.elementType.IrType()), list1_len)
		c.cbb.NewRet(c.memcmp(c.loadStructField(list1, 0), c.loadStructField(list2, 0), size))
	} else { // non-primitive types need to be seperately compared
		c.createFor(func() value.Value { return zero }, func() value.Value { return list1_len },
			func(index value.Value) {
				list1_arr, list2_arr := c.loadStructField(list1, 0), c.loadStructField(list2, 0)
				list1_at_count, list2_at_count := c.loadArrayElement(list1_arr, index), c.loadArrayElement(list2_arr, index)
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

/*
defines the _ddp_x_y_verkettet functions for a listType
and returns them in the order:
list_list, list_scalar, scalar_scalar, scalar_list

TODO: create the missing concat functions
*/
func (c *Compiler) defineConcats(listType *ddpIrListType) (*ir.Func, *ir.Func, *ir.Func, *ir.Func) {
	ret, list1, list2 := ir.NewParam("ret", ptr(listType.typ)), ir.NewParam("list1", ptr(listType.typ)), ir.NewParam("list2", ptr(listType.typ))

	concListList := c.mod.NewFunc(
		fmt.Sprintf("_ddp_%s_%s_verkettet", listType.typ.Name(), listType.typ.Name()),
		void,
		ret, list1, list2,
	)

	cbb, cf := c.cbb, c.cf // save the current basic block and ir function

	c.cf = concListList
	c.cbb = c.cf.NewBlock("")

	retLenPtr, retCapPtr := c.indexStruct(ret, 1), c.indexStruct(ret, 2)
	// ret->len = list1->len + list2->len
	c.cbb.NewStore(c.cbb.NewAdd(c.loadStructField(list1, 1), c.loadStructField(list2, 1)), retLenPtr)
	// ret->cap = list1->cap
	c.cbb.NewStore(c.loadStructField(list1, 2), retCapPtr)

	c.createWhile(func() value.Value {
		return c.cbb.NewICmp(enum.IPredSLT, c.loadStructField(ret, 2), c.loadStructField(ret, 1))
	}, func() {
		c.cbb.NewStore(retCapPtr, c.growCapacity(c.loadStructField(ret, 2)))
	})

	new_arr := c.growArr(c.loadStructField(list1, 0), c.loadStructField(list1, 2), c.loadStructField(ret, 2))
	c.cbb.NewStore(new_arr, c.indexStruct(ret, 0))

	if listType.elementType.IsPrimitive() {
		c.memcpy(c.indexArray(new_arr, c.loadStructField(list1, 1)), c.loadStructField(list2, 0), c.loadStructField(list2, 1))
	} else {
		list1Len, list2Len := c.loadStructField(list1, 1), c.loadStructField(list2, 1)
		list2Arr := c.loadStructField(list2, 0)
		c.createFor(func() value.Value { return zero }, func() value.Value { return list2Len },
			func(index value.Value) {
				elementPtr := c.indexArray(new_arr, c.cbb.NewAdd(list1Len, index))
				c.cbb.NewCall(listType.elementType.DeepCopyFunc(), elementPtr, c.loadArrayElement(list2Arr, index))
			},
		)
	}

	c.cbb.NewStore(constant.NewNull(ptr(listType.elementType.IrType())), c.indexStruct(list1, 0))
	c.cbb.NewStore(zero, c.indexStruct(list1, 1))
	c.cbb.NewStore(zero, c.indexStruct(list1, 2))

	c.cbb, c.cf = cbb, cf // restore the basic block and ir function

	c.insertFunction(concListList.Name(), nil, concListList)
	return concListList, nil, nil, nil
}
