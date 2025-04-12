package ast

import (
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type OperatorOverloadMap = map[Operator][]*FuncDecl

// represents a single DDP Module (source file),
// it's dependencies and public interface
type Module struct {
	// the absolute filepath from which the module comes
	FileName string
	// the token which specified the relative FileName
	// if the module was imported and not the main Module
	FileNameToken *token.Token
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
	// map of all overloads for all operators
	Operators OperatorOverloadMap
}

// returns the string-literal content by which this module was first imported
// or the short FileName if it is the main module
func (module *Module) GetIncludeFilename() string {
	if module.FileNameToken == nil {
		return filepath.Base(module.FileName)
	}
	return TrimStringLit(module.FileNameToken)
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
		for _, mod := range imprt.Modules {
			iterateModuleImportsRec(mod, fun, visited)
		}
	}
	fun(module)
}
