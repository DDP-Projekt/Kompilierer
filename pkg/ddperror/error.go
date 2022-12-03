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
