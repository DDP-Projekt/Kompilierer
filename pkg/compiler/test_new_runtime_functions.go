/*
test file
will be deleted later
*/
package compiler

import (
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

func (c *Compiler) test_new_runtime_functions() {
	c.initRuntimeFunctions()

	var (
		ddpint    = c.definePrimitiveType(i64, newIntT(i64, 0))
		ddpfloat  = c.definePrimitiveType(types.Double, constant.NewFloat(types.Double, 0.0))
		ddpbool   = c.definePrimitiveType(types.I1, newIntT(types.I1, 0))
		ddpchar   = c.definePrimitiveType(i32, newIntT(i32, 0))
		ddpstring = c.defineStringType()
	)

	var (
		ddpintlist    = c.defineListType("ddpintlist_test", ddpint)
		ddpfloatlist  = c.defineListType("ddpfloatlist_test", ddpfloat)
		ddpboollist   = c.defineListType("ddpboollist_test", ddpbool)
		ddpcharlist   = c.defineListType("ddpcharlist_test", ddpchar)
		ddpstringlist = c.defineListType("ddpstringlist_test", ddpstring)
	)
	_, _, _, _, _ = ddpintlist, ddpfloatlist, ddpboollist, ddpcharlist, ddpstringlist
}
