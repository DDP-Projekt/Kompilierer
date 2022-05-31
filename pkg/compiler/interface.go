package compiler

import (
	"KDDP/pkg/ast"
	"KDDP/pkg/parser"
	"KDDP/pkg/scanner"
)

// compile the given ddp file into llvm ir
func CompileFile(path string, errorHandler scanner.ErrorHandler) (string, error) {
	Ast, err := parser.ParseFile(path, errorHandler)
	if err != nil {
		return "", err
	}
	return New(Ast, errorHandler).Compile()
}

// compile the given ddp source code into llvm ir
func CompileSource(name string, src []byte, errorHandler scanner.ErrorHandler) (string, error) {
	Ast, err := parser.ParseSource(name, src, errorHandler)
	if err != nil {
		return "", err
	}
	return New(Ast, errorHandler).Compile()
}

// compile the given AST to llvm ir
func CompileAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) (string, error) {
	return New(Ast, errorHandler).Compile()
}
