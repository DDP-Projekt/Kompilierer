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
	var visitorFunc typeDeclVisitor
	visitorFunc = func(t ddptypes.Type, m *ast.Module) {
		if generic, isGeneric := ddptypes.CastGenericStructType(t); isGeneric {
			for _, instantitation := range generic.Instantiations {
				visitorFunc(instantitation, module)
			}
		}
		result[t] = m
	}
	ast.VisitModuleRec(module, typeDeclVisitor(visitorFunc))
	return result
}
