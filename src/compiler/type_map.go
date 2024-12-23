package compiler

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type typeDeclVisitor func(ddptypes.Type, *ast.Module)

var (
	_ ast.Visitor            = typeDeclVisitor(nil)
	_ ast.StructDeclVisitor  = typeDeclVisitor(nil)
	_ ast.TypeDefDeclVisitor = typeDeclVisitor(nil)
)

func (typeDeclVisitor) Visitor() {}

func (f typeDeclVisitor) VisitStructDecl(decl *ast.StructDecl) ast.VisitResult {
	f(decl.Type, decl.Module())
	return ast.VisitSkipChildren
}

func (f typeDeclVisitor) VisitTypeDefDecl(decl *ast.TypeDefDecl) ast.VisitResult {
	f(decl.Type, decl.Module())
	return ast.VisitSkipChildren
}

// returns a map of all struct types mapped to their origin module
func createTypeMap(module *ast.Module) map[ddptypes.Type]*ast.Module {
	result := make(map[ddptypes.Type]*ast.Module, 8)
	ast.VisitModuleRec(module, typeDeclVisitor(func(t ddptypes.Type, m *ast.Module) {
		result[t] = m
	}))
	return result
}
