package ast

import (
	"sort"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// check if the function is defined externally
func IsExternFunc(fun *FuncDecl) bool {
	return fun.Body == nil && fun.Def == nil
}

// check if the declaration is only a forward decl
func IsForwardDecl(fun *FuncDecl) bool {
	return fun.Def != nil
}

func IsOperatorOverload(fun *FuncDecl) bool {
	return fun.Operator != nil
}

// trims the "" from the literal
func TrimStringLit(lit *token.Token) string {
	if lit == nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(lit.Literal, "\""), "\"")
}

// returns wether table is the global scope
// table.Enclosing == nil
func IsGlobalScope(table *SymbolTable) bool {
	return table.Enclosing == nil
}

// applies fun to all declarations imported by imprt
// if len(imprt.ImportedSymbols) == 0, it is applied to all imprt.Module.PublicDecls,
// in which case the declarations are sorted by occurence in the source file
// otherwise to every tok in imprt.ImportedSymbols is used
//
//   - name is the name of the declaration (or the literal of imprt.ImportedSymbols if no decl is found)
//   - decl is nil if imprt.ImportedSymbols[x] is not present in import.Module.PublicDecls
//   - tok refers either to the string literal of the import path,
//     or to the identifier from imprt.ImportedSymbols
//
// the return value of fun indicates wether the iteration should continue or if we break
// true means continue, false means break
func IterateImportedDecls(imprt *ImportStmt, fun func(name string, decl Declaration, tok token.Token) bool) {
	if len(imprt.ImportedSymbols) == 0 {
		if len(imprt.Modules) == 0 {
			return
		}

		for _, module := range imprt.Modules {
			decls := make([]Declaration, 0, len(module.PublicDecls))
			for _, decl := range module.PublicDecls {
				decls = append(decls, decl)
			}

			// sort by occurence in the source file
			sort.Slice(decls, func(i, j int) bool {
				start := decls[i].GetRange().Start
				startj := decls[j].GetRange().Start
				return start.Line < startj.Line || start.Column < startj.Column
			})

			for _, decl := range decls {
				if !fun(decl.Name(), decl, imprt.FileName) {
					break
				}
			}
		}
		return
	}

	for _, name := range imprt.ImportedSymbols {
		var decl Declaration
		if imprt.SingleModule() != nil {
			decl = imprt.SingleModule().PublicDecls[name.Literal]
		}
		if !fun(name.Literal, decl, name) {
			break
		}
	}
}
