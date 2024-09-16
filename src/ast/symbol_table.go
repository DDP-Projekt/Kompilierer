package ast

import (
	"slices"

	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

// stores symbols for one scope of an ast
type SymbolTable struct {
	Enclosing    *SymbolTable             // enclosing scope (nil in the global scope)
	Declarations map[string]Declaration   // map of all variables, functions and structs
	Operators    map[Operator][]*FuncDecl // map of all overloads for all operators
}

func NewSymbolTable(enclosing *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Enclosing:    enclosing,
		Declarations: make(map[string]Declaration, 8),
		Operators:    make(map[Operator][]*FuncDecl),
	}
}

// returns the declaration corresponding to name and wether it exists
// if the symbol existed, the second bool is true, if it is a variable and false if it is a funciton
// call like this: decl, exists, isVar := LookupDecl(name)
func (scope *SymbolTable) LookupDecl(name string) (Declaration, bool, bool) {
	if decl, ok := scope.Declarations[name]; !ok {
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupDecl(name)
		}
		return nil, false, false
	} else {
		_, isVar := decl.(*VarDecl)
		return decl, true, isVar
	}
}

// inserts a declaration into the scope if it didn't exist yet
// and returns wether it already existed
// BadDecls are ignored
func (scope *SymbolTable) InsertDecl(name string, decl Declaration) bool {
	if _, ok := scope.Declarations[name]; ok {
		return true
	}
	if _, ok := decl.(*BadDecl); ok {
		return false
	}
	scope.Declarations[name] = decl
	return false
}

// returns the given type if present, else nil
//
//	Type, ok := table.LookupType(name)
func (scope *SymbolTable) LookupType(name string) (ddptypes.Type, bool) {
	if decl, ok := scope.Declarations[name]; !ok {
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupType(name)
		}
		return nil, false
	} else {
		switch decl := decl.(type) {
		case *StructDecl:
			return decl.Type, true
		case *TypeAliasDecl:
			return decl.Type, true
		case *TypeDefDecl:
			return decl.Type, true
		default:
			return nil, false
		}
	}
}

// returns all overloads for the given operator
func (scope *SymbolTable) LookupOperator(op Operator) []*FuncDecl {
	if overloads, ok := scope.Operators[op]; !ok {
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupOperator(op)
		}
		return nil
	} else {
		return overloads
	}
}

// adds the overload to the given operator
// sorting references to the top
func (scope *SymbolTable) AddOverload(op Operator, overload *FuncDecl) {
	if scope.Enclosing != nil {
		panic("Attempted to add overload in non-global scope")
	}

	overloads := scope.LookupOperator(op)

	// keep the slice sorted in descending order, so that references are prioritized
	i, _ := slices.BinarySearchFunc(overloads, overload, func(a, t *FuncDecl) int {
		countRefArgs := func(params []ParameterInfo) int {
			result := 0
			for i := range params {
				if params[i].Type.IsReference {
					result++
				}
			}
			return result
		}

		return countRefArgs(t.Parameters) - countRefArgs(a.Parameters)
	})

	overloads = slices.Insert(overloads, i, overload)
	scope.Operators[op] = overloads
}
