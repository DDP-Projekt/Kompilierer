/*
This file defines often used utility functions
to generate ir
*/
package compiler

import (
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// takes a value of pointerType and returns the type it points to
func getPointeeType(ptr value.Value) types.Type {
	return ptr.Type().(*types.PointerType).ElemType
}

// assumes ptr is a types.PointerType and returns it ElementType
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
