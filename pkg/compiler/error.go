package compiler

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type CompilerError struct {
	rang token.Range
	file string
	msg  string
}

func (err *CompilerError) String() string {
	return err.Msg()
}

func (err *CompilerError) Error() string {
	return err.String()
}

func (err *CompilerError) GetRange() token.Range {
	return err.rang
}

func (err *CompilerError) Msg() string {
	return err.msg
}

func (err *CompilerError) File() string {
	return err.file
}
