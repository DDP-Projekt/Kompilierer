package ddptypes

import (
	"slices"
)

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

func CastDeeplyNestedGenerics(t Type) ([]GenericType, bool) {
	t = GetNestedListUnderlying(t)
	generic, ok := t.(GenericType)
	if ok {
		return []GenericType{generic}, ok
	}

	var result []GenericType
	if structType, isStruct := CastStruct(t); isStruct {
		result = make([]GenericType, 0, len(structType.Fields))
		for _, field := range structType.Fields {
			genericFields, _ := CastDeeplyNestedGenerics(field.Type)
			genericFields = slices.DeleteFunc(genericFields, func(t GenericType) bool { return slices.Contains(result, t) })
			result = append(result, genericFields...)
		}
	}

	return result, len(result) > 0
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

	// TODO
	if structType, isStruct := CastStruct(instantiatedType); isStruct && structType.genericType != nil {
		instantiationTypes := make([]Type, len(structType.instantiatedWith))
		for i, t := range structType.instantiatedWith {
			if generic, isGeneric := CastGeneric(t); isGeneric {
				instantiationTypes[i] = genericTypes[generic.String()]
			} else {
				instantiationTypes[i] = t
			}
		}

		instantiatedType = GetInstantiatedStructType(structType.genericType, instantiationTypes)
	}

	for range listDepth {
		instantiatedType = ListType{Underlying: instantiatedType}
	}

	return instantiatedType
}

type GenericStructInstantiation struct {
	Type         *StructType
	GenericTypes []Type
}

type GenericStructType struct {
	StructType     StructType
	GenericTypes   []GenericType
	Instantiations []GenericStructInstantiation
}

func (*GenericStructType) ddpType() {}

func (t *GenericStructType) Gender() GrammaticalGender {
	return t.StructType.Gender()
}

func (t *GenericStructType) String() string {
	return t.StructType.String()
}

// returns a new struct type with the fields filled in correctly
func GetInstantiatedStructType(s *GenericStructType, genericTypes []Type) *StructType {
	for _, instantiation := range s.Instantiations {
		if slices.EqualFunc(instantiation.GenericTypes, genericTypes, Equal) {
			return instantiation.Type
		}
	}

	result := StructType{
		Name:             s.StructType.Name,
		GramGender:       s.Gender(),
		Fields:           make([]StructField, len(s.StructType.Fields)),
		genericType:      s,
		instantiatedWith: genericTypes,
	}

	genericTypesMap := make(map[string]Type, len(s.GenericTypes))
	for i, generic := range s.GenericTypes {
		genericTypesMap[generic.Name] = genericTypes[i]
	}

	for i, field := range s.StructType.Fields {
		result.Fields[i].Name = field.Name
		result.Fields[i].Type = GetInstantiatedType(s.StructType.Fields[i].Type, genericTypesMap)
	}

	s.Instantiations = append(s.Instantiations, GenericStructInstantiation{
		Type:         &result,
		GenericTypes: genericTypes,
	})

	return &result
}
