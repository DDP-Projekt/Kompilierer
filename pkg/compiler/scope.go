package compiler

import (
	"github.com/llir/llvm/ir/types"
	"github.com/llir/llvm/ir/value"
)

type varwrapper struct {
	v value.Value
	t types.Type
}

type scope struct {
	enclosing *scope                // enclosing scope, nil if it is the global scope
	variables map[string]varwrapper // variables in this scope
}

func newScope(enclosing *scope) *scope {
	return &scope{
		enclosing: enclosing,
		variables: make(map[string]varwrapper),
	}
}

// returns the named variable
func (s *scope) lookupVar(name string) varwrapper {
	if v, ok := s.variables[name]; !ok {
		if s.enclosing != nil {
			return s.enclosing.lookupVar(name)
		}
		return varwrapper{v: nil, t: void} // variable doesn't exist
	} else {
		return v
	}
}

// add a variable to the scope
func (s *scope) addVar(name string, val value.Value, ty types.Type) value.Value {
	s.variables[name] = varwrapper{v: val, t: ty}
	return val
}
