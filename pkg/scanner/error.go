package scanner

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type ScannerError struct {
	rang token.Range
	file string
	msg  string
}

func (err *ScannerError) String() string {
	return err.Msg()
}

func (err *ScannerError) Error() string {
	return err.String()
}

func (err *ScannerError) GetRange() token.Range {
	return err.rang
}

func (err *ScannerError) Msg() string {
	return err.msg
}

func (err *ScannerError) File() string {
	return err.file
}
