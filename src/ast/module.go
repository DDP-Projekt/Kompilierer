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
	// the token which specified the relative FileName
	// if the module was imported and not the main Module
	FileNameToken *token.Token
	// all the imported modules
	Imports []*ImportStmt
	// a set which contains all files needed
	// to link the final executable
	// contains .c, .lib, .a and .o files
	ExternalDependencies map[string]struct{}
	// the Ast of the Module
	Ast *Ast
	// map of references to all public functions and variables
	PublicDecls map[string]Declaration
	// map of references to all public struct decls
	PublicStructDecls map[string]*StructDecl
}

// returns the string-literal content by which this module was first imported
// or the short FileName if it is the main module
func (module *Module) GetIncludeFilename() string {
	if module.FileNameToken == nil {
		return filepath.Base(module.FileName)
	}
	return TrimStringLit(*module.FileNameToken)
}

// calls VisitAst on all Asts of module and it's imports
func VisitModuleRec(module *Module, visitor BaseVisitor) {
	visitModuleRec(module, visitor, make(map[*Module]struct{}))
}

func visitModuleRec(module *Module, visitor BaseVisitor, visited map[*Module]struct{}) {
	// return if already visited
	if _, ok := visited[module]; ok {
		return
	}
	visited[module] = struct{}{}
	for _, imprt := range module.Imports {
		if imprt.Module != nil {
			visitModuleRec(imprt.Module, visitor, visited)
		}
	}
	VisitAst(module.Ast, visitor)
}
