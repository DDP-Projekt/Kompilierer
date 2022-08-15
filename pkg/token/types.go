package token

// holds information about a DDP-Type
type DDPType struct {
	// the primitive type of the DDPType (element type for lists)
	// Zahl, Kommazahl, Boolean, Buchstabe, Text
	PrimitiveType TokenType
	// if the DDPType is a list
	IsList bool
}

func (ddpType DDPType) String() string {
	if ddpType.IsList {
		switch ddpType.PrimitiveType {
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
		}
	}
	return ddpType.PrimitiveType.String()
}

// wether or not the type is a primitive type
func (ddpType DDPType) IsPrimitive() bool {
	return !ddpType.IsList && ddpType.PrimitiveType != NICHTS
}

// wether or not the type is a numeric primitive type
// (ZAHL or KOMMAZAHL)
func (ddpType DDPType) IsNumeric() bool {
	return ddpType.IsPrimitive() && (ddpType.PrimitiveType == ZAHL || ddpType.PrimitiveType == KOMMAZAHL)
}

// create a new DDPType from a tokenType
func NewPrimitiveType(primitiveType TokenType) DDPType {
	return DDPType{PrimitiveType: primitiveType, IsList: false}
}

// create a new DDPType from a element type
func NewListType(elementType TokenType) DDPType {
	return DDPType{PrimitiveType: elementType, IsList: true}
}

/* Helper functions for primitive constants */

// unexported ddp-types returned by the helper functions (go has no const struct-literals)
var (
	ddpVoidType   = DDPType{PrimitiveType: NICHTS, IsList: false}
	ddpIntType    = DDPType{PrimitiveType: ZAHL, IsList: false}
	ddpFloatType  = DDPType{PrimitiveType: KOMMAZAHL, IsList: false}
	ddpBoolType   = DDPType{PrimitiveType: BOOLEAN, IsList: false}
	ddpCharType   = DDPType{PrimitiveType: BUCHSTABE, IsList: false}
	ddpStringType = DDPType{PrimitiveType: TEXT, IsList: false}
)

// this function serves as constant variable for a void type
func DDPVoidType() DDPType {
	return ddpVoidType
}

// this function serves as constant variable for a int type
func DDPIntType() DDPType {
	return ddpIntType
}

// this function serves as constant variable for a float type
func DDPFloatType() DDPType {
	return ddpFloatType
}

// this function serves as constant variable for a bool type
func DDPBoolType() DDPType {
	return ddpBoolType
}

// this function serves as constant variable for a char type
func DDPCharType() DDPType {
	return ddpCharType
}

// this function serves as constant variable for a string type
func DDPStringType() DDPType {
	return ddpStringType
}

type ArgType struct {
	Type        DDPType // the type
	IsReference bool    // if it is a reference
}

func (argType ArgType) String() string {
	if !argType.IsReference {
		return argType.Type.String()
	}
	if argType.Type.IsList {
		return argType.Type.String() + "n Referenz"
	}
	switch argType.Type.PrimitiveType {
	case ZAHL, KOMMAZAHL:
		return argType.Type.String() + "en Referenz"
	case BUCHSTABE:
		return "Buchstaben Referenz"
	default:
		return argType.Type.String() + " Referenz"
	}
}
