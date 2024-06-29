package ddptypes

type VoidType struct{}

func (VoidType) ddpType() {}

func (VoidType) Gender() GrammaticalGender {
	return INVALID_GENDER
}

func (VoidType) String() string {
	return "nichts"
}

/*
func Void() VoidType {
	return VoidType{}
}
*/
