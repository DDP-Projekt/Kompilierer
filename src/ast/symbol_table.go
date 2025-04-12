package ast

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type SymbolTable interface {
	// returns the enclosing scope (nil in the global scope)
	Enclosing() SymbolTable
	// returns the declaration corresponding to name and wether it exists
	// if the symbol existed, the second bool is true, if it is a variable and false if it is a funciton
	// call like this: decl, exists, isVar := LookupDecl(name)
	LookupDecl(name string) (Declaration, bool, bool)
	// inserts a declaration into the scope if it didn't exist yet
	// and returns wether it already existed
	// BadDecls are ignored
	InsertDecl(name string, decl Declaration) bool
	// returns the given type if present, else nil
	//
	//	Type, ok := table.LookupType(name)
	LookupType(name string) (ddptypes.Type, bool)
}

// stores symbols for one scope of an ast
type BasicSymbolTable struct {
	EnclosingTable SymbolTable            // enclosing scope (nil in the global scope)
	Declarations   map[string]Declaration // map of all variables, functions and structs
}

func NewSymbolTable(enclosing SymbolTable) SymbolTable {
	return &BasicSymbolTable{
		EnclosingTable: enclosing,
		Declarations:   make(map[string]Declaration, 8),
	}
}

func (scope *BasicSymbolTable) Enclosing() SymbolTable {
	return scope.EnclosingTable
}

// returns the declaration corresponding to name and wether it exists
// if the symbol existed, the second bool is true, if it is a variable and false if it is a funciton
// call like this: decl, exists, isVar := LookupDecl(name)
func (scope *BasicSymbolTable) LookupDecl(name string) (Declaration, bool, bool) {
	if decl, ok := scope.Declarations[name]; !ok {
		if scope.EnclosingTable != nil {
			return scope.EnclosingTable.LookupDecl(name)
		}
		return nil, false, false
	} else {
		return decl, true, IsVarConstDecl(decl)
	}
}

// inserts a declaration into the scope if it didn't exist yet
// and returns wether it already existed
// BadDecls are ignored
func (scope *BasicSymbolTable) InsertDecl(name string, decl Declaration) bool {
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
func (scope *BasicSymbolTable) LookupType(name string) (ddptypes.Type, bool) {
	if decl, ok := scope.Declarations[name]; !ok {
		if scope.EnclosingTable != nil {
			return scope.EnclosingTable.LookupType(name)
		}
		return nil, false
	} else {
		return IsTypeDecl(decl)
	}
}
