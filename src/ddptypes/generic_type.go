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
