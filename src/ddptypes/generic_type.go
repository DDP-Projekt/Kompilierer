package ddptypes

// represents a generic type in a declaration (not yet resolved)
type GenericType struct {
	Name string
}

func (*GenericType) ddpType() {}

func (*GenericType) Gender() GrammaticalGender {
	// TODO: generisches Maskulinum or INVALID_GENDER or extra gender?
	return MASKULIN
}

func (t *GenericType) String() string {
	return t.Name
}

func CastDeeplyNestedGeneric(t Type) (*GenericType, bool) {
	t = GetNestedListUnderlying(t)
	generic, ok := t.(*GenericType)
	return generic, ok
}

// represents a resolved generic type that is used during parsing of the generic function
type InstantiatedGenericType struct {
	Actual Type
}

func (*InstantiatedGenericType) ddpType() {}

func (t *InstantiatedGenericType) Gender() GrammaticalGender {
	return t.Actual.Gender()
}

func (t *InstantiatedGenericType) String() string {
	return t.Actual.String()
}

// instantiates a type by replacing nested occurences of generic types with the actual types
func GetInstantiatedType(t Type, genericTypes map[string]Type) Type {
	instantiatedType := t
	listType, isList := CastList(instantiatedType)

	listDepth := 0
	for isList {
		listDepth++
		instantiatedType = listType.Underlying

		if IsGeneric(instantiatedType) {
			break
		}

		listType, isList = CastList(instantiatedType)
	}

	if generic, ok := CastGeneric(instantiatedType); ok {
		instantiatedType = genericTypes[generic.Name]
	}

	for range listDepth {
		instantiatedType = ListType{Underlying: instantiatedType}
	}

	return instantiatedType
}
