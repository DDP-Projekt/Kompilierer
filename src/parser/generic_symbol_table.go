package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type genericSymbolTable struct {
	parserTable  ast.SymbolTable
	contextTable ast.SymbolTable
	genericTypes map[string]*ddptypes.InstantiatedGenericType
}

func newGenericSymbolTable(parserTable, contextTable ast.SymbolTable, genericTypes map[string]ddptypes.Type) ast.SymbolTable {
	instantiatedTypes := make(map[string]*ddptypes.InstantiatedGenericType, len(genericTypes))
	for name, typ := range genericTypes {
		instantiatedTypes[name] = &ddptypes.InstantiatedGenericType{Actual: typ}
	}

	return genericSymbolTable{parserTable: parserTable, contextTable: contextTable, genericTypes: instantiatedTypes}
}

func (s genericSymbolTable) Enclosing() ast.SymbolTable {
	return nil
}

func (s genericSymbolTable) LookupDecl(name string) (ast.Declaration, bool, bool) {
	if typ, ok := s.genericTypes[name]; ok {
		name = typ.String()
	}

	decl, exists, isVar := s.contextTable.LookupDecl(name)
	if !exists {
		decl, exists, isVar = s.parserTable.LookupDecl(name)
		if isVar {
			return nil, false, false
		}
	}
	return decl, exists, isVar
}

func (s genericSymbolTable) InsertDecl(name string, decl ast.Declaration) bool {
	return true
}

func (s genericSymbolTable) LookupType(name string) (ddptypes.Type, bool) {
	if typ, ok := s.genericTypes[name]; ok {
		return typ, ok
	}

	if typ, ok := s.contextTable.LookupType(name); ok {
		return typ, ok
	}

	return s.parserTable.LookupType(name)
}
