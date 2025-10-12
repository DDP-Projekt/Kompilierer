package ddptypes

import (
	"cmp"
	"slices"
)

// enum Type for the grammatical gender of a type
type GrammaticalGender int

const (
	INVALID_GENDER GrammaticalGender = -1
	MASKULIN                         = iota
	FEMININ
	NEUTRUM
)

//go-sumtype:decl Type

// holds information about a DDP-Type
type Type interface {
	// dummy method to make it a sealed type
	ddpType()
	// the grammatical Gender of the Type
	Gender() GrammaticalGender
	// string representation of the type (its name)
	String() string
}

// helper functions

// checks wether t matches any of the provided genders
func MatchesGender(t Type, genders ...GrammaticalGender) bool {
	// generic types match every gender
	if _, ok := t.(GenericType); ok {
		return true
	}

	if _, ok := t.(*InstantiatedGenericType); ok {
		return true
	}

	return slices.Contains(genders, t.Gender())
}

// checks wether t1 equals t2,
// that is, wether t1 and t2 refer to the same type
// throughout TypeAliases but not TypeDefs
func Equal(t1, t2 Type) bool {
	return GetUnderlying(t1) == GetUnderlying(t2)
}

// checks wether t1 equals t2,
// that is, wether t1 and t2 refer to the same type
// throughout TypeAliases and TypeDefs and also List Types
func DeepEqual(t1, t2 Type) bool {
	return getTrueListUnderlying(t1) == getTrueListUnderlying(t2)
}

// returns the underlying type for nested TypeAliases
func GetUnderlying(t Type) Type {
	switch typ := t.(type) {
	case *TypeAlias:
		return GetUnderlying(typ.Underlying)
	case ListType:
		return ListType{ElementType: GetUnderlying(typ.ElementType)}
	case ReferenceType:
		return ReferenceType{Type: GetUnderlying(typ.Type)}
	case *InstantiatedGenericType:
		return GetUnderlying(typ.Actual)
	default:
		return t
	}
}

func IsPrimitive(t Type) bool {
	_, ok := GetUnderlying(t).(PrimitiveType)
	return ok
}

// acts like primitiveType, ok := t.(PrimitiveType)
// but respects TypeAliases
func CastPrimitive(t Type) (PrimitiveType, bool) {
	primitiveType, ok := GetUnderlying(t).(PrimitiveType)
	return primitiveType, ok
}

func IsNumeric(t Type) bool {
	t = GetUnderlying(t)
	return t == ZAHL || t == KOMMAZAHL || t == BYTE
}

func IsList(t Type) bool {
	_, ok := GetUnderlying(t).(ListType)
	return ok
}

// acts like primitiveType, ok := t.(ListType)
// but respects TypeAliases
func CastList(t Type) (ListType, bool) {
	listType, ok := GetUnderlying(t).(ListType)
	return listType, ok
}

func IsVoid(t Type) bool {
	_, ok := GetUnderlying(t).(VoidType)
	return ok
}

func IsPrimitiveOrVoid(t Type) bool {
	return IsPrimitive(t) || IsVoid(t)
}

func IsStruct(t Type) bool {
	_, ok := GetUnderlying(t).(*StructType)
	return ok
}

// acts like primitiveType, ok := t.(*StructType)
// but respects TypeAliases
func CastStruct(t Type) (*StructType, bool) {
	structType, ok := GetUnderlying(t).(*StructType)
	return structType, ok
}

func IsTypeAlias(t Type) bool {
	_, ok := t.(*TypeAlias)
	return ok
}

func CastTypeAlias(t Type) (*TypeAlias, bool) {
	typeDef, ok := GetUnderlying(t).(*TypeAlias)
	return typeDef, ok
}

func IsTypeDef(t Type) bool {
	_, ok := GetUnderlying(t).(*TypeDef)
	return ok
}

func CastTypeDef(t Type) (*TypeDef, bool) {
	typeDef, ok := GetUnderlying(t).(*TypeDef)
	return typeDef, ok
}

func IsAny(t Type) bool {
	_, ok := GetUnderlying(t).(Variable)
	return ok
}

func IsGeneric(t Type) bool {
	_, ok := CastGeneric(t)
	return ok
}

func CastGeneric(t Type) (GenericType, bool) {
	t = GetUnderlying(t)
	generic, ok := t.(GenericType)
	return generic, ok
}

func CastGenericStructType(t Type) (*GenericStructType, bool) {
	t = GetUnderlying(t)
	generic, ok := t.(*GenericStructType)
	return generic, ok
}

func IsReference(t Type) bool {
	_, ok := CastReference(t)
	return ok
}

func CastReference(t Type) (ReferenceType, bool) {
	t = GetUnderlying(t)
	reference, ok := t.(ReferenceType)
	return reference, ok
}

// wether a is a reference to b
func IsReferenceTo(a, b Type) bool {
	aRef, isARef := CastReference(a)
	return isARef && Equal(aRef.Type, b)
}

// gets the underlying type for nested lists and References
// if typ is not a list or reference type typ is returned
func GetNestedType(typ Type) Type {
	typ = GetUnderlying(typ)
	for IsList(typ) || IsReference(typ) {
		switch typ.(type) {
		case ReferenceType:
			typ = GetNestedType(GetUnderlying(typ).(ReferenceType).Type)
		case ListType:
			typ = GetNestedType(GetUnderlying(typ).(ListType).ElementType)
		}
	}
	return typ
}

// if t is a Reference Type, the underlying Type is returned
func Deref(t Type) Type {
	if r, ok := CastReference(t); ok {
		return r.Type
	}
	return t
}

// trys to dereference src to target
func TryDeref(src, target Type) Type {
	srcRef, isSrcRef := CastReference(src)
	targetRef, isTargetRef := CastReference(target)

	if isSrcRef != isTargetRef {
		srcU, targetU := cmp.Or(srcRef.Type, src), cmp.Or(targetRef.Type, target)
		if Equal(srcU, targetU) {
			return srcU
		}
	}

	return src
}

// if only one of the two types is a reference, a deref is attempted
// otherwise the original types are returned
func TryDeref2(a, b Type) (Type, Type) {
	ar, aref := CastReference(a)
	br, bref := CastReference(b)

	if aref != bref {
		au, bu := cmp.Or(ar.Type, a), cmp.Or(br.Type, b)
		if Equal(au, bu) {
			return au, bu
		}
	}

	return a, b
}

// helper function to apply a predicate with TryDeref
func WithDeref[R any](ty, target Type, f func(Type) R) R {
	return f(TryDeref(ty, target))
}

// wether p applies to either t or it's dereferenced type
func MaybeDeref(t Type, p func(Type) bool) bool {
	if t, ok := CastReference(t); ok {
		return p(t.Type)
	}
	return p(t)
}

// wether src can be assigned to dest directly or through an implicit deref of one of the types
// Equal(TryDeref(src, dest), dest)
func EqualDeref(src, dest Type) bool {
	return Equal(TryDeref2(src, dest))
}

func IsListDeref(a Type) bool {
	return MaybeDeref(a, IsList)
}

func IsStructDeref(a Type) bool {
	return MaybeDeref(a, IsStruct)
}

func IsPrimitiveDeref(t Type) bool {
	return MaybeDeref(t, IsPrimitive)
}

func IsNumericDeref(t Type) bool {
	return MaybeDeref(t, IsNumeric)
}

func IsGenericDeref(t Type) bool {
	return MaybeDeref(t, IsGeneric)
}

func CastListDeref(t Type) (ListType, bool) {
	if ref, ok := CastReference(t); ok {
		return CastList(ref.Type)
	}
	return CastList(t)
}

func CastStructDeref(t Type) (*StructType, bool) {
	if ref, ok := CastReference(t); ok {
		return CastStruct(ref.Type)
	}
	return CastStruct(t)
}

func CastTypeDefDeref(t Type) (*TypeDef, bool) {
	if ref, ok := CastReference(t); ok {
		return CastTypeDef(ref.Type)
	}
	return CastTypeDef(t)
}
