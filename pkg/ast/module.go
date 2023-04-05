package ast

// represents a single DDP Module (source file),
// it's dependencies and public interface
type Module struct {
	// the absolute filepath from which the module comes
	FileName string
	// all the imported modules mapped by Module.FileName
	Imports []*ImportStmt
	// a set which contains all files needed
	// to link the final executable
	// contains .c, .lib, .a and .o files
	ExternalDependencies map[string]struct{}
	// the Ast of the Module
	Ast *Ast
	// map of all public functions and variables
	PublicDecls map[string]Declaration
}
