package ddptypes

type ListType struct {
	ElementType Type
}

func (ListType) ddpType() {}

func (ListType) Gender() GrammaticalGender {
	return FEMININ
}

func (listType ListType) String() string {
	if _, ok := listType.ElementType.(PrimitiveType); ok {
		switch listType.ElementType.(PrimitiveType) {
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
	} else if _, isVoid := listType.ElementType.(VoidType); isVoid {
		panic("void list type")
	}
	return listType.ElementType.String() + " Liste" // no 100% correct yet
}

// gets the underlying type of a List
// if typ is not a list it is returned unchanged
func GetListElementType(typ Type) Type {
	if listType, isList := CastList(typ); isList {
		return listType.ElementType
	}
	return typ
}

// gets the underlying type for nested lists
// if typ is not a list type typ is returned
func GetNestedListElementType(typ Type) Type {
	for IsList(typ) {
		typ = GetNestedListElementType(GetUnderlying(typ).(ListType).ElementType)
	}
	return typ
}

// helper that flattens list types to not contain typedefs/aliases
func getTrueListUnderlying(typ Type) Type {
	typ = TrueUnderlying(typ)
	if IsList(typ) {
		typ = ListType{ElementType: getTrueListUnderlying(typ.(ListType).ElementType)}
	}
	return typ
}

// Returns the True Underlying type for a list
// that is the element type of a list all the way down
// example:
// - int 				  	  -> int
// - list(int) 			 	  -> int
// - list(typedef(int)) 	  -> int
// - list(list(int))     	  -> int
// - list(list(typedef(int))) -> int
func ListTrueUnderlying(typ Type) Type {
	typ = TrueUnderlying(typ)
	if IsList(typ) {
		typ = ListTrueUnderlying(typ.(ListType).ElementType)
	}
	return typ
}
