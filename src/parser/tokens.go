/*
This file contains functions to match and consume tokens
*/
package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// a tokenWalker holds the state needed
// to work with a series of tokens
type tokenWalker struct {
	// the tokens to parse (without comments)
	tokens []token.Token
	// all the comments from the original tokens slice
	comments []token.Token
	// index of the current token
	cur int
	// called to report errors from consume functions
	errFunc func(ddperror.Code, token.Range, string)
}

// if the current tokenType is contained in types, advance
// returns wether we advanced or not
func (w *tokenWalker) matchAny(types ...token.TokenType) bool {
	for _, t := range types {
		if w.check(t) {
			w.advance()
			return true
		}
	}
	return false
}

// if the given sequence of tokens is matched, advance
// returns wether we advance or not
func (w *tokenWalker) matchSeq(types ...token.TokenType) bool {
	for i, t := range types {
		if w.peekN(i).Type != t {
			return false
		}
	}

	for range types {
		w.advance()
	}

	return true
}

// if the current token is of type t advance, otherwise error
func (w *tokenWalker) consume1(t token.TokenType) bool {
	if w.check(t) {
		w.advance()
		return true
	}

	w.errFunc(ddperror.SYN_UNEXPECTED_TOKEN, w.peek().Range, ddperror.MsgGotExpected(w.peek().Literal, t))
	return false
}

// consume a series of tokens
func (w *tokenWalker) consume(t ...token.TokenType) bool {
	for _, v := range t {
		if !w.consume1(v) {
			return false
		}
	}
	return true
}

// same as consume but tolerates multiple tokenTypes
func (w *tokenWalker) consumeAny(tokenTypes ...token.TokenType) bool {
	for _, v := range tokenTypes {
		if w.check(v) {
			w.advance()
			return true
		}
	}

	w.errFunc(ddperror.SYN_UNEXPECTED_TOKEN, w.peek().Range, ddperror.MsgGotExpected(w.peek().Literal, toInterfaceSlice[token.TokenType, any](tokenTypes)...))
	return false
}

// check if the current token is of type t without advancing
func (w *tokenWalker) check(t token.TokenType) bool {
	if w.atEnd() {
		return false
	}
	return w.peek().Type == t
}

// check if the current token is EOF
func (w *tokenWalker) atEnd() bool {
	return w.peek().Type == token.EOF
}

// return the current token and advance p.cur
func (w *tokenWalker) advance() *token.Token {
	if !w.atEnd() {
		w.cur++
		return w.previous()
	}
	return w.peek() // return EOF
}

// returns the current token without advancing
func (w *tokenWalker) peek() *token.Token {
	return &w.tokens[w.cur]
}

// returns the n'th token starting from current without advancing
// p.peekN(0) is equal to p.peek()
func (w *tokenWalker) peekN(n int) *token.Token {
	if w.cur+n >= len(w.tokens) || w.cur+n < 0 {
		return &w.tokens[len(w.tokens)-1] // EOF
	}
	return &w.tokens[w.cur+n]
}

// returns the token before peek()
func (w *tokenWalker) previous() *token.Token {
	if w.cur < 1 {
		return &token.Token{Type: token.ILLEGAL}
	}
	return &w.tokens[w.cur-1]
}

// opposite of advance
func (w *tokenWalker) decrease() {
	if w.cur > 0 {
		w.cur--
	}
}

// retrives the last comment which comes before pos
// if their are no comments before pos nil is returned
func (w *tokenWalker) commentBeforePos(pos token.Position) (result *token.Token) {
	if len(w.comments) == 0 {
		return nil
	}

	for i := range w.comments {
		// the scanner sets any tokens .Range.End.Column to 1 after the last char within the literal
		end := token.Position{Line: w.comments[i].Range.End.Line, Column: w.comments[i].Range.End.Column - 1}
		if end.IsBefore(pos) {
			result = &w.comments[i]
		} else {
			return result
		}
	}
	return result
}

// retrives the first comment which comes after pos
// if their are no comments after pos nil is returned
func (w *tokenWalker) commentAfterPos(pos token.Position) (result *token.Token) {
	if len(w.comments) == 0 {
		return nil
	}

	for i := range w.comments {
		if w.comments[i].Range.End.IsBehind(pos) {
			return &w.comments[i]
		}
	}
	return result
}

// retreives a leading or trailing comment of p.previous()
// prefers leading comments
// may return nil
func (p *tokenWalker) getLeadingOrTrailingComment() (result *token.Token) {
	tok := p.previous()
	comment := p.commentBeforePos(tok.Range.Start)
	// the comment must be between the identifier and the last token of the type
	if comment != nil && !comment.Range.Start.IsBehind(p.peekN(-2).Range.End) {
		comment = nil
	}
	// a trailing comment must be the next token after the identifier
	if trailingComment := p.commentAfterPos(tok.Range.End); comment == nil && trailingComment != nil &&
		trailingComment.Range.End.IsBefore(p.peek().Range.Start) {
		comment = trailingComment
	}

	return comment
}
