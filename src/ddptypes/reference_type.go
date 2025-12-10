package ddptypes

type ReferenceType struct {
	Type Type
}

func (ReferenceType) ddpType() {}

func (ReferenceType) Gender() GrammaticalGender {
	return FEMININ
}

func (t ReferenceType) String() string {
	if IsList(t.Type) {
		return t.Type.String() + "n Referenz"
	}

	if IsPrimitive(t.Type) {
		switch GetUnderlying(t.Type).(PrimitiveType) {
		case ZAHL, KOMMAZAHL:
			return t.Type.String() + "en Referenz"
		case BUCHSTABE:
			return "Buchstaben Referenz"
		case BYTE, WAHRHEITSWERT, TEXT:
			return t.Type.String() + " Referenz"
		}
	}

	if IsVoid(t.Type) {
		panic("void type Reference")
	}

	if IsStruct(t.Type) {
		return t.Type.String() + " Referenz"
	}

	if IsAny(t.Type) {
		return t.Type.String() + "n Referenz"
	}

	return t.Type.String() + " Referenz"
}
