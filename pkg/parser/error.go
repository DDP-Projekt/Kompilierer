package parser

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type ParserError struct {
	rang token.Range
	file string
	msg  string
}

func (err *ParserError) String() string {
	return err.Msg()
}

func (err *ParserError) Error() string {
	return err.String()
}

func (err *ParserError) GetRange() token.Range {
	return err.rang
}

func (err *ParserError) Msg() string {
	return err.msg
}

func (err *ParserError) File() string {
	return err.file
}
