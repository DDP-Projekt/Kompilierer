package scanner

import "DDP/pkg/token"

// scans the provided file
func ScanFile(path string, errorHandler ErrorHandler, mode Mode) ([]token.Token, error) {
	if scan, err := New(path, nil, errorHandler, mode); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}

// scans the provided source
func ScanSource(name string, src []byte, errorHandler ErrorHandler, mode Mode) ([]token.Token, error) {
	if scan, err := New(name, src, errorHandler, mode); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}

// scans the provided source as a function alias
// expects the alias without the enclosing ""
func ScanAlias(alias []byte, errorHandler ErrorHandler) ([]token.Token, error) {
	if scan, err := New("Alias", alias, errorHandler, ModeAlias); err != nil {
		return nil, err
	} else {
		return scan.ScanAll(), nil
	}
}
