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
	typ                types.Type // the typedef
	elementType        ddpIrType  // the ir-type of the list elements
	fromConstantsIrFun *ir.Func   // the fromConstans ir func
	freeIrFun          *ir.Func   // the free ir func
	deepCopyIrFun      *ir.Func   // the deepCopy ir func
	equalsIrFun        *ir.Func   // the equals ir func
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

	cbb := c.cbb // save the current block

	// start block
	c.cbb = irFunc.NewBlock("")
	comp := c.cbb.NewICmp(enum.IPredSGT, count, zero) // count > 0
	trueLabel, falseLabel, endBlock := irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock("")
	c.cbb.NewCondBr(comp, trueLabel, falseLabel)

	// count > 0 -> allocate the array
	c.cbb = trueLabel
	arr := c.allocateArr(elementType, count)
	c.cbb.NewBr(endBlock)

	// count <= 0 -> do nothing for the phi
	c.cbb = falseLabel
	c.cbb.NewBr(endBlock)

	// phi based on count
	c.cbb = endBlock
	result := c.cbb.NewLoad(arrType, c.cbb.NewPhi(ir.NewIncoming(arr, trueLabel), ir.NewIncoming(constant.NewNull(arrType), falseLabel)))

	// get pointers to the struct fields
	retArr, retLen, retCap := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)

	// assign the fields
	c.cbb.NewStore(result, retArr)
	c.cbb.NewStore(count, retLen)
	c.cbb.NewStore(count, retCap)

	c.cbb.NewRet(nil)

	c.cbb = cbb // restore the block

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

	cbb := c.cbb // save the current block

	c.cbb = irFunc.NewBlock("")

	if listType.elementType.IsPrimitive() {
		listArr, listCap := c.loadStructField(list, 0), c.loadStructField(list, 2)
		c.freeArr(listArr, listCap)
	} else {
		listArr, listLen := c.loadStructField(list, 0), c.loadStructField(list, 1)

		// initialize counter to 0 (ddpint counter = 0)
		counter := c.cbb.NewAlloca(i64)
		c.cbb.NewStore(zero, counter)

		// initialize the 4 blocks
		condBlock, bodyBlock, incrBlock, endBlock := irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock("")
		c.cbb.NewBr(condBlock)

		c.cbb = condBlock
		cond := c.cbb.NewICmp(enum.IPredSLT, c.cbb.NewLoad(counter.ElemType, counter), listLen) // check the condition (counter < list->len)
		c.cbb.NewCondBr(cond, bodyBlock, endBlock)

		// _ddp_free_x(list->arr[counter])
		c.cbb = bodyBlock
		currentCount := c.cbb.NewLoad(counter.ElemType, counter)
		val := c.loadArrayElement(listArr, currentCount)
		c.cbb.NewCall(listType.elementType.FreeFunc(), val)
		c.cbb.NewBr(incrBlock)

		// counter++
		c.cbb = incrBlock
		c.cbb.NewStore(c.cbb.NewAdd(newInt(1), currentCount), counter)
		c.cbb.NewBr(condBlock)

		c.cbb = endBlock
	}

	c.cbb.NewRet(nil)

	c.cbb = cbb // restore the block

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

	cbb := c.cbb // save the current block

	c.cbb = irFunc.NewBlock("")
	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)
	origArr, origLen, origCap := c.loadStructField(list, 0), c.loadStructField(list, 1), c.loadStructField(list, 2)

	arr := c.allocateArr(getPointeeTypeT(getPointeeType(arrFieldPtr)), origCap)

	// primitive types can easily be copied
	if listType.elementType.IsPrimitive() {
		c.memcpyArr(arr, origArr, origLen)
	} else { // non-primitive types need to be seperately deep-copied
		// initialize counter to 0 (ddpint counter = 0)
		counter := c.cbb.NewAlloca(i64)
		c.cbb.NewStore(zero, counter)

		// initialize the 4 blocks
		condBlock, bodyBlock, incrBlock, endBlock := irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock("")
		c.cbb.NewBr(condBlock)

		c.cbb = condBlock
		cond := c.cbb.NewICmp(enum.IPredSLT, c.cbb.NewLoad(counter.ElemType, counter), origLen) // check the condition (counter < list->len)
		c.cbb.NewCondBr(cond, bodyBlock, endBlock)

		// arr[counter] = _ddp_deep_copy_x(list->arr[counter])
		c.cbb = bodyBlock
		currentCount := c.cbb.NewLoad(counter.ElemType, counter)
		oldVal := c.loadArrayElement(origArr, currentCount)
		deepCopy := c.cbb.NewAlloca(listType.elementType.IrType())
		c.cbb.NewCall(listType.elementType.DeepCopyFunc(), deepCopy, oldVal)
		c.cbb.NewStore(c.cbb.NewLoad(deepCopy.ElemType, deepCopy), c.indexArray(arr, currentCount))
		c.cbb.NewBr(incrBlock)

		// counter++
		c.cbb = incrBlock
		c.cbb.NewStore(c.cbb.NewAdd(newInt(1), currentCount), counter)
		c.cbb.NewBr(condBlock)

		c.cbb = endBlock
	}

	c.cbb.NewStore(arr, arrFieldPtr)     // ret->arr = new_arr
	c.cbb.NewStore(origLen, lenFieldPtr) // ret->len = list->len
	c.cbb.NewStore(origCap, capFieldPtr) // ret->cap = list->cap

	c.cbb.NewRet(nil)

	c.cbb = cbb // restore the block

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

	cbb := c.cbb // save the current block

	c.cbb = irFunc.NewBlock("")

	// if (list1 == list2) return true;
	ptrs_equal_trueBlock, ptrs_equal_leaveBlock := irFunc.NewBlock(""), irFunc.NewBlock("")
	ptrs_equal := c.cbb.NewICmp(enum.IPredEQ, list1, list2)
	c.cbb.NewCondBr(ptrs_equal, ptrs_equal_trueBlock, ptrs_equal_leaveBlock)

	c.cbb = ptrs_equal_trueBlock
	c.cbb.NewRet(constant.True)

	// if (list1->len != list2->len) return false;
	c.cbb = ptrs_equal_leaveBlock
	len_unequal_trueBlock, len_unequal_leaveBlock := irFunc.NewBlock(""), irFunc.NewBlock("")
	list1_len := c.loadStructField(list1, 1)
	len_unequal := c.cbb.NewICmp(enum.IPredNE, list1_len, c.loadStructField(list2, 1))
	c.cbb.NewCondBr(len_unequal, len_unequal_trueBlock, len_unequal_leaveBlock)

	c.cbb = len_unequal_trueBlock
	c.cbb.NewRet(constant.False)

	// compare single elements
	c.cbb = len_unequal_leaveBlock
	// primitive types can easily be compared
	if listType.elementType.IsPrimitive() {
		// return memcmp(list1->arr, list2->arr, sizeof(T) * list1->len);
		size := c.cbb.NewMul(c.sizeof(listType.elementType.IrType()), list1_len)
		c.cbb.NewRet(c.memcmp(c.loadStructField(list1, 0), c.loadStructField(list2, 0), size))
	} else { // non-primitive types need to be seperately compared
		// initialize counter to 0 (ddpint counter = 0)
		counter := c.cbb.NewAlloca(i64)
		c.cbb.NewStore(zero, counter)

		// initialize the 4 blocks
		condBlock, bodyBlock, incrBlock, endBlock := irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock(""), irFunc.NewBlock("")
		c.cbb.NewBr(condBlock)

		c.cbb = condBlock
		cond := c.cbb.NewICmp(enum.IPredSLT, c.cbb.NewLoad(counter.ElemType, counter), list1_len) // check the condition (counter < list1->len)
		c.cbb.NewCondBr(cond, bodyBlock, endBlock)

		// if (!_ddp_T_equal(list1->arr[i], list2->arr[i])) return false;
		c.cbb = bodyBlock
		currentCount := c.cbb.NewLoad(counter.ElemType, counter)
		list1_arr, list2_arr := c.loadStructField(list1, 0), c.loadStructField(list2, 0)
		list1_at_count, list2_at_count := c.loadArrayElement(list1_arr, currentCount), c.loadArrayElement(list2_arr, currentCount)
		elements_unequal := c.cbb.NewXor(c.cbb.NewCall(listType.elementType.EqualsFunc(), list1_at_count, list2_at_count), newInt(1))
		elements_uneq_trueBlock := irFunc.NewBlock("")
		c.cbb.NewCondBr(elements_unequal, elements_uneq_trueBlock, incrBlock)

		c.cbb = elements_uneq_trueBlock
		c.cbb.NewRet(constant.False)

		// counter++
		c.cbb = incrBlock
		c.cbb.NewStore(c.cbb.NewAdd(newInt(1), currentCount), counter)
		c.cbb.NewBr(condBlock)

		c.cbb = endBlock
		c.cbb.NewRet(constant.True)
	}

	c.cbb = cbb // restore the block

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}