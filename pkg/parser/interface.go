package parser

import (
	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/ast"
	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/scanner"
	"github.com/Die-Deutsche-Programmiersprache/KDDP/pkg/token"
)

// parse the provided file into an Ast
func ParseFile(path string, errorHandler scanner.ErrorHandler) (*ast.Ast, error) {
	file, err := scanner.ScanFile(path, errorHandler, scanner.ModeStrictCapitalization)
	if err != nil {
		return nil, err
	}
	return New(file, errorHandler).Parse(), nil
}

// parse the provided source into an Ast
func ParseSource(name string, src []byte, errorHandler scanner.ErrorHandler) (*ast.Ast, error) {
	file, err := scanner.ScanSource(name, src, errorHandler, scanner.ModeStrictCapitalization)
	if err != nil {
		return nil, err
	}
	return New(file, errorHandler).Parse(), nil
}

// parse the provided tokens into an Ast
func ParseTokens(tokens []token.Token, errorHandler scanner.ErrorHandler) *ast.Ast {
	return New(tokens, errorHandler).Parse()
}
