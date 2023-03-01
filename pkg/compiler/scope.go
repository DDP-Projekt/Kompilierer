package compiler

import (
	"github.com/llir/llvm/ir/value"
)

// wraps a ir ir alloca + ir type for a variable
type varwrapper struct {
	val   value.Value // alloca or global-Def in the ir
	typ   ddpIrType   // ir type of the variable
	isRef bool
}

// wraps local variables of a scope + the enclosing scope
type scope struct {
	enclosing      *scope                // enclosing scope, nil if it is the global scope
	variables      map[string]varwrapper // variables in this scope
	non_primitives []varwrapper          // in case of scope-nested returns, these have to be freed
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
		return varwrapper{val: nil, typ: nil} // variable doesn't exist (should not happen, resolver should take care of that)
	} else {
		return v
	}
}

// add a variable to the scope
func (scope *scope) addVar(name string, val value.Value, ty ddpIrType, isRef bool) value.Value {
	scope.variables[name] = varwrapper{val: val, typ: ty, isRef: isRef}
	return val
}

func (scope *scope) addNonPrimitive(val value.Value, typ ddpIrType) value.Value {
	scope.non_primitives = append(scope.non_primitives, varwrapper{val: val, typ: typ, isRef: false})
	return val
}
