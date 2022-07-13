package compiler

import (
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

// wraps a ir ir alloca + ir type for a variable
type varwrapper struct {
	val value.Value // alloca in the ir
	typ types.Type  // ir type of the variable
}

// wraps local variables of a scope + the enclosing scope
type scope struct {
	enclosing *scope                // enclosing scope, nil if it is the global scope
	variables map[string]varwrapper // variables in this scope
}

// create a new scope in the enclosing scope
func newScope(enclosing *scope) *scope {
	return &scope{
		enclosing: enclosing,
		variables: make(map[string]varwrapper),
	}
}

// returns the named variable
// if not present the enclosing scopes are checked
// until the global scope
func (s *scope) lookupVar(name string) varwrapper {
	if v, ok := s.variables[name]; !ok {
		if s.enclosing != nil {
			return s.enclosing.lookupVar(name)
		}
		return varwrapper{val: nil, typ: void} // variable doesn't exist (should not happen, resolver should take care of that)
	} else {
		return v
	}
}

// add a variable to the scope
func (scope *scope) addVar(name string, val value.Value, ty types.Type) value.Value {
	scope.variables[name] = varwrapper{val: val, typ: ty}
	return val
}
