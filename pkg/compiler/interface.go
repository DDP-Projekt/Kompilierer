package compiler

import (
	"errors"
	"io"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/parser"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
)

// compile the given source or file to llvm ir
// if src is not nil it is used, otherwise the path in file is read
// returns the resulting llvm ir as string
func Compile(file string, src []byte, errorHandler scanner.ErrorHandler) (string, *CompileResult, error) {
	var Ast *ast.Ast
	var err error

	if src != nil {
		Ast, err = parser.ParseSource(file, src, errorHandler)
	} else {
		Ast, err = parser.ParseFile(file, errorHandler)
	}

	if err != nil {
		return "", nil, err
	}

	return New(Ast, errorHandler).Compile(nil)
}

// compile the given source or file to llvm ir
// if src is not nil it is used, otherwise the path in file is read
// writes the resulting llvm ir to w
func CompileTo(file string, src []byte, errorHandler scanner.ErrorHandler, writer io.Writer) (*CompileResult, error) {
	if writer == nil {
		return nil, errors.New("w was nil")
	}

	var Ast *ast.Ast
	var err error

	if src != nil {
		Ast, err = parser.ParseSource(file, src, errorHandler)
	} else {
		Ast, err = parser.ParseFile(file, errorHandler)
	}

	if err != nil {
		return nil, err
	}

	_, result, err := New(Ast, errorHandler).Compile(writer)
	return result, err
}

// compile the given AST to llvm ir
func CompileAst(Ast *ast.Ast, errorHandler scanner.ErrorHandler) (string, *CompileResult, error) {
	return New(Ast, errorHandler).Compile(nil)
}

// compile the given AST to llvm ir
func CompileAstTo(Ast *ast.Ast, errorHandler scanner.ErrorHandler, writer io.Writer) (*CompileResult, error) {
	if writer == nil {
		return nil, errors.New("w was nil")
	}
	_, result, err := New(Ast, errorHandler).Compile(writer)
	return result, err
}
