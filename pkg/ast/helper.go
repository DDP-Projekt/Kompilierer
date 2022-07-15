package ast

import "strings"

// check if the function is inbuilt
func IsInbuiltFunc(fun *FuncDecl) bool {
	return strings.HasPrefix(fun.Name.Literal, "§")
}

// check if the function is defined externally
func IsExternFunc(fun *FuncDecl) bool {
	return fun.Body == nil
}
