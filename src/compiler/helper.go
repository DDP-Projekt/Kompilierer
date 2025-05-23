package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	ddpbyte  = types.I8
	ddpbool  = types.I1
	ddpchar  = i32

	ptr = types.NewPointer

	i8ptr = ptr(i8)

	zero      = newInt(0) // 0: i64
	zerof     = constant.NewFloat(ddpfloat, 0)
	zero8     = newIntT(ddpbyte, 0)
	all_ones  = newInt(^0) // int with all bits set to 1
	all_ones8 = newIntT(ddpbyte, ^0)
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
	ddpType = ddptypes.TrueUnderlying(ddpType)
	if listType, isList := ddptypes.CastList(ddpType); isList {
		underlying := ddptypes.TrueUnderlying(listType.ElementType)
		switch underlying {
		case ddptypes.ZAHL:
			return c.ddpintlist
		case ddptypes.KOMMAZAHL:
			return c.ddpfloatlist
		case ddptypes.BYTE:
			return c.ddpbytelist
		case ddptypes.WAHRHEITSWERT:
			return c.ddpboollist
		case ddptypes.BUCHSTABE:
			return c.ddpcharlist
		case ddptypes.TEXT:
			return c.ddpstringlist
		case ddptypes.VARIABLE:
			return c.ddpanylist
		default:
			return c.structTypes[underlying.(*ddptypes.StructType)].listType
		}
	} else {
		switch ddpType {
		case ddptypes.ZAHL:
			return c.ddpinttyp
		case ddptypes.KOMMAZAHL:
			return c.ddpfloattyp
		case ddptypes.BYTE:
			return c.ddpbytetyp
		case ddptypes.WAHRHEITSWERT:
			return c.ddpbooltyp
		case ddptypes.BUCHSTABE:
			return c.ddpchartyp
		case ddptypes.TEXT:
			return c.ddpstring
		case ddptypes.VARIABLE:
			return c.ddpany
		case ddptypes.VoidType{}:
			return c.void
		default: // struct types
			return c.structTypes[ddpType.(*ddptypes.StructType)]
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
	case c.ddpbytetyp:
		return c.ddpbytelist
	case c.ddpbooltyp:
		return c.ddpboollist
	case c.ddpchartyp:
		return c.ddpcharlist
	case c.ddpstring:
		return c.ddpstringlist
	case c.ddpany:
		return c.ddpanylist
	default:
		return ty.(*ddpIrStructType).listType
	}
}

// returns the aligned size of a type
func (c *compiler) getTypeSize(ty ddpIrType) uint64 {
	return c.llTarget.targetData.TypeAllocSize(ty.LLVMType())
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
	hasher                = sha256.New()
	mangledNamesCacheDecl = sync.Map{}
	mangledNamesCacheType = sync.Map{}
)

// returns the mangled name of a declaration
// mangled names are cached and unique per module
// NOTE: think about making this demanglable
func (c *compiler) mangledNameDecl(decl ast.Declaration) string {
	if mangledName, ok := mangledNamesCacheDecl.Load(decl); ok {
		return mangledName.(string)
	}

	declName := decl.Name()
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		// extern functions may not be name-mangled
		if ast.IsExternFunc(decl) || decl.IsExternVisible {
			return decl.Name()
		}
		if ast.IsGenericInstantiation(decl) {
			declName += "_generic_"
			for _, p := range decl.Parameters {
				declName += strings.ReplaceAll(p.Type.Type.String(), " ", "_")
			}
		}
	case *ast.VarDecl:
		if decl.IsExternVisible {
			return decl.Name()
		}
	default:
		// do nothing
	}

	mangledName := mangledNameBase(declName, decl.Module())
	mangledNamesCacheDecl.Store(decl, mangledName)
	return mangledName
}

// returns the mangled name of a struct type
// mangled names are cached and unique per module
// NOTE: think about making this demanglable
func (c *compiler) mangledNameType(t ddptypes.Type) string {
	if mangledName, ok := mangledNamesCacheType.Load(t); ok {
		return mangledName.(string)
	}

	module, ok := c.typeMap[t]
	if !ok {
		panic(fmt.Errorf("type %s not in typeMap", t))
	}

	name := t.String()
	if structType, isStruct := ddptypes.CastStruct(ddptypes.TrueUnderlying(t)); isStruct {
		parent, types := ddptypes.InstantiatedFrom(structType)
		if parent != nil {
			name = strings.Join(mapSlice(types, ddptypes.Type.String), "-") + "-" + structType.String()
		}
	}

	mangledName := mangledNameBase(name, module)
	mangledNamesCacheType.Store(t, mangledName)
	return mangledName
}

// base function for mangledNameType and mangledNameDecl
// should not be called directly
func mangledNameBase(name string, module *ast.Module) string {
	hasher.Reset()
	hasher.Write([]byte(getHashableModuleName(module)))
	mangledName := name + "_mod_" + hex.EncodeToString(hasher.Sum(nil))
	return mangledName
}

// compares two values of same type for equality
func (c *compiler) compare_values(lhs, rhs value.Value, typ ddpIrType) value.Value {
	switch typ {
	case c.ddpinttyp, c.ddpbytetyp, c.ddpbooltyp, c.ddpchartyp:
		c.latestReturn = c.cbb.NewICmp(enum.IPredEQ, lhs, rhs)
	case c.ddpfloattyp:
		c.latestReturn = c.cbb.NewFCmp(enum.FPredOEQ, lhs, rhs)
	default:
		c.latestReturn = c.cbb.NewCall(typ.EqualsFunc(), lhs, rhs)
	}
	c.latestReturnType = c.ddpbooltyp
	return c.latestReturn
}
