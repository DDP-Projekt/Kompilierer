package ddptypes

import (
	"fmt"
)

// represents a single field of a struct
type StructField struct {
	// name of the field
	Name string
	// type of the field
	Type Type
}

// represents the type of a ddp struct
type StructType struct {
	// name of the struct
	Name       string
	cachedName *string
	// grammatical gender of the struct name
	GramGender GrammaticalGender
	// fields of the struct
	// in order of declaration
	Fields []StructField
	// the genericStructType that instantiated this type
	// nil if this is not an instantiated generic
	genericType      *GenericStructType
	instantiatedWith []Type // the GenericTypes of the parent that this struct was instantiated with
}

func (*StructType) ddpType() {}

func (t *StructType) Gender() GrammaticalGender {
	return t.GramGender
}

func (t *StructType) String() string {
	if t.cachedName != nil {
		return *t.cachedName
	}

	name := ""
	if t.genericType != nil {
		for _, typParam := range t.instantiatedWith {
			typParamString := typParam.String()
			if _, ok := typParam.(*StructType); ok {
				name += fmt.Sprintf("(%s)-", typParamString)
			} else {
				name += fmt.Sprintf("%s-", typParamString)
			}
		}
	}
	name += t.Name

	t.cachedName = &name

	return *t.cachedName
}

// checks wether two structs are structurally equal, that is
// their fields have the same type and are in the same order
// even ignoring TypeDefs
func StructurallyEqual(t1, t2 *StructType) bool {
	if len(t1.Fields) != len(t2.Fields) {
		return false
	}

	for i := range t1.Fields {
		trueType1 := TrueUnderlying(t1.Fields[i].Type)
		trueType2 := TrueUnderlying(t2.Fields[i].Type)

		structType1, isStruct1 := CastStruct(trueType1)
		structType2, isStruct2 := CastStruct(trueType2)

		if isStruct1 && isStruct2 {
			if !StructurallyEqual(structType1, structType2) {
				return false
			}
		} else if !Equal(trueType1, trueType2) {
			return false
		}
	}

	return true
}

// returns the Generic Type this struct was instantiated from or nil if it is not generic
func InstantiatedFrom(s *StructType) (*GenericStructType, []Type) {
	return s.genericType, s.instantiatedWith
}
