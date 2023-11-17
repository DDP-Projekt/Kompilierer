package ddptypes

type ListType struct {
	Underlying Type
}

func (ListType) Gender() GrammaticalGender {
	return FEMININ
}

func (listType ListType) String() string {
	if IsPrimitive(listType.Underlying) {
		switch listType.Underlying.(PrimitiveType) {
		case ZAHL:
			return "Zahlen Liste"
		case KOMMAZAHL:
			return "Kommazahlen Liste"
		case WAHRHEITSWERT:
			return "Wahrheitswert Liste"
		case BUCHSTABE:
			return "Buchstaben Liste"
		case TEXT:
			return "Text Liste"
		default:
			panic("invaid primitive type")
		}
	} else if _, isVoid := listType.Underlying.(VoidType); isVoid {
		panic("void list type")
	}
	return listType.Underlying.String() + " Liste" // no 100% correct yet
}

// gets the underlying type for nested lists
// if typ is not a list type typ is returned
func GetNestedUnderlying(typ Type) Type {
	if listTyp, isList := typ.(ListType); isList {
		typ = listTyp.Underlying
	} else {
		return typ
	}

	for {
		underlying, isList := typ.(ListType)
		if !isList {
			return typ
		}
		typ = underlying
	}
}
