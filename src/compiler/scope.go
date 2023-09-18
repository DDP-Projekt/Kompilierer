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
	enclosing   *scope                // enclosing scope, nil if it is the global scope
	variables   map[string]varwrapper // variables in this scope
	temporaries []varwrapper          // intermediate values that need to be freed when the scope ends
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

func (scope *scope) addTemporary(val value.Value, typ ddpIrType) value.Value {
	scope.temporaries = append(scope.temporaries, varwrapper{val: val, typ: typ, isRef: false})
	return val
}

// removes the given value from scope.temporaries giving ownership to the caller
// who now has to make sure it is freed
// returns the value
func (scope *scope) claimTemporary(val value.Value) value.Value {
	// loop backwards, because we mostlikely claim new values more frequently
	for i := len(scope.temporaries) - 1; i >= 0; i-- {
		// remove the found value and return
		if scope.temporaries[i].val == val {
			scope.temporaries = append(scope.temporaries[0:i], scope.temporaries[i+1:]...)
			return val
		}
	}
	panic("attempted Value claim not found in scope.temporaries")
}
