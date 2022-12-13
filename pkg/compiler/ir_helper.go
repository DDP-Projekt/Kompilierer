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

// calculates the size of the given type
// and returns it as i64
func (c *Compiler) sizeof(typ types.Type) value.Value {
	size_ptr := c.cbb.NewGetElementPtr(typ, constant.NewNull(ptr(typ)), newIntT(i32, 1))
	size_i := c.cbb.NewPtrToInt(size_ptr, i64)
	return size_i
}
