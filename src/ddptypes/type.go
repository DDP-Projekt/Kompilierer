package ddptypes

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

// checks wether t1 equals t2,
// that is, wether t1 and t2 refer to the same type
// throughout TypeAliases but not TypeDefs
func Equal(t1, t2 Type) bool {
	return GetUnderlying(t1) == GetUnderlying(t2)
}

// returns the underlying type for nested TypeAliases
func GetUnderlying(t Type) Type {
	if alias, ok := t.(*TypeAlias); ok {
		return GetUnderlying(alias.Underlying)
	}
	return t
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
	return t == ZAHL || t == KOMMAZAHL
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

func IsTypedef(t Type) bool {
	_, ok := GetUnderlying(t).(*TypeDef)
	return ok
}

func CastTypedef(t Type) (*TypeDef, bool) {
	typeDef, ok := GetUnderlying(t).(*TypeDef)
	return typeDef, ok
}
