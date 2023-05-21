package parser

import (
	"errors"
	"path/filepath"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
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
	// maps modules that have already been passed by their filenames
	// the the parser may add new imported modules to this map
	// so that after passing it contains *at least* all modules
	// that the resulting module depends on
	Modules map[string]*ast.Module
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
	if options.Modules == nil {
		options.Modules = make(map[string]*ast.Module)
	}
	if options.ErrorHandler == nil {
		options.ErrorHandler = ddperror.EmptyHandler
	}
	return nil
}

// parse the provided ddp-source-code from the given Options
// if an error occured the resulting Ast is nil
func Parse(options Options) (*ast.Module, error) {
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

	module := newParser(options.FileName, options.Tokens, options.Modules, options.ErrorHandler).parse()
	if options.FileName != "" {
		path, err := filepath.Abs(options.FileName)
		if err != nil {
			return nil, err
		}
		module.FileName = path
	}
	return module, nil
}
