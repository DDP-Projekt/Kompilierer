package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/compiler/llvm"
)

// wraps a ir ir alloca + ir type for a variable
type varwrapper struct {
	val        llvm.Value // alloca or global-Def in the ir
	typ        ddpIrType  // ir type of the variable
	restoreLoc llvm.Value // restore location for manually added GC pointers
	// protected temporaries or variables should not be freed in exitScope()
	// either because they will be freed by hand (e.g. like in ForRangeStmt)
	// or because they are only recored to be in the LiveValues list for GC
	protected bool
	// weak_protected is a special case for for-range temporaries, which should
	// be freed on return statements, unlike GC LiveValues, which are never freed
	weak_protected bool
}

// wraps local variables of a scope + the enclosing scope
type scope struct {
	enclosing   *scope                      // enclosing scope, nil if it is the global scope
	variables   map[*ast.VarDecl]varwrapper // variables in this scope
	temporaries []varwrapper                // intermediate values that need to be freed when the scope ends
}

// create a new scope in the enclosing scope
func newScope(enclosing *scope) *scope {
	return &scope{
		enclosing: enclosing,
		variables: make(map[*ast.VarDecl]varwrapper),
	}
}

func (s *scope) isGlobalScope() bool {
	return s.enclosing == nil
}

// returns the named variable
// if not present the enclosing scopes are checked
// until the global scope
func (s *scope) lookupVar(decl *ast.VarDecl) varwrapper {
	if v, ok := s.variables[decl]; !ok {
		if s.enclosing != nil {
			return s.enclosing.lookupVar(decl)
		}
		return varwrapper{val: llvm.Value{}, typ: nil} // variable doesn't exist (should not happen, resolver should take care of that)
	} else {
		return v
	}
}

// add a variable to the scope
func (scope *scope) addVar(decl *ast.VarDecl, val llvm.Value, ty ddpIrType) llvm.Value {
	scope.variables[decl] = varwrapper{val: val, typ: ty, protected: false}
	return val
}

func (scope *scope) addProtected(decl *ast.VarDecl, val llvm.Value, ty ddpIrType) llvm.Value {
	scope.variables[decl] = varwrapper{val: val, typ: ty, protected: true}
	return val
}

func (scope *scope) protectTemporary(val llvm.Value, weak bool) {
	for i := len(scope.temporaries) - 1; i >= 0; i-- {
		if scope.temporaries[i].val == val {
			scope.temporaries[i].protected = true
			scope.temporaries[i].weak_protected = weak
			return
		}
	}
	panic("attempted Value protection not found in scope.temporaries")
}

func (scope *scope) unprotectTemporary(val llvm.Value) {
	for i := len(scope.temporaries) - 1; i >= 0; i-- {
		if scope.temporaries[i].val == val {
			scope.temporaries[i].protected = false
			scope.temporaries[i].weak_protected = false
			return
		}
	}
	panic("attempted Value unprotection not found in scope.temporaries")
}

func (scope *scope) addRestoreLoc(val, restore llvm.Value) {
	for i := len(scope.temporaries) - 1; i >= 0; i-- {
		if scope.temporaries[i].val == val {
			scope.temporaries[i].restoreLoc = restore
			return
		}
	}
	panic("attempted Value restoreLoc addition not found in scope.temporaries")
}

func (scope *scope) addTemporary(val llvm.Value, typ ddpIrType) ddpValue {
	scope.temporaries = append(scope.temporaries, varwrapper{val: val, typ: typ, protected: false})
	return newImmediate(val, typ)
}

// mainly used for adding values that need to be recorded as live for the GC
func (scope *scope) addProtectedTemporary(val llvm.Value, typ ddpIrType) ddpValue {
	temp := scope.addTemporary(val, typ)
	scope.protectTemporary(val, false)
	return temp
}

func (scope *scope) addWeakProtectedTemporary(val llvm.Value, typ ddpIrType) ddpValue {
	temp := scope.addTemporary(val, typ)
	scope.protectTemporary(val, true)
	return temp
}

func (scope *scope) addGCLiveValue(val, restore llvm.Value) llvm.Value {
	scope.temporaries = append(scope.temporaries, varwrapper{val: val, typ: nil, restoreLoc: restore, protected: true})
	return val
}

// removes the given value from scope.temporaries giving ownership to the caller
// who now has to make sure it is freed
// returns the value
func (scope *scope) claimTemporary(val llvm.Value) llvm.Value {
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
