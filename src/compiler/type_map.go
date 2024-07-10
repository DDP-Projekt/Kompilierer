package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type structDeclVisitor func(decl *ast.StructDecl)

var (
	_ ast.Visitor           = structDeclVisitor(nil)
	_ ast.StructDeclVisitor = structDeclVisitor(nil)
)

func (structDeclVisitor) Visitor() {}

func (f structDeclVisitor) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	f(decl)
	return ast.VisitSkipChildren
}

// returns a map of all struct types mapped to their origin module
func createTypeMap(module *ast.Module) map[ddptypes.Type]*ast.Module {
	result := make(map[ddptypes.Type]*ast.Module, 8)
	ast.VisitModuleRec(module, structDeclVisitor(func(decl *ast.StructDecl) {
		result[decl.Type] = decl.Module()
	}))
	return result
}
