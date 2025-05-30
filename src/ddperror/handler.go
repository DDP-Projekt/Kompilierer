package ddperror

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Handler func(Error) // used by most ddp packages

// does nothing
func EmptyHandler(Error) {}

// creates a basic handler that prints the formatted error on a line
func MakeBasicHandler(w io.Writer) Handler {
	return func(err Error) {
		fmt.Fprintf(w, "%s: %s\n", makeErrorHeader(err, ""), err)
	}
}

// creates a rust-like error handler writing to w
// where src is the source-code from which the errors come
// and file is the filename where src comes from
func MakeAdvancedHandler(file string, src []byte, w io.Writer) Handler {
	lines := strings.Split(string(src), "\n")
	file = filepath.Clean(file)
	basicHandler := MakeBasicHandler(w)

	return func(err Error) {
		// we don't have the text of included files
		// so we handle them with the basic error handler
		if filepath.Clean(err.File) != file {
			basicHandler(err)
			return
		}

		// helper to find the maximum of two uints
		uMax := func(a, b uint) uint {
			if a > b {
				return a
			}
			return b
		}

		rnge := err.Range
		maxLineCount, maxLineNumLen := 0, utf8.RuneCountInString(fmt.Sprintf("%d", uMax(rnge.Start.Line, rnge.End.Line)))
		fmt.Fprintf(w, "%s\n\n", makeErrorHeader(err, filepath.Dir(file)))

		for lineIndex := rnge.Start.Line - 1; lineIndex < rnge.End.Line; lineIndex++ {
			replaceAndCount := func(slice []rune) int {
				return utf8.RuneCountInString(strings.TrimRight(strings.ReplaceAll(string(slice), "\t", "    "), "\r"))
			}

			line := []rune(lines[lineIndex])
			head := fmt.Sprintf("%*d |  ", maxLineNumLen, lineIndex+1)
			printLine := strings.TrimRight(strings.ReplaceAll(string(line), "\t", "    "), "\r")
			lineLen := utf8.RuneCountInString(printLine)
			lineStart := utf8.RuneCountInString(printLine) - utf8.RuneCountInString(strings.TrimLeft(printLine, " "))
			fmt.Fprintf(w, "%s%s\n", head, printLine)

			if lineLen > maxLineCount {
				maxLineCount = lineLen
			}

			fmt.Fprintf(w, "%*s |  ", maxLineNumLen, "")

			if lineIndex == rnge.Start.Line-1 {
				startLen := replaceAndCount(line[:rnge.Start.Column-1])
				printN(w, startLen, " ")
				restLen := replaceAndCount(line[rnge.Start.Column-1:])
				if rnge.Start.Line == rnge.End.Line {
					restLen = replaceAndCount(line[rnge.Start.Column-1 : rnge.End.Column-1])
				}
				printN(w, restLen, "^")
			} else if lineIndex < rnge.End.Line-1 {
				printN(w, lineStart, " ")
				printN(w, lineLen-lineStart, "^")
			} else {
				restLen := replaceAndCount(line[:rnge.End.Column-1])
				if lineStart < restLen {
					printN(w, lineStart, " ")
					printN(w, lineLen-lineStart, "^")
				} else {
					printN(w, restLen, "^")
				}
			}
			fmt.Fprint(w, "\n")
		}

		fmt.Fprintf(w, "\n%s.\n\n", err)
		printN(w, maxLineCount, "-")
		fmt.Fprint(w, "\n\n")
	}
}

// creates a Handler that panics with the passed ddperror.Error if called
func MakePanicHandler() Handler {
	return func(err Error) {
		panic(err)
	}
}

// helper to create the common error header of all handlers
// prints the error type, code and place
func makeErrorHeader(err Error, file string) string {
	kind := "Fehler"
	prefix := err.Code.ErrorPrefix()
	if err.Level == LEVEL_WARN {
		kind = "Warnung"
		prefix = err.Code.WarningPrefix()
	}
	if path, e := filepath.Rel(file, err.File); e != nil {
		file = err.File
	} else {
		file = path
	}
	return fmt.Sprintf("%s %s (%04d) in %s (Z: %d, S: %d)",
		prefix,
		kind,
		err.Code,
		file,
		err.Range.Start.Line,
		err.Range.Start.Column,
	)
}

// a handler that captures it's errores
// used for testing
type Collector struct {
	Errors []Error
}

// wether the handler was called
func (m *Collector) DidError() bool {
	return len(m.Errors) != 0
}

// creates a Handler for the given MockHandler
func (m *Collector) GetHandler() Handler {
	return func(err Error) {
		m.Errors = append(m.Errors, err)
	}
}

// helper function to print s n-times
func printN(w io.Writer, n int, s string) {
	for i := 0; i < n; i++ {
		fmt.Fprint(w, s)
	}
}
