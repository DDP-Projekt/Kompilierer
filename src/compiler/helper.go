package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"sync"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"

	"github.com/llir/llvm/ir"
	"github.com/llir/llvm/ir/constant"
	"github.com/llir/llvm/ir/enum"
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// often used types declared here to shorten their names
var (
	i8  = types.I8
	i32 = types.I32
	i64 = types.I64

	// convenience declarations for often used types
	ddpint   = i64
	ddpfloat = types.Double
	ddpbool  = types.I1
	ddpchar  = i32

	ptr = types.NewPointer

	zero  = newInt(0) // 0: i64
	zerof = constant.NewFloat(ddpfloat, 0)
)

func newInt(value int64) *constant.Int {
	return constant.NewInt(ddpint, value)
}

func newIntT(typ *types.IntType, value int64) *constant.Int {
	return constant.NewInt(typ, value)
}

// wrapper for c.cf.Blocks[0].NewAlloca
// because allocatin on c.cbb can cause stackoverflows in loops
func (c *compiler) NewAlloca(elemType types.Type) *ir.InstAlloca {
	return c.cf.Blocks[0].NewAlloca(elemType)
}

// turn a ddptypes.Type into the corresponding llvm type
func (c *compiler) toIrType(ddpType ddptypes.Type) ddpIrType {
	if listType, isList := ddpType.(ddptypes.ListType); isList {
		switch listType.Underlying {
		case ddptypes.ZAHL:
			return c.ddpintlist
		case ddptypes.KOMMAZAHL:
			return c.ddpfloatlist
		case ddptypes.WAHRHEITSWERT:
			return c.ddpboollist
		case ddptypes.BUCHSTABE:
			return c.ddpcharlist
		case ddptypes.TEXT:
			return c.ddpstringlist
		default:
			return c.structTypes[listType.Underlying.String()].listType
		}
	} else {
		switch ddpType {
		case ddptypes.ZAHL:
			return c.ddpinttyp
		case ddptypes.KOMMAZAHL:
			return c.ddpfloattyp
		case ddptypes.WAHRHEITSWERT:
			return c.ddpbooltyp
		case ddptypes.BUCHSTABE:
			return c.ddpchartyp
		case ddptypes.TEXT:
			return c.ddpstring
		case ddptypes.VoidType{}:
			return c.void
		default: // struct types
			return c.structTypes[ddpType.String()]
		}
	}
}

// used to handle possible reference parameters
func (c *compiler) toIrParamType(ty ddptypes.ParameterType) types.Type {
	irType := c.toIrType(ty.Type)

	if !ty.IsReference && irType.IsPrimitive() {
		return irType.IrType()
	}

	return irType.PtrType()
}

func (c *compiler) getListType(ty ddpIrType) *ddpIrListType {
	switch ty {
	case c.ddpinttyp:
		return c.ddpintlist
	case c.ddpfloattyp:
		return c.ddpfloatlist
	case c.ddpbooltyp:
		return c.ddpboollist
	case c.ddpchartyp:
		return c.ddpcharlist
	case c.ddpstring:
		return c.ddpstringlist
	}
	c.err("no list type found for elementType %s", ty.Name())
	return nil // unreachable
}

func getHashableModuleName(mod *ast.Module) string {
	return "ddp_" + strings.TrimSuffix(
		strings.ReplaceAll(
			strings.ReplaceAll(
				filepath.ToSlash(mod.FileName),
				"/",
				"_",
			),
			":",
			"_",
		),
		".ddp",
	)
}

// creates the name of the module_init function
func getModuleInitDisposeName(mod *ast.Module) (string, string) {
	name := getHashableModuleName(mod)
	return name + "_init", name + "_dispose"
}

var (
	hasher            = sha256.New()
	mangledNamesCache = sync.Map{}
)

// NOTE: think about making this demanglable
func mangledName(decl ast.Declaration) string {
	// if mangledName, ok := mangledNamesCache.Load(decl); ok {
	// 	return mangledName.(string)
	// }

	hasher.Reset()
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		// extern functions may not be name-mangled
		if ast.IsExternFunc(decl) || decl.IsExternVisible {
			return decl.Name()
		}
	case *ast.VarDecl:
		if decl.IsExternVisible {
			return decl.Name()
		}
	default:
		return decl.Name()
	}

	hasher.Write([]byte(getHashableModuleName(decl.Module())))
	mangledName := decl.Name() + "_mod_" + hex.EncodeToString(hasher.Sum(nil))
	// mangledNamesCache.Store(decl, mangledName)
	return mangledName
}

// compares two values of same type for equality
func (c *compiler) compare_values(lhs, rhs value.Value, typ ddpIrType) value.Value {
	switch typ {
	case c.ddpinttyp, c.ddpbooltyp, c.ddpchartyp:
		c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
	case c.ddpfloattyp:
		c.latestReturn = c.cbb.NewFCmp(enum.FPredOEQ, lhs, rhs)
	default:
		c.latestReturn = c.cbb.NewCall(typ.EqualsFunc(), lhs, rhs)
	}
	c.latestReturnType = c.ddpbooltyp
	return c.latestReturn
}
