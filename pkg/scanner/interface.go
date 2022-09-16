package scanner

import (
	"strings"

	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// scans the provided file
func ScanFile(path string, errorHandler ddperror.Handler, mode Mode) ([]token.Token, error) {
	if scan, err := New(path, nil, errorHandler, mode); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}

// scans the provided source
func ScanSource(name string, src []byte, errorHandler ddperror.Handler, mode Mode) ([]token.Token, error) {
	if scan, err := New(name, src, errorHandler, mode); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}

// scans the provided source as a function alias
// expects the alias without the enclosing ""
func ScanAlias(alias token.Token, errorHandler ddperror.Handler) ([]token.Token, error) {
	if scan, err := New("Alias", []byte(strings.Trim(alias.Literal, "\"")), errorHandler, ModeAlias); err != nil {
		return nil, err
	} else {
		scan.file, scan.line, scan.column, scan.indent = alias.File, alias.Line(), alias.Column(), alias.Indent
		return scan.ScanAll(), nil
	}
}
