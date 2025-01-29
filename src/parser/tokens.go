/*
This file contains functions to match and consume tokens
*/
package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// if the current tokenType is contained in types, advance
// returns wether we advanced or not
func (p *parser) matchAny(types ...token.TokenType) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

// if the given sequence of tokens is matched, advance
// returns wether we advance or not
func (p *parser) matchSeq(types ...token.TokenType) bool {
	for i, t := range types {
		if p.peekN(i).Type != t {
			return false
		}
	}

	for range types {
		p.advance()
	}

	return true
}

// consumeSeq a series of tokens
func (p *parser) consumeSeq(t ...token.TokenType) bool {
	for _, v := range t {
		if !p.check(v) {
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, t))
			return false
		}

		p.advance()
	}
	return true
}

// same as consume but tolerates multiple tokenTypes
func (p *parser) consumeAny(tokenTypes ...token.TokenType) bool {
	for _, v := range tokenTypes {
		if p.check(v) {
			p.advance()
			return true
		}
	}

	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, toInterfaceSlice[token.TokenType, any](tokenTypes)...))
	return false
}

// check if the current token is of type t without advancing
func (p *parser) check(t token.TokenType) bool {
	if p.atEnd() {
		return false
	}
	return p.peek().Type == t
}

// check if the current token is EOF
func (p *parser) atEnd() bool {
	return p.peek().Type == token.EOF
}

// return the current token and advance p.cur
func (p *parser) advance() *token.Token {
	if !p.atEnd() {
		p.cur++
		return p.previous()
	}
	return p.peek() // return EOF
}

// returns the current token without advancing
func (p *parser) peek() *token.Token {
	return &p.tokens[p.cur]
}

// returns the n'th token starting from current without advancing
// p.peekN(0) is equal to p.peek()
func (p *parser) peekN(n int) *token.Token {
	if p.cur+n >= len(p.tokens) || p.cur+n < 0 {
		return &p.tokens[len(p.tokens)-1] // EOF
	}
	return &p.tokens[p.cur+n]
}

// returns the token before peek()
func (p *parser) previous() *token.Token {
	if p.cur < 1 {
		return &token.Token{Type: token.ILLEGAL}
	}
	return &p.tokens[p.cur-1]
}

// opposite of advance
func (p *parser) decrease() {
	if p.cur > 0 {
		p.cur--
	}
}

// retrives the last comment which comes before pos
// if their are no comments before pos nil is returned
func (p *parser) commentBeforePos(pos token.Position) (result *token.Token) {
	if len(p.comments) == 0 {
		return nil
	}

	for i := range p.comments {
		// the scanner sets any tokens .Range.End.Column to 1 after the last char within the literal
		end := token.Position{Line: p.comments[i].Range.End.Line, Column: p.comments[i].Range.End.Column - 1}
		if end.IsBefore(pos) {
			result = &p.comments[i]
		} else {
			return result
		}
	}
	return result
}

// retrives the first comment which comes after pos
// if their are no comments after pos nil is returned
func (p *parser) commentAfterPos(pos token.Position) (result *token.Token) {
	if len(p.comments) == 0 {
		return nil
	}

	for i := range p.comments {
		if p.comments[i].Range.End.IsBehind(pos) {
			return &p.comments[i]
		}
	}
	return result
}

// retreives a leading or trailing comment of p.previous()
// prefers leading comments
// may return nil
func (p *parser) getLeadingOrTrailingComment() (result *token.Token) {
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
