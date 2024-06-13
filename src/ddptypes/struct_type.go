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
	// nasty hack to get name mangling for structs working
	// TODO: refactor
	ModName string
}

func (t *StructType) Gender() GrammaticalGender {
	return t.GramGender
}

func (t *StructType) String() string {
	return t.Name
}
