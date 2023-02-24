/*
This file defines often used utility functions
to generate ir
*/
package compiler

import (
	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// takes a value of pointerType and returns the type it points to
func getPointeeType(ptr value.Value) types.Type {
	return getPointeeTypeT(ptr.Type())
}

// assumes ptr is a types.PointerType and returns its ElementType
func getPointeeTypeT(ptr types.Type) types.Type {
	return ptr.(*types.PointerType).ElemType
}

// calculates the size of the given type
// and returns it as i64
func (c *Compiler) sizeof(typ types.Type) value.Value {
	size_ptr := c.cbb.NewGetElementPtr(typ, constant.NewNull(ptr(typ)), newIntT(i32, 1))
	size_i := c.cbb.NewPtrToInt(size_ptr, i64)
	return size_i
}

// the GROW_CAPACITY macro from the runtime
func (c *Compiler) growCapacity(cap value.Value) value.Value {
	trueBlock, falseBlock, endBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	cond := c.cbb.NewICmp(enum.IPredSLT, cap, newInt(8))
	c.cbb.NewCondBr(cond, trueBlock, falseBlock)

	c.cbb = trueBlock
	c.cbb.NewBr(endBlock)

	c.cbb = falseBlock
	newCap := c.cbb.NewMul(cap, newInt(2))
	c.cbb.NewBr(endBlock)

	c.cbb = endBlock
	return c.cbb.NewPhi(ir.NewIncoming(newInt(8), trueBlock), ir.NewIncoming(newCap, falseBlock))
}

// uses the GetElementPtr instruction to index a pointer
// returns a pointer to the value
func (c *Compiler) indexArray(arr value.Value, index value.Value) value.Value {
	gep := c.cbb.NewGetElementPtr(getPointeeType(arr), arr, index)
	gep.InBounds = true
	return gep
}

func (c *Compiler) loadArrayElement(arr value.Value, index value.Value) value.Value {
	elementPtr := c.indexArray(arr, index)
	return c.cbb.NewLoad(getPointeeType(arr), elementPtr)
}

// uses the GetElementPtr instruction to index struct fields
// returns a pointer to the field
func (c *Compiler) indexStruct(structPtr value.Value, index int64) value.Value {
	structType := getPointeeType(structPtr)
	return c.cbb.NewGetElementPtr(structType, structPtr, zero32, newIntT(i32, index))
}

// indexStruct followed by a load on the result
// returns the value of the field
func (c *Compiler) loadStructField(structPtr value.Value, index int64) value.Value {
	fieldPtr := c.indexStruct(structPtr, index)
	return c.cbb.NewLoad(getPointeeType(fieldPtr), fieldPtr)
}

// generates a new if-else statement
// cond is the condition
// genTrueBody generates the then-body
// genFalseBody may be nil of no else is required
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *Compiler) createIfElese(cond value.Value, genTrueBody, genFalseBody func()) {
	trueBlock, falseBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewCondBr(cond, trueBlock, falseBlock)

	c.cbb = trueBlock
	genTrueBody()
	c.cbb.NewBr(leaveBlock)

	c.cbb = falseBlock
	if genFalseBody != nil {
		genFalseBody()
	}
	c.cbb.NewBr(leaveBlock)

	c.cbb = leaveBlock
}

// generates a new ternary-operator expression using phi-nodes
// cond is the condition, true/falseVal should produce values of the same type
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *Compiler) createTernary(cond value.Value, trueVal, falseVal func() value.Value) value.Value {
	trueLabel, falseLabel, endBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewCondBr(cond, trueLabel, falseLabel)

	// count > 0 -> allocate the array
	c.cbb = trueLabel
	trVal := trueVal()
	c.cbb.NewBr(endBlock)

	// count <= 0 -> do nothing for the phi
	c.cbb = falseLabel
	falVal := falseVal()
	c.cbb.NewBr(endBlock)

	// phi based on count
	c.cbb = endBlock
	return c.cbb.NewPhi(ir.NewIncoming(trVal, trueLabel), ir.NewIncoming(falVal, falseLabel))
}

// generates a new while-loop using cond as condition
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *Compiler) createWhile(cond func() value.Value, genBody func()) {
	condBlock, bodyBlock, leaveBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewBr(condBlock)

	c.cbb = condBlock
	c.cbb.NewCondBr(cond(), bodyBlock, leaveBlock)

	c.cbb = bodyBlock
	genBody()
	c.cbb.NewBr(condBlock)

	c.cbb = leaveBlock
}

// generates a new for-loop using iterStart/End to get the start and end value (should return i64)
// and genBody to generate what should be done in the body-Block
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *Compiler) createFor(iterStart, iterEnd func() value.Value, genBody func(index value.Value)) {
	// initialize counter to 0 (ddpint counter = 0)
	counter := c.cbb.NewAlloca(i64)
	c.cbb.NewStore(iterStart(), counter)

	// initialize the 4 blocks
	condBlock, bodyBlock, incrBlock, endBlock := c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock(""), c.cf.NewBlock("")
	c.cbb.NewBr(condBlock)

	c.cbb = condBlock
	cond := c.cbb.NewICmp(enum.IPredSLT, c.cbb.NewLoad(counter.ElemType, counter), iterEnd()) // check the condition (counter < list->len)
	c.cbb.NewCondBr(cond, bodyBlock, endBlock)

	// arr[counter] = _ddp_deep_copy_x(list->arr[counter])
	c.cbb = bodyBlock
	genBody(c.cbb.NewLoad(i64, counter))
	c.cbb.NewBr(incrBlock)

	// counter++
	c.cbb = incrBlock
	c.cbb.NewStore(c.cbb.NewAdd(newInt(1), c.cbb.NewLoad(i64, counter)), counter)
	c.cbb.NewBr(condBlock)

	c.cbb = endBlock
}
