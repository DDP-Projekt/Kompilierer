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

// gets the underlying type of a List
func GetListUnderlying(typ Type) Type {
	if listTyp, isList := GetUnderlying(typ).(ListType); isList {
		return listTyp.Underlying
	}
	return typ
}

// gets the underlying type for nested lists
// if typ is not a list type typ is returned
func GetNestedListUnderlying(typ Type) Type {
	for IsList(typ) {
		typ = GetListUnderlying(typ)
	}
	return typ
}
