package ddptypes

// a type alias only refers to another type with a new name
type TypeAlias struct {
	Name       string
	Underlying Type
	GramGender GrammaticalGender
}

func (a *TypeAlias) String() string {
	return a.Name
}

func (a *TypeAlias) Gender() GrammaticalGender {
	return a.GramGender
}

func (*TypeAlias) ddpType() {}

// a typedef defines a new type that is equal to another
type TypeDef struct {
	Name       string
	Underlying Type
	GramGender GrammaticalGender
}

func (*TypeDef) ddpType() {}

func (d *TypeDef) String() string {
	return d.Name
}

func (d *TypeDef) Gender() GrammaticalGender {
	return d.GramGender
}

// returns the underlying type for nested TypeAliases and TypeDefs
func (d *TypeDef) TrueUnderlying() Type {
	t := GetUnderlying(d.Underlying)
	if typedef, ok := t.(*TypeDef); ok {
		return typedef.TrueUnderlying()
	}
	return t
}
