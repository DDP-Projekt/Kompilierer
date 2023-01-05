package ddperror

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

type Handler func(Error) // used by most ddp packages

// does nothing
func EmptyHandler(Error) {}

// basic handler that prints the formatted error on a line
func MakeBasicHandler(w io.Writer) Handler {
	return func(err Error) {
		fmt.Fprintf(w, "Fehler in %s in Zeile %d, Spalte %d: %s\n", err.File, err.Range.Start.Line, err.Range.Start.Column, err.Msg)
	}
}

// creates a rust-like error handler with src writing to w
func MakeAdvancedHandler(file string, src []byte, w io.Writer) Handler {
	lines := strings.Split(string(src), "\n")

	return func(err Error) {
		if err.File != file {
			fmt.Fprintf(w, "Fehler in %s in Zeile %d, Spalte %d: %s\n", err.File, err.Range.Start.Line, err.Range.Start.Column, err.Msg)
			return
		}

		printN := func(n int, s string) {
			for i := 0; i < n; i++ {
				fmt.Fprint(w, s)
			}
		}

		rnge := err.Range
		maxLineCount, maxLineNumLen := 0, utf8.RuneCountInString(fmt.Sprintf("%d", uMax(rnge.Start.Line, rnge.End.Line)))
		fmt.Fprintf(w, "Fehler in %s (Z %d, S %d)\n\n", err.File, rnge.Start.Line, rnge.Start.Column)

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
				printN(startLen, " ")
				restLen := replaceAndCount(line[rnge.Start.Column-1:])
				if rnge.Start.Line == rnge.End.Line {
					restLen = replaceAndCount(line[rnge.Start.Column-1 : rnge.End.Column-1])
				}
				printN(restLen, "^")
			} else if lineIndex < rnge.End.Line-1 {
				printN(lineStart, " ")
				printN(lineLen-lineStart, "^")
			} else {
				restLen := replaceAndCount(line[:rnge.End.Column-1])
				if lineStart < restLen {
					printN(lineStart, " ")
					printN(lineLen-lineStart, "^")
				} else {
					printN(restLen, "^")
				}
			}
			fmt.Fprint(w, "\n")
		}

		fmt.Fprintf(w, "\n%s.\n\n", err.Msg)
		printN(maxLineCount, "-")
		fmt.Fprint(w, "\n\n")
	}
}

func uMax(a, b uint) uint {
	if a > b {
		return a
	}
	return b
}
