package ast

import "strings"

// check if the function is inbuilt
func IsInbuiltFunc(fun *FuncDecl) bool {
	return strings.HasPrefix(fun.Name.Literal, "§")
}
