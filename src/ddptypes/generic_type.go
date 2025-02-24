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
