package interpreter

import "DDP/pkg/ast"

// saves variable states
type environment struct {
	enclosing *environment
	variables map[string]value
	functions map[string]*ast.FuncDecl
}

func newEnvironment(enclosing *environment) *environment {
	return &environment{
		enclosing: enclosing,
		variables: make(map[string]value),
		functions: make(map[string]*ast.FuncDecl),
	}
}

// receives the value of the var name and checks if it exists or not
func (e *environment) lookupVar(name string) (value, bool) {
	if v, ok := e.variables[name]; !ok {
		if e.enclosing != nil {
			return e.enclosing.lookupVar(name)
		}
		return nil, false // variable doesn't exist
	} else {
		return v, true
	}
}

// receives the function of name and checks if it exists or not
func (e *environment) lookupFunc(name string) (*ast.FuncDecl, bool) {
	if v, ok := e.functions[name]; !ok {
		if e.enclosing != nil {
			return e.enclosing.lookupFunc(name)
		}
		return nil, false
	} else {
		return v, true
	}
}

// updates a variable in the environment or adds it if it was missing
func (s *environment) updateVar(name string, val value) {
	if _, exists := s.lookupVar(name); !exists {
		s.addVar(name, val)
	} else {
		for env := s; ; env = env.enclosing {
			if _, ok := env.variables[name]; ok {
				env.variables[name] = val
				break
			}
		}
	}
}

func (s *environment) addVar(name string, val value) {
	s.variables[name] = val
}

// updates a function in the environment or adds it if it was missing
func (s *environment) addFunc(name string, fun *ast.FuncDecl) {
	s.functions[name] = fun
}
