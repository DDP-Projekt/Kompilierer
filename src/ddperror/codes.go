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
	SYN_GENDER_MISMATCH                           // the expected and actual grammatical gender used mismatched
	SYN_INVALID_OPERATOR                          // the given string is not a valid operator
)

// semantic error codes
const (
	SEM_NAME_ALREADY_DEFINED                 Code = iota + 2000 // multiple definition of Name (duplicate variable/function names)
	SEM_NAME_UNDEFINED                                          // a function or variable name is not yet defined
	SEM_PARAM_NAME_TYPE_COUNT_MISMATCH                          // number of names and types of parameters in function declaration don't match
	SEM_EXPECTED_LINKABLE_FILEPATH                              // expected a filepath to a .c, .lib, ... file
	SEM_NON_GLOBAL_FUNCTION                                     // a function was declared in a non-global scope
	SEM_MISSING_RETURN                                          // return statement in function body is missing
	SEM_MALFORMED_ALIAS                                         // an alias is malformed (too few/many parameters, too short, whatever)
	SEM_ALIAS_ALREADY_TAKEN                                     // the alias already stands for another function
	SEM_ALIAS_ALREADY_DEFINED                                   // the alias already stands for a different function
	SEM_ALIAS_MUST_BE_GLOBAL                                    // a non-global alias declaration was found
	SEM_ALIAS_BAD_ARGS                                          // the number of arguments is wrong or then names don't match etc.
	SEM_GLOBAL_RETURN                                           // a return statement outside a function was found
	SEM_BAD_NAME_CONTEXT                                        // a function name was used in place of a variable name or vice versa
	SEM_NON_GLOBAL_PUBLIC_DECL                                  // a non-global variable was declared public
	SEM_NON_GLOBAL_TYPE_DECL                                    // a non-global type was declared
	SEM_BAD_FIELD_ACCESS                                        // a non-existend field was accessed or similar
	SEM_BAD_PUBLIC_MODIFIER                                     // a public modifier was missing or similar, for example in a struct decl
	SEM_BREAK_CONTINUE_NOT_IN_LOOP                              // a break or continue statement was found outside a loop
	SEM_UNKNOWN_TYPE                                            // a type was used that was not imported yet
	SEM_UNNECESSARY_EXTERN_VISIBLE                              // a function was specified as both extern visible and extern defined
	SEM_BAD_OPERATOR_PARAMS                                     // the function parameters do not fit the overloaded operator (e.g. they are too few/too many)
	SEM_OVERLOAD_ALREADY_DEFINED                                // an overload for the given types is already present
	SEM_TODO_STMT_FOUND                                         // ... wurde gefunden
	SEM_BAD_TYPEDEF                                             // any was typedefed
	SEM_FORWARD_DECL_WITHOUT_DEF                                // a function was declared as forward decl but never defined
	SEM_WRONG_DECL_MODULE                                       // a definition was provided for a function from a different module
	SEM_DEFINITION_ALREADY_DEFINED                              // a forward decl was already defined
	SEM_GENERIC_FUNCTION_WITHOUT_TYPEPARAM                      // a generic function without any generic parameters
	SEM_GENERIC_STRUCT_WITHOUT_TYPEPARAM                        // a generic struct without any generic parameters
	SEM_GENERIC_FUNCTION_EXTERN_VISIBLE                         // a generic function was marked as extern visible
	SEM_GENERIC_FUNCTION_BODY_UNDEFINED                         // a generic function was declared without a definition
	SEM_ERROR_INSTANTIATING_GENERIC_FUNCTION                    // an error occured while instantiating a generic function
	SEM_CANNOT_INSTANTIATE_NON_GENERIC_TYPE                     // a non-generic type was given with type parameters
	SEM_UNABLE_TO_UNIFY_FIELD_TYPES                             // for a generic struct alias not all typeparamters could be unified
	SEM_CONSTANT_IS_NOT_ASSIGNABLE                              // for a generic struct alias not all typeparamters could be unified
)

// type error codes
const (
	TYP_TYPE_MISMATCH                               Code = iota + 3000 // simple type mismatch in an operator
	TYP_BAD_ASSIGNEMENT                                                // invalid variable assignement
	TYP_BAD_INDEXING                                                   // type error in index expression
	TYP_BAD_LIST_LITERAL                                               // wrong type in list literal
	TYP_BAD_CAST                                                       // invalid type conversion
	TYP_EXPECTED_REFERENCE                                             // a variable (reference parameter) was expected
	TYP_INVALID_REFERENCE                                              // a char in a string was tried to be passed as refernce
	TYP_BAD_CONDITION                                                  // condition value was not of type boolean
	TYP_BAD_FOR                                                        // one of the expressions in a for loop was not of type int
	TYP_WRONG_RETURN_TYPE                                              // the return type did not match the function signature
	TYP_BAD_FIELD_ACCESS                                               // a non-struct type was accessed or similar
	TYP_PRIVATE_FIELD_ACCESS                                           // a non-public field was accessed from another module
	TYP_BAD_OPERATOR_RETURN_TYPE                                       // the return type of a operator overload is void
	TYP_REFERENCE_TYPE_PARAM                                           // a generic type param was a reference
	TYP_GENERIC_TYPE_NOT_UNIFIED                                       // a generic field type could not be unified
	TYP_COULD_NOT_INSTANTIATE_GENERIC                                  // a generic type could not be instantiated with the given parameters
	TYP_GENERIC_EXTERN_FUNCTION_BAD_PARAM_OR_RETURN                    // a generic extern functions parameter was not a list or reference
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
		return "Sonstiger"
	} else if code.IsSyntaxError() {
		return "Syntax"
	} else if code.IsSemanticError() {
		return "Semantischer"
	} else if code.IsTypeError() {
		return "Typ"
	}
	return "Invalider"
}

// returns the Prefix before "Warnung" of the given code
// Prefixes are:
// Syntax, Semantischer, Typ or nothing for MISC
func (code Code) WarningPrefix() string {
	if code.IsMiscError() {
		return "Sonstige"
	} else if code.IsSyntaxError() {
		return "Syntax"
	} else if code.IsSemanticError() {
		return "Semantik"
	} else if code.IsTypeError() {
		return "Typ"
	}
	return "Invalide"
}
