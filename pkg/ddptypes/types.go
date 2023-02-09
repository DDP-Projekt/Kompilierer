package ddptypes

import "fmt"

// enum type for primitive types (+ void)
type PrimitiveType int

const (
	ILLEGAL   = -1   // illegal type to indicate errors
	NICHTS    = iota // void
	ZAHL             // int64
	KOMMAZAHL        // float64
	BOOLEAN          // bool
	BUCHSTABE        // int32
	TEXT             // string
)

func (p PrimitiveType) String() string {
	switch p {
	case NICHTS:
		return "nichts"
	case ZAHL:
		return "Zahl"
	case KOMMAZAHL:
		return "Kommazahl"
	case BOOLEAN:
		return "Boolean"
	case BUCHSTABE:
		return "Buchstabe"
	case TEXT:
		return "Text"
	}
	panic(fmt.Sprintf("invalid type (%d)", p))
}

// holds information about a DDP-Type
type Type struct {
	// the primitive type of the DDPType (element type for lists)
	Primitive PrimitiveType
	// if the DDPType is a list
	IsList bool
}

func (ddpType Type) String() string {
	if ddpType.IsList {
		switch ddpType.Primitive {
		case NICHTS:
			panic("invalid list type (void list)")
		case ZAHL:
			return "Zahlen Liste"
		case KOMMAZAHL:
			return "Kommazahlen Liste"
		case BOOLEAN:
			return "Boolean Liste"
		case BUCHSTABE:
			return "Buchstaben Liste"
		case TEXT:
			return "Text Liste"
		default:
			panic(fmt.Sprintf("invalid primitive type (%d)", ddpType.Primitive))
		}
	}
	return ddpType.Primitive.String()
}

// wether or not the type is a primitive type
func (ddpType Type) IsPrimitive() bool {
	return !ddpType.IsList && ddpType.Primitive != NICHTS
}

// wether or not the type is a numeric primitive type
// (ZAHL or KOMMAZAHL)
func (ddpType Type) IsNumeric() bool {
	return ddpType.IsPrimitive() && (ddpType.Primitive == ZAHL || ddpType.Primitive == KOMMAZAHL)
}

// create a new DDPType from a tokenType
func Primitive(primitiveType PrimitiveType) Type {
	return Type{Primitive: primitiveType, IsList: false}
}

// create a new DDPType from a element type
func List(elementType PrimitiveType) Type {
	return Type{Primitive: elementType, IsList: true}
}

/* Helper functions for primitive constants */

// unexported ddp-types returned by the helper functions (go has no const struct-literals)
var (
	ddpVoidType   = Type{Primitive: NICHTS, IsList: false}
	ddpIntType    = Type{Primitive: ZAHL, IsList: false}
	ddpFloatType  = Type{Primitive: KOMMAZAHL, IsList: false}
	ddpBoolType   = Type{Primitive: BOOLEAN, IsList: false}
	ddpCharType   = Type{Primitive: BUCHSTABE, IsList: false}
	ddpStringType = Type{Primitive: TEXT, IsList: false}
)

func Illegal() Type {
	return Primitive(ILLEGAL)
}

// this function serves as constant variable for a void type
func Void() Type {
	return ddpVoidType
}

// this function serves as constant variable for a int type
func Int() Type {
	return ddpIntType
}

// this function serves as constant variable for a float type
func Float() Type {
	return ddpFloatType
}

// this function serves as constant variable for a bool type
func Bool() Type {
	return ddpBoolType
}

// this function serves as constant variable for a char type
func Char() Type {
	return ddpCharType
}

// this function serves as constant variable for a string type
func String() Type {
	return ddpStringType
}

// represents the type of a function parameter which may be a reference
type ParameterType struct {
	Type        Type
	IsReference bool
}

func (paramType ParameterType) String() string {
	if !paramType.IsReference {
		return paramType.Type.String()
	}
	if paramType.Type.IsList {
		switch paramType.Type.Primitive {
		case ZAHL, KOMMAZAHL, BOOLEAN, BUCHSTABE, TEXT:
			return paramType.Type.String() + "en Referenz"
		}
	}
	switch paramType.Type.Primitive {
	case ZAHL, KOMMAZAHL:
		return paramType.Type.String() + "en Referenz"
	case BUCHSTABE:
		return "Buchstaben Referenz"
	case BOOLEAN, TEXT:
		return paramType.Type.String() + " Referenz"
	}
	panic(fmt.Sprintf("invalid reference type (%d)", paramType.Type.Primitive))
}
