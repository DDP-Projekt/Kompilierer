package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type Level uint8

const (
	LEVEL_INFO Level = iota
	LEVEL_WARN
	LEVEL_ERROR
)

// Message type for ddp-errors
type Message struct {
	Code  int         // the error code
	Range token.Range // the range the error spans in its source
	Msg   string      // the error message
	File  string      // the filepath (or uri, url or whatever) in which the error occured
	Level Level       // the level of the error
}

// simple string representation of the error
// should only be used for debugging
// the error-handlers handle the real formatting
func (err Message) String() string {
	return fmt.Sprintf("(%04d) %s (Z: %d, S: %d): %s", err.Code, err.File, err.Range.Start.Line, err.Range.Start.Column, err.Msg)
}

func (err Message) Error() string {
	return err.Msg
}
