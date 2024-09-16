package ast

import (
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// represents a single DDP Module (source file),
// it's dependencies and public interface
type Module struct {
	// the absolute filepath from which the module comes
	FileName string
	// all the imported modules
	Imports []*ImportStmt
	// First token in the file if present, or nil otherwise
	Comment *token.Token
	// a set which contains all files needed
	// to link the final executable
	// contains .c, .lib, .a and .o files
	ExternalDependencies map[string]struct{}
	// the Ast of the Module
	Ast *Ast
	// map of references to all public functions, variables and structs
	PublicDecls map[string]Declaration
}

// Calls fun with module and every module it imports recursively
// meaning fun is called on module, all of its imports and all of their imports and so on
// a single time per module
func IterateModuleImports(module *Module, fun func(*Module)) {
	if module != nil {
		iterateModuleImportsRec(module, fun, make(map[*Module]struct{}, len(module.Imports)+1))
	}
}

func iterateModuleImportsRec(module *Module, fun func(*Module), visited map[*Module]struct{}) {
	// return if already visited
	if _, ok := visited[module]; ok {
		return
	}
	visited[module] = struct{}{}
	for _, imprt := range module.Imports {
		if imprt.Module != nil {
			iterateModuleImportsRec(imprt.Module, fun, visited)
		}
	}
	fun(module)
}
