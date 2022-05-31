package interpreter

import (
	"KDDP/pkg/ast"
	"KDDP/pkg/parser"
	"KDDP/pkg/scanner"
)

// interpret the given ddp file
func InterpretFile(path string, errorHandler scanner.ErrorHandler) error {
	Ast, err := parser.ParseFile(path, errorHandler)
	if err != nil {
		return err
	}
	return New(Ast, errorHandler).Interpret()
}

// interpret the given ddp source code
func InterpretSource(name string, src []byte, errorHandler scanner.ErrorHandler) error {
	Ast, err := parser.ParseSource(name, src, errorHandler)
	if err != nil {
		return err
	}
	return New(Ast, errorHandler).Interpret()
}

// interpret the given AST
func InterpretAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) error {
	return New(Ast, errorHandler).Interpret()
}
