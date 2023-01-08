package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// Error type for ddp-errors
type Error struct {
	Code  Code        // the error code
	Range token.Range // the range the error spans in its source
	Msg   string      // the error message
	File  string      // the filepath (or uri, url or whatever) in which the error occured
}

// simple string representation of the error
// should only be used for debugging
// the error-handlers handle the real formatting
func (err Error) String() string {
	return fmt.Sprintf("(%04d) %s (Z: %d, S: %d): %s", err.Code, err.File, err.Range.Start.Line, err.Range.Start.Column, err.Msg)
}

// create a new Error from the given parameters
func New(code Code, Range token.Range, msg, file string) Error {
	return Error{
		Code:  code,
		Range: Range,
		Msg:   msg,
		File:  file,
	}
}
