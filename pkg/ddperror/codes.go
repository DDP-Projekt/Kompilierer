package ddperror

type Code int // type of an ddperror Code

// syntax error codes
const (
	SYN_EXPECTED_IDENT Code = iota
)

// semantic error codes
const (
	SEM_UNDEFINED_IDENT Code = iota + 1000
)

// type error codes
const (
	TYP_TYPE_MISMATCH Code = iota + 2000
)

func (code Code) IsSyntaxError() bool {
	return code >= 0 && code < 1000
}

func (code Code) IsSemanticError() bool {
	return code >= 1000 && code < 2000
}

func (code Code) IsTypeError() bool {
	return code >= 2000 && code < 3000
}

// returns the Prefix before "Fehler" of the given code
// Prefixes are:
// Syntax, Semantischer, Typ
func (code Code) ErrorPrefix() string {
	if code.IsSyntaxError() {
		return "Syntax"
	} else if code.IsSemanticError() {
		return "Semantischer"
	} else if code.IsTypeError() {
		return "Typ"
	}
	return "Invalider"
}
