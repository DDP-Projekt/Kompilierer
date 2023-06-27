package ddptypes

// enum Type for the grammatical gender of a type
type GrammaticalGender int

const (
	INVALID  GrammaticalGender = -1
	MASKULIN                   = iota
	FEMININ
	NEUTRUM
)

// holds information about a DDP-Type
type Type interface {
	// the grammatical Gender of the Type
	Gender() GrammaticalGender
	// string representation of the type
	String() string
}

// helper functions

func IsPrimitive(t Type) bool {
	_, ok := t.(PrimitiveType)
	return ok
}

func IsNumeric(t Type) bool {
	return t == ZAHL || t == KOMMAZAHL
}

func IsList(t Type) bool {
	_, ok := t.(ListType)
	return ok
}

func IsVoid(t Type) bool {
	_, ok := t.(VoidType)
	return ok
}
