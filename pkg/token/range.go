// range.go defines types to work with token positions in an AST
// these types are not used by Token itself, but are meant to be used
// by code-analysis tools
package token

// a position in a ddp source-file
// Line and Column are 1-based
// and measured in utf8-characters not bytes
type Position struct {
	Line   uint // 1-based Line index in the corresponding file
	Column uint // 1-based Column index in the corresponding file
}

// a range in a ddp source-file
type Range struct {
	Start Position // First Character position in the Range
	End   Position // Last Character position in the Range
}

// creates a new range from the first character of begin
// to the last character of end
func NewRange(begin, end Token) Range {
	return Range{
		Start: begin.Range.Start,
		End:   end.Range.End,
	}
}

// Get the starting position of a Token
func NewStartPos(tok Token) Position {
	return tok.Range.Start
}

// Get the ending position of a Token
func NewEndPos(tok Token) Position {
	return tok.Range.End
}
