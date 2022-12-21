/*
This file defines functions to define
and work with lists
*/
package compiler

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
)

// defines the struct of a list type
// and all the necessery functions
// from the given elementType and name
func (c *Compiler) defineListType(name string, elementType types.Type) types.Type {
	listType := c.mod.NewTypeDef(name, types.NewStruct(
		ptr(elementType), // underlying array
		ddpint,           // length
		ddpint,           // capacity
	))

	c.defineFromConstants(listType)
	c.defineFree(listType)
	c.defineDeepCopy(listType)

	return listType
}

// returns the elementType of the given listType
// listType must have been declared using defineListType
func getListElementType(listType types.Type) types.Type {
	return listType.(*types.StructType).Fields[0].(*types.PointerType).ElemType
}

// defines the _ddp_x_from_constants function for a listType
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

	cbb := c.cbb

	// start block
	c.cbb = irFunc.NewBlock("")
	comp := c.cbb.NewICmp(enum.IPredSGT, count, zero)
	trueLabel, falseLabel, endBlock := irFunc.NewBlock(""), irFunc.NewBlock(""), ir.NewBlock("")
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
	result := c.cbb.NewPhi(ir.NewIncoming(arr, trueLabel), ir.NewIncoming(constant.NewNull(ptr(elementType)), falseLabel))

	// get pointers to the struct fields
	retArr, retLen, retCap := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)

	// assign the fields
	c.cbb.NewStore(result, retArr)
	c.cbb.NewStore(count, retLen)
	c.cbb.NewStore(count, retCap)

	c.cbb.NewRet(nil)

	c.cbb = cbb

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

// defines the _ddp_free_x function for a listType
func (c *Compiler) defineFree(listType types.Type) *ir.Func {
	list := ir.NewParam("list", ptr(listType))

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_free_%s", listType.Name()),
		void,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb := c.cbb

	c.cbb = irFunc.NewBlock("")
	c.freeArr(c.loadStructField(list, 0), c.loadStructField(list, 2))
	c.cbb.NewRet(nil)

	c.cbb = cbb

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}

// defines the _ddp_deep_copy_x function for a listType
func (c *Compiler) defineDeepCopy(listType types.Type) *ir.Func {
	ret, list := ir.NewParam("ret", ptr(listType)), ir.NewParam("list", ptr(listType))

	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_deep_copy_%s", listType.Name()),
		void,
		ret,
		list,
	)
	irFunc.CallingConv = enum.CallingConvC

	cbb := c.cbb

	arrFieldPtr, lenFieldPtr, capFieldPtr := c.indexStruct(ret, 0), c.indexStruct(ret, 1), c.indexStruct(ret, 2)
	origArr, origLen, origCap := c.loadStructField(list, 0), c.loadStructField(list, 1), c.loadStructField(list, 2)

	arr := c.allocateArr(getPointeeTypeT(getPointeeType(arrFieldPtr)), origCap)
	c.memcpyArr(arr, origArr, origLen)

	// TODO: consider types that need to be deep copied (strings)

	c.cbb.NewStore(arr, arrFieldPtr)
	c.cbb.NewStore(lenFieldPtr, origLen)
	c.cbb.NewStore(capFieldPtr, origCap)

	c.cbb = cbb

	c.insertFunction(irFunc.Name(), nil, irFunc)
	return irFunc
}
