package compiler

import (
	"DDP/pkg/ast"
	"DDP/pkg/token"
	"fmt"
	"strings"

	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/types"
)

// often used types declared here to shorten their names
var (
	void = types.Void

	i8  = types.I8
	i16 = types.I16
	i32 = types.I32
	i64 = types.I64

	cstr = ptr(ddpchar)

	ddpint    = i64
	ddpfloat  = types.Double
	ddpbool   = types.I1
	ddpchar   = i16
	ddpstring = types.NewStruct()
	ddpstrptr = types.NewPointer(ddpstring)

	ptr = types.NewPointer

	zero  = constant.NewInt(ddpint, 0)
	fzero = newFloat(0)
	one   = constant.NewInt(ddpint, 1)

	VK_STRING = constant.NewInt(i8, 0)
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

func newFloat(value float64) *constant.Float {
	return constant.NewFloat(ddpfloat, value)
}

// turn a tokenType into the corresponding llvm type
func toDDPType(t token.TokenType) types.Type {
	switch t {
	case token.NICHTS:
		return void
	case token.ZAHL:
		return ddpint
	case token.KOMMAZAHL:
		return ddpfloat
	case token.BOOLEAN:
		return ddpbool
	case token.BUCHSTABE:
		return ddpchar
	case token.TEXT:
		return ddpstring
	}
	panic(fmt.Errorf("illegal ddp type to ir type conversion (%s)", t.String()))
}

// check if the function is inbuilt
func isInbuiltFunc(fun *ast.FuncDecl) bool {
	return strings.HasPrefix(fun.Name.Literal, "§")
}
