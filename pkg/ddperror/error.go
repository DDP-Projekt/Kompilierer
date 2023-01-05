package ddperror

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// Error type for ddp-errors
type Error struct {
	Range token.Range // the range the error spans
	Msg   string      // the error message
	File  string      // the filepath in which the error occured
}

func (err Error) String() string {
	return fmt.Sprintf("%s (Z %d S %d): %s", err.File, err.Range.Start.Line, err.Range.Start.Column, err.Msg)
}
