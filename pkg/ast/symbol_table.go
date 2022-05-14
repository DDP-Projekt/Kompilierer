package ast

import (
	"DDP/pkg/token"
)

// stores symbols for one scope of an ast
type SymbolTable struct {
	Enclosing *SymbolTable               // enclosing scope (nil in the global scope)
	variables map[string]token.TokenType // tokenType is used as type identifier (e.g. token.ZAHL -> int)
	functions map[string]*FuncDecl       // same here, but token.NICHTS stands for void (also, only the global-scope can have function declarations)
}

func NewSymbolTable(enclosing *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Enclosing: enclosing,
		variables: make(map[string]token.TokenType),
		functions: make(map[string]*FuncDecl),
	}
}

// returns the type of the variable name and if it exists in the table or it's enclosing scopes
func (s *SymbolTable) LookupVar(name string) (token.TokenType, bool) {
	if v, ok := s.variables[name]; !ok {
		if s.Enclosing != nil {
			return s.Enclosing.LookupVar(name)
		}
		return token.ILLEGAL, false // variable doesn't exist
	} else {
		return v, true
	}
}

// returns the type of the variable name and if it exists in the table or it's enclosing scopes
func (s *SymbolTable) LookupFunc(name string) (*FuncDecl, bool) {
	if v, ok := s.functions[name]; !ok {
		if s.Enclosing != nil {
			return s.Enclosing.LookupFunc(name)
		}
		return nil, false // function doesn't exist
	} else {
		return v, true
	}
}

// inserts a variable into the scope if it didn't exist yet
// and returns wether it already existed
func (s *SymbolTable) InsertVar(name string, typ token.TokenType) bool {
	switch typ {
	case token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE, token.TEXT:
	default:
		panic("invalid type argument")
	}
	if _, ok := s.variables[name]; ok {
		return true
	}
	s.variables[name] = typ
	return false
}

// inserts a function into the scope if it didn't exist yet
// and returns wether it already existed
func (s *SymbolTable) InsertFunc(name string, fun *FuncDecl) bool {
	switch fun.Type.Type {
	case token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE, token.TEXT, token.NICHTS:
	default:
		panic("invalid type argument")
	}
	if _, ok := s.functions[name]; ok {
		return true
	}
	s.functions[name] = fun
	return false
}

func (s *SymbolTable) Merge(other *SymbolTable) {
	for k, v := range other.functions {
		s.functions[k] = v
	}
	for k, v := range other.variables {
		s.variables[k] = v
	}
}

func (s *SymbolTable) Copy() *SymbolTable {
	table := &SymbolTable{
		Enclosing: s.Enclosing,
		variables: make(map[string]token.TokenType),
		functions: make(map[string]*FuncDecl),
	}
	for k, v := range s.variables {
		table.variables[k] = v
	}
	for k, v := range s.functions {
		table.functions[k] = v
	}
	return table
}
