package ast

// stores symbols for one scope of an ast
type SymbolTable struct {
	Enclosing *SymbolTable         // enclosing scope (nil in the global scope)
	Variables map[string]*VarDecl  // tokenType is used as type identifier (e.g. token.ZAHL -> int)
	Functions map[string]*FuncDecl // same here, but token.NICHTS stands for void (also, only the global-scope can have function declarations)
}

func NewSymbolTable(enclosing *SymbolTable) *SymbolTable {
	return &SymbolTable{
		Enclosing: enclosing,
		Variables: make(map[string]*VarDecl),
		Functions: make(map[string]*FuncDecl),
	}
}

// returns the type of the variable name and if it exists in the table or it's enclosing scopes
func (scope *SymbolTable) LookupVar(name string) (*VarDecl, bool) {
	if val, ok := scope.Variables[name]; !ok {
		// if the variable was not found here we recursively check the enclosing scopes
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupVar(name)
		}
		return nil, false // variable doesn't exist
	} else {
		return val, true
	}
}

// returns the type of the variable name and if it exists in the table or it's enclosing scopes
func (scope *SymbolTable) LookupFunc(name string) (*FuncDecl, bool) {
	if val, ok := scope.Functions[name]; !ok {
		// if the function was not found here we recursively check the enclosing scopes
		if scope.Enclosing != nil {
			return scope.Enclosing.LookupFunc(name)
		}
		return nil, false // function doesn't exist
	} else {
		return val, true
	}
}

// inserts a variable into the scope if it didn't exist yet
// and returns wether it already existed
func (scope *SymbolTable) InsertVar(name string, decl *VarDecl) bool {
	if _, ok := scope.Variables[name]; ok {
		return true
	}

	scope.Variables[name] = decl
	return false
}

// inserts a function into the scope if it didn't exist yet
// and returns wether it already existed
func (scope *SymbolTable) InsertFunc(name string, fun *FuncDecl) bool {
	if _, ok := scope.Functions[name]; ok {
		return true
	}

	scope.Functions[name] = fun
	return false
}

// merge other into scope, replacing already existing symbols
func (scope *SymbolTable) Merge(other *SymbolTable) {
	for k, val := range other.Functions {
		scope.Functions[k] = val
	}

	for k, val := range other.Variables {
		scope.Variables[k] = val
	}
}

// return a copy of scope
// not a deep copy, FuncDecl pointers stay the same
func (scope *SymbolTable) Copy() *SymbolTable {
	table := &SymbolTable{
		Enclosing: scope.Enclosing,
		Variables: make(map[string]*VarDecl),
		Functions: make(map[string]*FuncDecl),
	}

	for k, val := range scope.Variables {
		table.Variables[k] = val
	}

	for k, val := range scope.Functions {
		table.Functions[k] = val
	}

	return table
}
