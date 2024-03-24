package ast

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// stores symbols for one scope of an ast
type SymbolTable struct {
	Enclosing    *SymbolTable           // enclosing scope (nil in the global scope)
	Declarations map[string]Declaration // map of all variables, functions and structs
	limit        *token.Position        // if non-nil, only declarations before this position are returned from LookupDecl
	imports      []*ImportStmt          // used to properly enforce the limit
}

func NewSymbolTable(enclosing *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Enclosing:    enclosing,
		Declarations: make(map[string]Declaration),
		limit:        nil,
	}
}

// wraps the given table with a Limit (replaced if already present)
func (scope *SymbolTable) WithLimit(limit token.Position) *SymbolTable {
	enclosing := scope.Enclosing
	if enclosing != nil {
		enclosing = enclosing.WithLimit(limit)
	}
	return &SymbolTable{
		Enclosing:    enclosing,
		Declarations: scope.Declarations,
		limit:        &limit,
		imports:      scope.imports,
	}
}

// helper to check if a declaration is within the limit of the scope
func (scope *SymbolTable) checkLimit(decl Declaration) bool {
	if scope.limit == nil {
		return true
	}

	importedBy := func(decl Declaration, stmt *ImportStmt) bool {
		if stmt.Module != decl.Module() {
			return false
		}

		if len(stmt.ImportedSymbols) == 0 {
			_, ok := stmt.Module.PublicDecls[decl.Name()]
			return ok
		}
		for i := range stmt.ImportedSymbols {
			if stmt.ImportedSymbols[i].Literal == decl.Name() {
				return true
			}
		}
		return false
	}

	for _, stmt := range scope.imports {
		if importedBy(decl, stmt) {
			return stmt.Range.End.IsBefore(*scope.limit)
		}
	}
	return decl.GetRange().End.IsBefore(*scope.limit)
}

// returns the declaration corresponding to name and wether it exists
// if the symbol existed, the second bool is true, if it is a variable and false if it is a funciton
// call like this: decl, exists, isVar := LookupDecl(name)
func (scope *SymbolTable) LookupDecl(name string) (Declaration, bool, bool) {
	if decl, ok := scope.Declarations[name]; !ok || !scope.checkLimit(decl) {
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
	if decl, ok := scope.Declarations[name]; !ok || !scope.checkLimit(decl) {
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupType(name)
		}
		return nil, false
	} else {
		if structDecl, ok := decl.(*StructDecl); ok {
			return structDecl.Type, true
		}
		return nil, false
	}
}

// adds a new import statement to the tables internal state
// imports are used for limit tracking and are important for ExpressionDecls to work properly
func (scope *SymbolTable) AddImportStmt(stmt *ImportStmt) {
	scope.imports = append(scope.imports, stmt)
}
