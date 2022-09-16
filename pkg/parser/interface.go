package parser

import (
	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// parse the provided file into an Ast
func ParseFile(path string, errorHandler ddperror.Handler) (*ast.Ast, error) {
	file, err := scanner.ScanFile(path, errorHandler, scanner.ModeStrictCapitalization)
	if err != nil {
		return nil, err
	}

	Ast := New(file, errorHandler).Parse()
	Ast.File = path
	return Ast, nil
}

// parse the provided source into an Ast
// an optional file name can be provided for better error messages
func ParseSource(name string, src []byte, errorHandler ddperror.Handler) (*ast.Ast, error) {
	file, err := scanner.ScanSource(name, src, errorHandler, scanner.ModeStrictCapitalization)
	if err != nil {
		return nil, err
	}

	Ast := New(file, errorHandler).Parse()
	Ast.File = name
	return Ast, nil
}

// parse the provided tokens into an Ast
func ParseTokens(tokens []token.Token, errorHandler ddperror.Handler) *ast.Ast {
	return New(tokens, errorHandler).Parse()
}
