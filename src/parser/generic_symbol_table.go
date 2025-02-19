package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
)

type genericSymbolTable struct {
	parserTable  ast.SymbolTable
	contextTable ast.SymbolTable
	declarations map[string]ast.Declaration
}

func newGenericSymbolTable(parserTable, contextTable ast.SymbolTable) ast.SymbolTable {
	return genericSymbolTable{parserTable: parserTable, contextTable: contextTable, declarations: make(map[string]ast.Declaration)}
}

func (s genericSymbolTable) Enclosing() ast.SymbolTable {
	return nil
}

func (s genericSymbolTable) LookupDecl(name string) (ast.Declaration, bool, bool) {
	decl, exists := s.declarations[name]
	isVar := ast.IsVarConstDecl(decl)

	if !exists {
		decl, exists, isVar = s.contextTable.LookupDecl(name)
	}
	if !exists {
		decl, exists, isVar = s.parserTable.LookupDecl(name)
		if isVar {
			return nil, false, false
		}
	}
	return decl, exists, isVar
}

func (s genericSymbolTable) InsertDecl(name string, decl ast.Declaration) bool {
	_, exists, _ := s.LookupDecl(name)
	if exists {
		return true
	}
	s.declarations[name] = decl
	return false
}

func (s genericSymbolTable) LookupType(name string) (ddptypes.Type, bool) {
	if decl, ok := s.declarations[name]; !ok {
		typ, ok := s.contextTable.LookupType(name)
		if !ok {
			typ, ok = s.parserTable.LookupType(name)
		}
		return typ, ok
	} else {
		return ast.IsTypeDecl(decl)
	}
}
