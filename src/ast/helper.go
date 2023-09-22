package ast

import (
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// check if the function is defined externally
func IsExternFunc(fun *FuncDecl) bool {
	return fun.Body == nil
}

// trims the "" from the literal
func TrimStringLit(lit *token.Token) string {
	if lit == nil {
		return ""
	}
	return strings.Trim(lit.Literal, "\"")
}

// applies fun to all declarations imported by imprt
// if len(imprt.ImportedSymbols) == 0, it is applied to all imprt.Module.PublicDecls
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
		if imprt.Module == nil {
			return
		}

		for name, decl := range imprt.Module.PublicDecls {
			if !fun(name, decl, imprt.FileName) {
				break
			}
		}
	} else {
		for _, name := range imprt.ImportedSymbols {
			var decl Declaration
			if imprt.Module != nil {
				decl = imprt.Module.PublicDecls[name.Literal]
			}
			if !fun(name.Literal, decl, name) {
				break
			}
		}
	}
}
