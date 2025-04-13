package ddperror

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/token"
)

type Level uint8

const (
	LEVEL_INVALID Level = iota
	LEVEL_WARN
	LEVEL_ERROR
)

// Error type for ddp-errors
type Error struct {
	Code                 Code        // the error code
	Range                token.Range // the range the error spans in its source
	Msg                  string      // the error message
	File                 string      // the filepath (or uri, url or whatever) in which the error occured
	Level                Level       // the level of the error
	WrappedGenericErrors []Error     // optional slice of generic errors wrapped by this error
}

// simple string representation of the error
// should only be used for debugging
// the error-handlers handle the real formatting
func (err Error) String() string {
	msg := strings.Builder{}
	printIndentedError(&msg, err, 0)
	return msg.String()
}

const ERROR_INDENT = "    "

func printIndentedError(builder *strings.Builder, err Error, indent int) {
	printN(builder, indent, ERROR_INDENT)

	if indent == 0 {
		builder.WriteString(err.Msg)
	} else {
		fmt.Fprintf(builder, "(%04d) %s (Z: %d, S: %d): %s", err.Code, shortFileName(err.File), err.Range.Start.Line, err.Range.Start.Column, err.Msg)
	}

	if len(err.WrappedGenericErrors) != 0 {
		builder.WriteString(":")
		for _, err := range err.WrappedGenericErrors {
			builder.WriteString("\n")
			printIndentedError(builder, err, indent+1)
			if len(err.WrappedGenericErrors) != 0 {
				builder.WriteString("\n")
			}
		}
	}
}

func shortFileName(s string) string {
	return filepath.Base(s)
}

func (err Error) Error() string {
	return err.String()
}

// create a new Error from the given parameters
func New(code Code, level Level, Range token.Range, msg, file string) Error {
	return Error{
		Code:  code,
		Range: Range,
		Msg:   msg,
		File:  file,
		Level: level,
	}
}
