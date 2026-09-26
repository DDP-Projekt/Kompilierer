package compiler

import (
	"debug/dwarf"
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

func (c *compiler) toDIType(ddpType ddptypes.Type) llvm.Metadata {
	if c.diBuilder == nil {
		return llvm.Metadata{}
	}
	return c.toDITypeFromIr(c.toIrType(ddpType))
}

// generic placeholder handling should be handled by the caller
func (c *compiler) toDITypeFromIr(irType ddpIrType) llvm.Metadata {
	if c.diBuilder == nil || irType == nil {
		return llvm.Metadata{}
	}
	if md, ok := c.diTypeCache[irType]; ok {
		return md
	}
	md := c.buildDIType(irType)
	c.diTypeCache[irType] = md
	return md
}

// alignInBits returns the ABI alignment of typ, in bits, for use as a
// DIType's AlignInBits field.
func (c *compiler) alignInBits(typ llvm.Type) uint32 {
	return uint32(c.llTargetData.ABITypeAlignment(typ)) * 8
}

// sizeInBits returns the ABI storage size of typ, in bits.
func (c *compiler) sizeInBits(typ llvm.Type) uint64 {
	return c.llTargetData.TypeAllocSize(typ) * 8
}

func (c *compiler) buildDIType(irType ddpIrType) llvm.Metadata {
	switch t := irType.(type) {
	case *ddpIrVoidType:
		return llvm.Metadata{} // "void" is the absence of a DIType
	case *ddpIrPrimitiveType:
		return c.buildPrimitiveDIType(t)
	case *ddpIrStringType:
		return c.buildStringDIType(t)
	case *ddpIrListType:
		return c.buildListDIType(t.name, t.typ, c.toDITypeFromIr(t.elementType))
	case *ddpIrGenericListType:
		// compiler-internal placeholder used while compiling generic function
		// bodies before instantiation; the element type isn't known here
		return c.buildListDIType(t.Name(), t.typ, llvm.Metadata{})
	case *ddpIrAnyType:
		return c.buildAnyDIType(t)
	case *ddpIrStructType:
		return c.buildStructDIType(t)
	case *ddpIrReferenceType:
		return c.diBuilder.CreatePointerType(llvm.DIPointerType{
			Pointee:     c.toDITypeFromIr(t.underlying),
			SizeInBits:  c.sizeInBits(t.typ),
			AlignInBits: c.alignInBits(t.typ),
		})
	default:
		panic(fmt.Sprintf("buildDIType: unhandled ddpIrType %T", irType))
	}
}

func (c *compiler) buildPrimitiveDIType(t *ddpIrPrimitiveType) llvm.Metadata {
	primType, ok := t.DDPType().(ddptypes.PrimitiveType)
	if !ok {
		panic(fmt.Sprintf("buildPrimitiveDIType: non-primitive ddptypes.Type %s", t.DDPType()))
	}

	var encoding llvm.DwarfTypeEncoding
	switch primType {
	case ddptypes.ZAHL:
		encoding = llvm.DW_ATE_signed
	case ddptypes.KOMMAZAHL:
		encoding = llvm.DW_ATE_float
	case ddptypes.BYTE:
		encoding = llvm.DW_ATE_unsigned_char
	case ddptypes.WAHRHEITSWERT:
		encoding = llvm.DW_ATE_boolean
	case ddptypes.BUCHSTABE:
		encoding = llvm.DW_ATE_UTF
	default:
		panic(fmt.Sprintf("buildPrimitiveDIType: unhandled PrimitiveType %s", primType))
	}

	return c.diBuilder.CreateBasicType(llvm.DIBasicType{
		Name:       t.DDPType().String(),
		SizeInBits: c.sizeInBits(t.llType),
		Encoding:   encoding,
	})
}

func (c *compiler) buildStringDIType(t *ddpIrStringType) llvm.Metadata {
	charType := c.diBuilder.CreateBasicType(llvm.DIBasicType{
		Name: "utf8-Zeichen", SizeInBits: 8, Encoding: llvm.DW_ATE_unsigned_char,
	})
	strPtrType := c.diBuilder.CreatePointerType(llvm.DIPointerType{
		Pointee: charType, SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
	})
	zahlType := c.toDITypeFromIr(c.ddpinttyp)

	members := []llvm.Metadata{
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "str", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
			OffsetInBits: c.llTargetData.ElementOffset(t.typ, string_str_field_index) * 8,
			Type:         strPtrType,
		}),
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "cap", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ddpint), AlignInBits: c.alignInBits(c.ddpint),
			OffsetInBits: c.llTargetData.ElementOffset(t.typ, string_cap_field_index) * 8,
			Type:         zahlType,
		}),
	}

	return c.diBuilder.CreateStructType(c.diCompileUnit, llvm.DIStructType{
		Name: t.DDPType().String(), File: c.diFile,
		SizeInBits: c.sizeInBits(t.typ), AlignInBits: c.alignInBits(t.typ),
		Elements: members, UniqueID: t.Name(),
	})
}

func (c *compiler) buildListDIType(name string, typ llvm.Type, elementDIType llvm.Metadata) llvm.Metadata {
	zahlType := c.toDITypeFromIr(c.ddpinttyp)
	arrType := c.diBuilder.CreatePointerType(llvm.DIPointerType{
		Pointee: elementDIType, SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
	})

	members := []llvm.Metadata{
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "arr", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
			OffsetInBits: c.llTargetData.ElementOffset(typ, list_arr_field_index) * 8,
			Type:         arrType,
		}),
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "len", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ddpint), AlignInBits: c.alignInBits(c.ddpint),
			OffsetInBits: c.llTargetData.ElementOffset(typ, list_len_field_index) * 8,
			Type:         zahlType,
		}),
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "cap", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ddpint), AlignInBits: c.alignInBits(c.ddpint),
			OffsetInBits: c.llTargetData.ElementOffset(typ, list_cap_field_index) * 8,
			Type:         zahlType,
		}),
	}

	return c.diBuilder.CreateStructType(c.diCompileUnit, llvm.DIStructType{
		Name: name, File: c.diFile,
		SizeInBits: c.sizeInBits(typ), AlignInBits: c.alignInBits(typ),
		Elements: members, UniqueID: name,
	})
}

// describes an any as { vtable: void*, wert: [16 x Byte] }
// meaning the runtime is currently not aware of the actual stored type
func (c *compiler) buildAnyDIType(t *ddpIrAnyType) llvm.Metadata {
	voidPtrType := c.diBuilder.CreatePointerType(llvm.DIPointerType{
		SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
	}) // Pointee left zero -> "void*"
	byteType := c.toDITypeFromIr(c.ddpbytetyp)
	byteArrType := c.diBuilder.CreateArrayType(llvm.DIArrayType{
		SizeInBits: 16 * 8, AlignInBits: c.alignInBits(c.i8),
		ElementType: byteType, Subscripts: []llvm.DISubrange{{Lo: 0, Count: 16}},
	})

	members := []llvm.Metadata{
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "vtable", File: c.diFile,
			SizeInBits: c.sizeInBits(c.ptr), AlignInBits: c.alignInBits(c.ptr),
			OffsetInBits: c.llTargetData.ElementOffset(t.typ, any_vtable_ptr_index) * 8,
			Type:         voidPtrType,
		}),
		c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: "wert", File: c.diFile,
			SizeInBits: 16 * 8, AlignInBits: c.alignInBits(c.i8),
			OffsetInBits: c.llTargetData.ElementOffset(t.typ, any_value_index) * 8,
			Type:         byteArrType,
		}),
	}

	return c.diBuilder.CreateStructType(c.diCompileUnit, llvm.DIStructType{
		Name: t.DDPType().String(), File: c.diFile,
		SizeInBits: c.sizeInBits(t.typ), AlignInBits: c.alignInBits(t.typ),
		Elements: members, UniqueID: t.Name(),
	})
}

func (c *compiler) buildStructDIType(t *ddpIrStructType) llvm.Metadata {
	fwdDecl := c.diBuilder.CreateReplaceableCompositeType(c.diCompileUnit, llvm.DIReplaceableCompositeType{
		Tag: dwarf.TagStructType, Name: t.ddpType.String(), File: c.diFile,
		SizeInBits: c.sizeInBits(t.typ), AlignInBits: c.alignInBits(t.typ),
		UniqueID: t.name,
	})
	c.diTypeCache[t] = fwdDecl

	members := make([]llvm.Metadata, len(t.fieldIrTypes))
	for i, field := range t.fieldIrTypes {
		members[i] = c.diBuilder.CreateMemberType(c.diCompileUnit, llvm.DIMemberType{
			Name: t.fieldDDPTypes[i].Name, File: c.diFile,
			SizeInBits:   c.sizeInBits(field.LLType()),
			AlignInBits:  c.alignInBits(field.LLType()),
			OffsetInBits: c.llTargetData.ElementOffset(t.typ, i) * 8,
			Type:         c.toDITypeFromIr(field),
		})
	}

	real := c.diBuilder.CreateStructType(c.diCompileUnit, llvm.DIStructType{
		Name: t.ddpType.String(), File: c.diFile,
		SizeInBits: c.sizeInBits(t.typ), AlignInBits: c.alignInBits(t.typ),
		Elements: members, UniqueID: t.name,
	})
	fwdDecl.ReplaceAllUsesWith(real)
	c.diTypeCache[t] = real
	return real
}

func (c *compiler) diScopeForVar(scp *scope, line int) llvm.Metadata {
	b := c.builder()
	if scp != b.fnScope && scp.diScope.C == nil {
		parent := b.diScopeFor(scp.enclosing)
		scp.diScope = c.diBuilder.CreateLexicalBlock(parent, llvm.DILexicalBlock{File: c.diFile, Line: line})
	}
	return b.diScopeFor(scp)
}

// argNo == 0 means "local variable, or global"
// noop if debug info is disabled
func (c *compiler) emitVarDebugInfo(scp *scope, decl *ast.VarDecl, val llvm.Value, argNo int) {
	if c.diBuilder == nil {
		return
	}

	diType := c.toDIType(decl.Type)
	name := decl.NameTok.Literal
	line := int(decl.GetRange().Start.Line)

	if scp.isGlobalScope() {
		globalExpr := c.diBuilder.CreateGlobalVariableExpression(c.diCompileUnit, llvm.DIGlobalVariableExpression{
			Name: name, LinkageName: c.mangledNameDecl(decl), File: c.diFile, Line: line,
			Type: diType, LocalToUnit: true, Expr: c.diBuilder.CreateExpression(nil),
		})
		val.AddMetadata(c.llctx.MDKindID("dbg"), globalExpr)
		return
	}

	scope := c.diScopeForVar(scp, line)

	var diVar llvm.Metadata
	if argNo > 0 {
		diVar = c.diBuilder.CreateParameterVariable(scope, llvm.DIParameterVariable{
			Name: name, File: c.diFile, Line: line, Type: diType, AlwaysPreserve: true, ArgNo: argNo,
		})
	} else {
		diVar = c.diBuilder.CreateAutoVariable(scope, llvm.DIAutoVariable{
			Name: name, File: c.diFile, Line: line, Type: diType, AlwaysPreserve: true,
		})
	}

	b := c.builder()
	c.diBuilder.InsertDeclareAtEnd(val, diVar, c.diBuilder.CreateExpression(nil),
		llvm.DebugLoc{Line: uint(line), Col: 0, Scope: scope}, b.cb)
}
