package ddptypes

// represents the type of a function parameter
// which may be a reference
type ParameterType struct {
	Type        Type
	IsReference bool
}

func (paramType ParameterType) String() string {
	if !paramType.IsReference {
		return paramType.Type.String()
	}

	if IsList(paramType.Type) {
		return paramType.Type.String() + "n Referenz"
	}

	if IsPrimitive(paramType.Type) {
		switch paramType.Type.(PrimitiveType) {
		case ZAHL, KOMMAZAHL:
			return paramType.Type.String() + "en Referenz"
		case BUCHSTABE:
			return "Buchstaben Referenz"
		case BOOLEAN, TEXT:
			return paramType.Type.String() + " Referenz"
		}
	}

	if IsVoid(paramType.Type) {
		panic("void type Reference")
	}

	panic("unknown type")
}
