package parser

import (
	"errors"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// Options on where to get the source-tokens from
type Options struct {
	// Optional Filename to name the source
	// this file is read if Source and Tokens are nil
	FileName string
	// Optional ddp-source-code
	// if nil, FileName is read
	Source []byte
	// Optional scanned token slice
	// if nil, Source is used
	Tokens []token.Token
	// ErrorHandler used during scanning and parsing
	// May be nil
	ErrorHandler ddperror.Handler
}

func (options *Options) ToScannerOptions(scannerMode scanner.Mode) scanner.Options {
	return scanner.Options{
		FileName:     options.FileName,
		Source:       options.Source,
		ScannerMode:  scannerMode,
		ErrorHandler: options.ErrorHandler,
	}
}

func validateOptions(options *Options) error {
	if options.Source == nil && options.Tokens == nil && options.FileName == "" {
		return errors.New("Kein Quellcode gegeben")
	}
	if options.ErrorHandler == nil {
		options.ErrorHandler = ddperror.EmptyHandler
	}
	return nil
}

// parse the provided ddp-source-code from the given Options
// if an error occured the resulting Ast is nil
func Parse(options Options) (*ast.Ast, error) {
	// validate the options
	err := validateOptions(&options)
	if err != nil {
		return nil, err
	}

	if options.Tokens == nil {
		options.Tokens, err = scanner.Scan(options.ToScannerOptions(scanner.ModeStrictCapitalization))
		if err != nil {
			return nil, err
		}
	}

	Ast := New(options.Tokens, options.ErrorHandler).Parse()
	if options.FileName != "" {
		Ast.File = options.FileName
	}
	return Ast, nil

}
