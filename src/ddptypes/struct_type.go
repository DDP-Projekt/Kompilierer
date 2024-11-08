package ddptypes

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
	Name string
	// grammatical gender of the struct name
	GramGender GrammaticalGender
	// fields of the struct
	// in order of declaration
	Fields []StructField
}

func (*StructType) ddpType() {}

func (t *StructType) Gender() GrammaticalGender {
	return t.GramGender
}

func (t *StructType) String() string {
	return t.Name
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
