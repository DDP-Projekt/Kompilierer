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

// this function serves as constant variable for a void type
func DDPVoidType() DDPType {
	return DDPType{PrimitiveType: NICHTS, IsList: false}
}

// this function serves as constant variable for a int type
func DDPIntType() DDPType {
	return DDPType{PrimitiveType: ZAHL, IsList: false}
}

// this function serves as constant variable for a float type
func DDPFloatType() DDPType {
	return DDPType{PrimitiveType: KOMMAZAHL, IsList: false}
}

// this function serves as constant variable for a bool type
func DDPBoolType() DDPType {
	return DDPType{PrimitiveType: BOOLEAN, IsList: false}
}

// this function serves as constant variable for a char type
func DDPCharType() DDPType {
	return DDPType{PrimitiveType: BUCHSTABE, IsList: false}
}

// this function serves as constant variable for a string type
func DDPStringType() DDPType {
	return DDPType{PrimitiveType: TEXT, IsList: false}
}
