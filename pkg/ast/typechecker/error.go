package typechecker

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type TypecheckerError struct {
	rang token.Range
	file string
	msg  string
}

func (err *TypecheckerError) String() string {
	return err.Msg()
}

func (err *TypecheckerError) Error() string {
	return err.String()
}

func (err *TypecheckerError) GetRange() token.Range {
	return err.rang
}

func (err *TypecheckerError) Msg() string {
	return err.msg
}

func (err *TypecheckerError) File() string {
	return err.file
}
