package ast

import (
	"fmt"
	"sort"
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
		if imprt.Module == nil {
			return
		}

		decls := make([]Declaration, 0, len(imprt.Module.PublicDecls))
		for _, decl := range imprt.Module.PublicDecls {
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

// originally in ddperror/messages.go but moved here to avoid circular imports
func MsgAliasAlreadyExists(alias Alias) string {
	typ := "die Funktion"
	switch alias.(type) {
	case *StructAlias:
		typ = "die Struktur"
	case *ExpressionAlias:
		typ = "den Ausdruck"
	default:
	}
	return fmt.Sprintf("Der Alias %s steht bereits für %s '%s'", alias.GetOriginal().Literal, typ, alias.Decl().Name())
}
