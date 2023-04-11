package scanner

import (
	"errors"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// Options on where to get the source-tokens from
type Options struct {
	// Optional Filename to name the source
	// this file is read if Source is nil
	FileName string
	// Optional ddp-source-code
	// if nil, FileName is read
	Source []byte
	// the mode used during scanning
	ScannerMode Mode
	// ErrorHandler used during scanning
	// May be nil
	ErrorHandler ddperror.Handler
}

func validateOptions(options *Options) error {
	if options.Source == nil && options.FileName == "" {
		return errors.New("Kein Quellcode gegeben")
	}
	if options.ScannerMode == ModeAlias {
		return errors.New("Benutze scanner.ScanAlias um einen Alias zu scannen")
	}
	if options.ErrorHandler == nil {
		options.ErrorHandler = ddperror.EmptyHandler
	}
	return nil
}

// scan the provided ddp-source-code from the given Options
// if an error occured the resulting tokens are nil
func Scan(options Options) ([]token.Token, error) {
	if err := validateOptions(&options); err != nil {
		return nil, err
	}

	if scan, err := New(options.FileName, options.Source, options.ErrorHandler, options.ScannerMode); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}

// scans the provided source as a function alias
// expects the alias without the enclosing ""
func ScanAlias(alias token.Token, errorHandler ddperror.Handler) ([]token.Token, error) {
	if scan, err := New("Alias", []byte(ast.TrimStringLit(alias)), errorHandler, ModeAlias); err != nil {
		return nil, err
	} else {
		scan.file, scan.line, scan.column, scan.indent = alias.File, alias.Line(), alias.Column(), alias.Indent
		return scan.ScanAll(), nil
	}
}
