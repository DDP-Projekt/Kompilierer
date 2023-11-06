package ddptypes

type VoidType struct{}

func (VoidType) Gender() GrammaticalGender {
	return INVALID
}

func (VoidType) String() string {
	return "nichts"
}

/*
func Void() VoidType {
	return VoidType{}
}
*/
