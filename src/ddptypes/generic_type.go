package ddptypes

// represents a generic type in a declaration (not yet resolved)
type GenericType struct {
	Name string
}

func (GenericType) ddpType() {}

func (GenericType) Gender() GrammaticalGender {
	// TODO: generisches Maskulinum or INVALID_GENDER or extra gender?
	return MASKULIN
}

func (t GenericType) String() string {
	return t.Name
}

func CastDeeplyNestedGeneric(t Type) (GenericType, bool) {
	t = GetNestedListUnderlying(t)
	generic, ok := t.(GenericType)
	return generic, ok
}

// helper to unify generic types in a loop
func UnifyGenericType(argType Type, paramType ParameterType, genericTypes map[string]Type) Type {
	instantiatedType, genericType := argType, paramType.Type

	argListType, isArgList := CastList(instantiatedType)
	paramListType, isParamList := CastList(genericType)

	listDepth := 0
	for isArgList && isParamList {
		listDepth++
		instantiatedType, genericType = argListType.Underlying, paramListType.Underlying

		if IsGeneric(genericType) {
			break
		}

		argListType, isArgList = CastList(instantiatedType)
		paramListType, isParamList = CastList(genericType)
	}

	if isParamList && !isArgList {
		return nil
	}

	if generic, ok := CastGeneric(genericType); ok {
		unified := false
		genericType, unified = genericTypes[generic.Name]

		if !unified {
			genericTypes[generic.Name] = instantiatedType
			genericType = instantiatedType
		}
	}

	for range listDepth {
		genericType = ListType{Underlying: genericType}
	}
	return genericType
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
