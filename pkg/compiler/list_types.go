/*
This file defines functions to define
and work with lists
*/
package compiler

import (
	"fmt"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/types"
)

func (c *Compiler) defineListType(name string, elementType types.Type) types.Type {
	listType := c.mod.NewTypeDef(name, types.NewStruct(
		ptr(elementType), // underlying array
		ddpint,           // length
		ddpint,           // capacity
	))

	return listType
}

func (c *Compiler) defineFromConstants(elementTypeName string, listType types.Type) *ir.Func {
	irFunc := c.mod.NewFunc(
		fmt.Sprintf("_ddp_%s_from_constants", elementTypeName),
	)
}
