package ast

// stores symbols for one scope of an ast
type SymbolTable struct {
	Enclosing    *SymbolTable           // enclosing scope (nil in the global scope)
	declarations map[string]Declaration // map of all variables and functions
}

func NewSymbolTable(enclosing *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Enclosing:    enclosing,
		declarations: make(map[string]Declaration),
	}
}

// returns the declaration corresponding to name and wether it exists
// if the symbol existed, the second bool is true, if it is a variable and false if it is a funciton
// call like this: decl, exists, isVar := LookupDecl(name)
func (scope *SymbolTable) LookupDecl(name string) (Declaration, bool, bool) {
	if decl, ok := scope.declarations[name]; !ok {
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
	if _, ok := scope.declarations[name]; ok {
		return true
	}
	if _, ok := decl.(*BadDecl); ok {
		return false
	}
	scope.declarations[name] = decl
	return false
}
