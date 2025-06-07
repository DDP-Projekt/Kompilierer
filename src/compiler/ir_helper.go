/*
This file defines often used utility functions
to generate ir
*/
package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// calculates the size of the given type
// and returns it as i64
func (c *compiler) sizeof(typ llvm.Type) llvm.Value {
	size_ptr := c.builder().CreateGEP(typ, c.Null, []llvm.Value{c.newIntT(c.i32, 1)}, "")
	return c.builder().CreatePtrToInt(size_ptr, c.i64, "")
}

func (c *compiler) floatOrByteAsInt(src llvm.Value, from ddpIrType) llvm.Value {
	switch from {
	case c.ddpinttyp:
		return src
	case c.ddpfloattyp:
		return c.builder().CreateFPToSI(src, c.ddpint, "")
	case c.ddpbytetyp:
		return c.builder().CreateZExt(src, c.ddpint, "")
	default:
		panic("non numeric type passed to floatOrByteAsInt")
	}
}

func (c *compiler) intOrByteAsFloat(src llvm.Value, from ddpIrType) llvm.Value {
	switch from {
	case c.ddpinttyp:
		return c.builder().CreateSIToFP(src, c.ddpfloat, "")
	case c.ddpfloattyp:
		return src
	case c.ddpbytetyp:
		return c.builder().CreateUIToFP(src, c.ddpfloat, "")
	default:
		panic("non numeric type passed to intOrByteAsFloat")
	}
}

func (c *compiler) intOrFloatAsByte(src llvm.Value, from ddpIrType) llvm.Value {
	switch from {
	case c.ddpinttyp:
		return c.builder().CreateTrunc(src, c.ddpbyte, "")
	case c.ddpfloattyp:
		return c.builder().CreateFPToUI(src, c.ddpbyte, "")
	case c.ddpbytetyp:
		return src
	default:
		panic("non numeric type passed to intOrFloatAsByte")
	}
}

func (c *compiler) numericCast(src llvm.Value, from, to ddpIrType) llvm.Value {
	switch to {
	case c.ddpinttyp:
		return c.floatOrByteAsInt(src, from)
	case c.ddpfloattyp:
		return c.intOrByteAsFloat(src, from)
	case c.ddpbytetyp:
		return c.intOrFloatAsByte(src, from)
	default:
		panic("non numeric type passed to numericCast")
	}
}

// the GROW_CAPACITY macro from the runtime
func (c *compiler) growCapacity(capa llvm.Value) llvm.Value {
	trueBlock, falseBlock, endBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	cond := c.builder().CreateICmp(llvm.IntSLT, capa, c.newInt(8), "")
	c.builder().CreateCondBr(cond, trueBlock, falseBlock)

	c.builder().setBlock(trueBlock)
	c.builder().CreateBr(endBlock)

	c.builder().setBlock(falseBlock)
	newCap := c.builder().CreateFPToSI(c.builder().CreateFMul(c.builder().CreateSIToFP(capa, c.ddpfloat, ""), c.newFloat(1.5), ""), c.i64, "")
	c.builder().CreateBr(endBlock)

	c.builder().setBlock(endBlock)
	result := c.builder().CreatePHI(c.ddpint, "")
	result.AddIncoming([]llvm.Value{c.newInt(8), newCap}, []llvm.BasicBlock{trueBlock, falseBlock})
	return result
}

// uses the GetElementPtr instruction to index a pointer
// returns a pointer to the value
func (c *compiler) indexArray(elementType llvm.Type, arr llvm.Value, index llvm.Value) llvm.Value {
	gep := c.builder().CreateInBoundsGEP(elementType, arr, []llvm.Value{index}, "")
	return gep
}

func (c *compiler) loadArrayElement(elementType llvm.Type, arr llvm.Value, index llvm.Value) llvm.Value {
	elementPtr := c.indexArray(elementType, arr, index)
	return c.builder().CreateLoad(elementType, elementPtr, "")
}

// uses the GetElementPtr instruction to index struct fields
// returns a pointer to the field
func (c *compiler) indexStruct(structType llvm.Type, structPtr llvm.Value, index int) llvm.Value {
	return c.builder().CreateGEP(structType, structPtr, []llvm.Value{c.newIntT(c.i32, 0), c.newIntT(c.i32, int64(index))}, "")
}

// indexStruct followed by a load on the result
// returns the value of the field
func (c *compiler) loadStructField(structType llvm.Type, structPtr llvm.Value, index int) llvm.Value {
	fieldPtr := c.indexStruct(structType, structPtr, index)
	return c.builder().CreateLoad(structType.StructElementTypeAtIndex(index), fieldPtr, "")
}

// generates a new if-else statement
// cond is the condition
// genTrueBody generates the then-body
// genFalseBody may be nil if no else is required
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *compiler) createIfElse(cond llvm.Value, genTrueBody, genFalseBody func()) {
	trueBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock()
	falseBlock := leaveBlock
	if genFalseBody != nil {
		// to keep the order of blocks in the ir correct
		leaveBlock = c.builder().newBlock()
	}
	c.builder().CreateCondBr(cond, trueBlock, falseBlock)

	c.builder().setBlock(trueBlock)
	genTrueBody()
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(leaveBlock)
	}

	c.builder().setBlock(falseBlock)
	if genFalseBody != nil {
		genFalseBody()
		if c.builder().cb.Terminator().IsNil() {
			c.builder().CreateBr(leaveBlock)
		}
	}

	c.builder().setBlock(leaveBlock)
}

// generates a new ternary-operator expression using phi-nodes
// cond is the condition, true/falseVal should produce values of the same type
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *compiler) createTernary(resultType llvm.Type, cond llvm.Value, trueVal, falseVal func() llvm.Value) llvm.Value {
	trueLabel, falseLabel, endBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().CreateCondBr(cond, trueLabel, falseLabel)

	// cond == true
	c.builder().setBlock(trueLabel)
	trVal := trueVal()
	c.builder().CreateBr(endBlock)

	// cond == false
	c.builder().setBlock(falseLabel)
	falVal := falseVal()
	c.builder().CreateBr(endBlock)

	// phi based on cond
	c.builder().setBlock(endBlock)
	result := c.builder().CreatePHI(resultType, "")
	result.AddIncoming([]llvm.Value{trVal, falVal}, []llvm.BasicBlock{trueLabel, falseLabel})
	return result
}

// generates a new while-loop using cond as condition
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *compiler) createWhile(cond func() llvm.Value, genBody func()) {
	condBlock, bodyBlock, leaveBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(condBlock)
	c.builder().CreateCondBr(cond(), bodyBlock, leaveBlock)

	c.builder().setBlock(bodyBlock)
	genBody()
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(condBlock)
	}

	c.builder().setBlock(leaveBlock)
}

// generates a new for-loop using iterStart/End to get the start and end value (should return i64)
// and genBody to generate what should be done in the body-Block
// c.cbb and c.cf must be set/restored correctly by the caller
func (c *compiler) createFor(iterStart llvm.Value, genCond func(index llvm.Value) llvm.Value, genBody func(index llvm.Value)) {
	// initialize counter to 0 (ddpint counter = 0)
	counter := c.NewAlloca(c.i64)
	c.builder().CreateStore(iterStart, counter)

	// initialize the 4 blocks
	condBlock, bodyBlock, incrBlock, endBlock := c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock(), c.builder().newBlock()
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(condBlock)
	current_count := c.builder().CreateLoad(c.i64, counter, "")
	cond := genCond(current_count) // check the condition (counter < list->len)
	c.builder().CreateCondBr(cond, bodyBlock, endBlock)

	// arr[counter] = _ddp_deep_copy_x(list->arr[counter])
	c.builder().setBlock(bodyBlock)
	genBody(c.builder().CreateLoad(c.i64, counter, ""))
	if c.builder().cb.Terminator().IsNil() {
		c.builder().CreateBr(incrBlock)
	}

	// counter++
	c.builder().setBlock(incrBlock)
	c.builder().CreateStore(c.builder().CreateAdd(c.newInt(1), c.builder().CreateLoad(c.i64, counter, ""), ""), counter)
	c.builder().CreateBr(condBlock)

	c.builder().setBlock(endBlock)
}

// helper to be used together with createFor
// creates a default condition to loop up to a certain value
func (c *compiler) forDefaultCond(limit llvm.Value) func(llvm.Value) llvm.Value {
	return func(index llvm.Value) llvm.Value {
		return c.builder().CreateICmp(llvm.IntSLT, index, limit, "")
	}
}
