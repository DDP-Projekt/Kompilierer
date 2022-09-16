package resolver

import "github.com/DDP-Projekt/Kompilierer/pkg/token"

type ResolverError struct {
	rang token.Range
	file string
	msg  string
}

func (err *ResolverError) String() string {
	return err.Msg()
}

func (err *ResolverError) Error() string {
	return err.String()
}

func (err *ResolverError) GetRange() token.Range {
	return err.rang
}

func (err *ResolverError) Msg() string {
	return err.msg
}

func (err *ResolverError) File() string {
	return err.file
}
