package ddptypes

type ListType struct {
	Underlying Type
}

func (ListType) ddpType() {}

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
// if typ is not a list it is returned unchanged
func GetListUnderlying(typ Type) Type {
	if listType, isList := CastList(typ); isList {
		return listType.Underlying
	}
	return typ
}

// gets the underlying type for nested lists
// if typ is not a list type typ is returned
func GetNestedListUnderlying(typ Type) Type {
	for IsList(typ) {
		typ = GetNestedListUnderlying(GetUnderlying(typ).(ListType).Underlying)
	}
	return typ
}

func getTrueListUnderlying(typ Type) Type {
	typ = TrueUnderlying(typ)
	if IsList(typ) {
		typ = ListType{Underlying: getTrueListUnderlying(typ.(ListType).Underlying)}
	}
	return typ
}

func ListTrueUnderlying(typ Type) Type {
	typ = TrueUnderlying(typ)
	if IsList(typ) {
		typ = ListTrueUnderlying(typ.(ListType).Underlying)
	}
	return typ
}
