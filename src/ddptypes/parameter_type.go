package ddptypes

// represents the type of a function parameter
// which may be a reference
type ParameterType struct {
	Type        Type
	IsReference bool
}

func ParamTypesEqual(p1, p2 ParameterType) bool {
	return p1.IsReference == p2.IsReference && Equal(p1.Type, p2.Type)
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
		case WAHRHEITSWERT, TEXT:
			return paramType.Type.String() + " Referenz"
		}
	}

	if IsVoid(paramType.Type) {
		panic("void type Reference")
	}

	if IsStruct(paramType.Type) {
		return paramType.Type.String() + " Referenz"
	}

	if IsAny(paramType.Type) {
		return paramType.Type.String() + "n Referenz"
	}

	return paramType.Type.String() + " Referenz"
}
