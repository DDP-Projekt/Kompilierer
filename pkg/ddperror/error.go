package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// ddp-error interface for all
// ddp packages to use
type Error interface {
	fmt.Stringer
	error
	GetRange() token.Range // the range the error spans
	Msg() string           // the error message
	File() string          // the filepath in which the error occured
}

type Handler func(Error) // used by most ddp packages

// Prints the error on a line
func DefaultHandler(err Error) {
	fmt.Printf("Fehler in %s in Zeile %d, Spalte %d: %s\n", err.File(), err.GetRange().Start.Line, err.GetRange().Start.Column, err.Msg())
}

// does nothing
func EmptyHandler(Error) {}
