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
func TrimStringLit(lit token.Token) string {
	return strings.Trim(lit.Literal, "\"")
}
