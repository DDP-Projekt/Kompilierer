package ddperror

type Code uint // type of an ddperror Code

// some errors that don't fit into any category
const (
	MISC_INCLUDE_ERROR Code = iota
)

// syntax error codes
const (
	SYN_UNEXPECTED_TOKEN       Code = iota + 1000 // wrong tokenType found (used in parser.consume etc.)
	SYN_EXPECTED_LITERAL                          // a literal was expected but a whole expression or other was found
	SYN_EXPECTED_IDENTIFIER                       // expected an identifier like a parameter name
	SYN_EXPECTED_TYPENAME                         // a typename was expected
	SYN_EXPECTED_CAPITAL                          // a capitalized letter was expected
	SYN_MALFORMED_LITERAL                         // a literal was malformed
	SYN_MALFORMED_INCLUDE_PATH                    // a include path was malformed or not parsable in some way
	SYN_MALFORMED_ALIAS                           // an alias is malformed syntatically (invalid tokens etc.)
	SYN_INVALID_UTF8                              // text was not valid utf8
)

// semantic error codes
const (
	SEM_NAME_ALREADY_DEFINED           Code = iota + 2000 // multiple definition of Name (duplicate variable/function names)
	SEM_NAME_UNDEFINED                                    // a function or variable name is not yet defined
	SEM_PARAM_NAME_TYPE_COUNT_MISMATCH                    // number of names and types of parameters in function declaration don't match
	SEM_EXPECTED_LINKABLE_FILEPATH                        // expected a filepath to a .c, .lib, ... file
	SEM_NON_GLOBAL_FUNCTION                               // a function was declared in a non-global scope
	SEM_MISSING_RETURN                                    // return statement in function body is missing
	SEM_MALFORMED_ALIAS                                   // an alias is malformed (too few/many parameters, too short, whatever)
	SEM_ALIAS_ALREADY_TAKEN                               // the alias already stands for another function
	SEM_ALIAS_ALREADY_DEFINED                             // the alias already stands for a different function
	SEM_ALIAS_MUST_BE_GLOBAL                              // a non-global alias declaration was found
	SEM_GLOBAL_RETURN                                     // a return statement outside a function was found
)

// type error codes
const (
	TYP_TYPE_MISMATCH Code = iota + 3000
)

func (code Code) IsMiscError() bool {
	return code < 1000
}

func (code Code) IsSyntaxError() bool {
	return code >= 1000 && code < 2000
}

func (code Code) IsSemanticError() bool {
	return code >= 2000 && code < 3000
}

func (code Code) IsTypeError() bool {
	return code >= 3000 && code < 4000
}

// returns the Prefix before "Fehler" of the given code
// Prefixes are:
// Syntax, Semantischer, Typ or nothing for MISC
func (code Code) ErrorPrefix() string {
	if code.IsMiscError() {
		return ""
	} else if code.IsSyntaxError() {
		return "Syntax"
	} else if code.IsSemanticError() {
		return "Semantischer"
	} else if code.IsTypeError() {
		return "Typ"
	}
	return "Invalider"
}
