// range.go defines types to work with token positions in an AST
// these types are not used by Token itself, but are meant to be used
// by code-analysis tools
package token

import "unicode/utf8"

// a position in a ddp source-file
type Position struct {
	Line   int
	Column int
}

// a range in a ddp source-file
type Range struct {
	Start Position
	End   Position
}

// creates a new range from the first character of begin
// to the last character of end
func NewRange(begin, end Token) Range {
	return Range{
		Start: Position{
			Line:   begin.Line,
			Column: begin.Column,
		},
		End: Position{
			Line:   end.Line,
			Column: end.Column + utf8.RuneCountInString(end.Literal),
		},
	}
}

func NewStartPos(tok Token) Position {
	return Position{
		Line:   tok.Line,
		Column: tok.Column,
	}
}

func NewEndPos(tok Token) Position {
	return Position{
		Line:   tok.Line,
		Column: tok.Column + utf8.RuneCountInString(tok.Literal),
	}
}
