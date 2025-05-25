package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

// holds the type of a primitive ddptype (ddpint, ddpfloat, ddpbool, ddpchar)
type ddpIrStructType struct {
	typ           llvm.Type
	fieldIrTypes  []ddpIrType
	fieldDDPTypes []ddptypes.StructField
	name          string
	vtable        llvm.Value
	defaultValue  llvm.Value
	freeIrFun     llvm.Value // the free ir func
	deepCopyIrFun llvm.Value // the deepCopy ir func
	equalsIrFun   llvm.Value // the equals ir func
	listType      *ddpIrListType
}

var _ ddpIrType = (*ddpIrStructType)(nil)

func (t *ddpIrStructType) LLType() llvm.Type {
	return t.typ
}

func (t *ddpIrStructType) Name() string {
	return t.name
}

func (*ddpIrStructType) IsPrimitive() bool {
	return false
}

func (t *ddpIrStructType) DefaultValue() llvm.Value {
	return t.defaultValue
}

func (t *ddpIrStructType) VTable() llvm.Value {
	return t.vtable
}

func (t *ddpIrStructType) FreeFunc() llvm.Value {
	return t.freeIrFun
}

func (t *ddpIrStructType) DeepCopyFunc() llvm.Value {
	return t.deepCopyIrFun
}

func (t *ddpIrStructType) EqualsFunc() llvm.Value {
	return t.equalsIrFun
}

func (c *compiler) defineOrDeclareAllDeclTypes(decl *ast.StructDecl) {
	switch typ := decl.Type.(type) {
	case *ddptypes.StructType:
		c.defineOrDeclareStructType(typ)
	case *ddptypes.GenericStructType:
		for _, instantiation := range typ.Instantiations {
			c.defineOrDeclareStructType(instantiation)
		}
	default:
		c.err("unexpected type %s in StructDecl %s", typ, decl.Name())
	}
}

// recursively defines (or declares, if not from this module) a struct type and all it's field types
func (c *compiler) defineOrDeclareStructType(typ *ddptypes.StructType) {
	// if the struct type is already defined, don't define/declare it again
	if _, exists := c.structTypes[typ]; exists {
		return
	}

	// not fully instantiated types (i.e. from generic function decls) are not needed
	if _, hasGenericTypes := ddptypes.CastDeeplyNestedGenerics(typ); hasGenericTypes {
		c.structTypes[typ] = nil
		return
	}

	name := c.mangledNameType(typ)
	// if the type comes from outside the current module
	// we only declare it's functions as they are defined in it's own module
	declarationOnly := c.typeMap[typ] != c.ddpModule

	structType := &ddpIrStructType{}
	structType.name = name
	// recursively declare all types this type depends on
	structType.fieldIrTypes = mapSlice(typ.Fields, func(field ddptypes.StructField) ddpIrType {
		if fieldStructType, isStruct := ddptypes.CastStruct(ddptypes.ListTrueUnderlying(field.Type)); isStruct {
			c.defineOrDeclareStructType(fieldStructType)
		}

		return c.toIrType(field.Type)
	})
	structType.fieldDDPTypes = typ.Fields

	structType.typ = c.llctx.StructType(
		mapSlice(structType.fieldIrTypes, func(t ddpIrType) llvm.Type { return t.LLType() }),
		false)

	structType.freeIrFun = c.createStructFree(structType, declarationOnly)
	structType.deepCopyIrFun = c.createStructDeepCopy(structType, declarationOnly)
	structType.equalsIrFun = c.createStructEquals(structType, declarationOnly)

	structType.listType = c.createListType("ddp"+structType.name+"list", structType, declarationOnly)

	vtable := llvm.AddGlobal(c.llmod, c.ptr, name+"_vtable")
	vtable.SetLinkage(llvm.ExternalLinkage)
	vtable.SetVisibility(llvm.DefaultVisibility)

	if !declarationOnly {
		vtable.SetInitializer(llvm.ConstStruct([]llvm.Value{
			llvm.ConstInt(c.ddpint, c.getTypeSize(structType), false),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[0]),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[1]),
			llvm.ConstNull(c.vtable_type.StructElementTypes()[2]),
		}, false))
	}

	structType.vtable = vtable

	structType.defaultValue = llvm.ConstNull(structType.typ)

	c.structTypes[typ] = structType
}

func (c *compiler) createStructFree(structTyp *ddpIrStructType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_free_"+structTyp.name, llvm.FunctionType(c.voidtyp.LLType(), []llvm.Type{c.ptr}, false))
	defer c.disposeAndPop()

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	structParam := llFuncBuilder.llFn.Param(0)

	// free non-primitives
	for i, field := range structTyp.fieldIrTypes {
		c.freeNonPrimitive(c.indexStruct(structTyp.typ, structParam, i), field)
	}

	c.builder().CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn)
	return llFuncBuilder.llFn
}

func (c *compiler) createStructDeepCopy(structTyp *ddpIrStructType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_deep_copy_"+structTyp.name, llvm.FunctionType(c.void, []llvm.Type{c.ptr, c.ptr}, false))
	defer c.disposeAndPop()

	ret, structParam := llFuncBuilder.llFn.Param(0), llFuncBuilder.llFn.Param(1)

	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	// deep-copy non-primitives
	for i, field := range structTyp.fieldIrTypes {
		dstPtr := c.indexStruct(structTyp.typ, ret, i)
		if !field.IsPrimitive() {
			srcPtr := c.indexStruct(structTyp.typ, structParam, i)
			c.deepCopyInto(dstPtr, srcPtr, field)
		} else {
			llFuncBuilder.CreateStore(c.loadStructField(structTyp.typ, structParam, i), dstPtr)
		}
	}

	llFuncBuilder.CreateRet(llvm.Value{})

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn)
	return llFuncBuilder.llFn
}

func (c *compiler) createStructEquals(structTyp *ddpIrStructType, declarationOnly bool) llvm.Value {
	llFuncBuilder := c.newBuilder("ddp_"+structTyp.name+"_equal", llvm.FunctionType(c.ddpbool, []llvm.Type{c.ptr, c.ptr}, false))
	defer c.disposeAndPop()

	struct1, struct2 := llFuncBuilder.llFn.Param(0), llFuncBuilder.llFn.Param(1)
	if declarationOnly {
		llFuncBuilder.llFn.SetLinkage(llvm.ExternalLinkage)
		return llFuncBuilder.llFn
	}

	// if (struct1 == struct2) return true;
	ptrs_equal := llFuncBuilder.CreateICmp(llvm.IntEQ, struct1, struct2, "")
	c.createIfElse(ptrs_equal, func() {
		c.builder().CreateRet(c.True)
	},
		nil,
	)

	// compare every single field and return if one is not equal
	for i, field := range structTyp.fieldIrTypes {
		var f1, f2 llvm.Value
		if field.IsPrimitive() {
			f1, f2 = c.loadStructField(structTyp.typ, struct1, i), c.loadStructField(structTyp.typ, struct2, i)
		} else {
			f1, f2 = c.indexStruct(structTyp.typ, struct1, i), c.indexStruct(structTyp.typ, struct2, i)
		}
		equal := c.compare_values(f1, f2, field)
		is_not_equal := c.builder().CreateXor(equal, c.newInt(1), "")
		c.createIfElse(is_not_equal, func() {
			c.builder().CreateRet(c.False)
		}, nil)
	}

	c.builder().CreateRet(c.True)

	c.insertFunction(llFuncBuilder.fnName, nil, llFuncBuilder.llFn)
	return llFuncBuilder.llFn
}

func mapSlice[T, U any](s []T, mapper func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}

func getFieldIndex(fieldName string, typ *ddpIrStructType) int64 {
	for i, field := range typ.fieldDDPTypes {
		if field.Name == fieldName {
			return int64(i)
		}
	}
	return -1
}
