package ddptypes

type Variable struct{}

func (Variable) ddpType() {}

func (Variable) Gender() GrammaticalGender {
	return FEMININ
}

func (Variable) String() string {
	return "Variable"
}

var VARIABLE = Variable{}
