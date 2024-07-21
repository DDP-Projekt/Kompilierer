/*
This file contains functions to parse DDP Types
*/
package parser

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// converts a TokenType to a Type
func (p *parser) tokenTypeToType(t token.TokenType) ddptypes.Type {
	switch t {
	case token.NICHTS:
		return ddptypes.VoidType{}
	case token.ZAHL:
		return ddptypes.ZAHL
	case token.KOMMAZAHL:
		return ddptypes.KOMMAZAHL
	case token.WAHRHEITSWERT:
		return ddptypes.WAHRHEITSWERT
	case token.BUCHSTABE:
		return ddptypes.BUCHSTABE
	case token.TEXT:
		return ddptypes.TEXT
	case token.VARIABLE:
		return ddptypes.VARIABLE
	}
	p.panic("invalid TokenType (%d)", t)
	return ddptypes.VoidType{} // unreachable
}

// parses tokens into a DDPType
// expects the next token to be the start of the type
// returns nil and errors if no typename was found
func (p *parser) parseType() ddptypes.Type {
	if !p.matchAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER, token.VARIABLE) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.VARIABLE:
		return p.tokenTypeToType(p.previous().Type)
	case token.WAHRHEITSWERT, token.TEXT:
		if !p.matchAny(token.LISTE) {
			return p.tokenTypeToType(p.previous().Type)
		}
		return ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-2).Type)}
	case token.ZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.ListType{Underlying: ddptypes.ZAHL}
	case token.KOMMAZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}
	case token.BUCHSTABEN:
		if p.peekN(-2).Type == token.EINEN || p.peekN(-2).Type == token.JEDEN { // edge case in function return types and for-range loops
			return ddptypes.BUCHSTABE
		}
		p.consume(token.LISTE)
		return ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}
	case token.VARIABLEN:
		p.consume(token.LISTE)
		return ddptypes.ListType{Underlying: ddptypes.VARIABLE}
	case token.IDENTIFIER:
		if Type, exists := p.scope().LookupType(p.previous().Literal); exists {
			if p.matchAny(token.LISTE) {
				return ddptypes.ListType{Underlying: Type}
			}
			return Type
		}
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
	}

	return nil // unreachable
}

// parses tokens into a DDPType which must be a list type
// expects the next token to be the start of the type
// returns VoidList and errors if no typename was found
// returns a ddptypes.ListType
func (p *parser) parseListType() ddptypes.ListType {
	if !p.matchAny(token.WAHRHEITSWERT, token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER, token.VARIABLEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Listen-Typname"))
		return ddptypes.ListType{Underlying: ddptypes.VoidType{}} // void indicates error
	}

	result := ddptypes.ListType{Underlying: ddptypes.VoidType{}} // void indicates error
	switch p.previous().Type {
	case token.WAHRHEITSWERT, token.TEXT:
		result = ddptypes.ListType{Underlying: p.tokenTypeToType(p.previous().Type)}
	case token.ZAHLEN:
		result = ddptypes.ListType{Underlying: ddptypes.ZAHL}
	case token.KOMMAZAHLEN:
		result = ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}
	case token.BUCHSTABEN:
		result = ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}
	case token.VARIABLEN:
		result = ddptypes.ListType{Underlying: ddptypes.VARIABLE}
	case token.IDENTIFIER:
		if Type, exists := p.scope().LookupType(p.previous().Literal); exists {
			result = ddptypes.ListType{Underlying: Type}
		} else {
			p.err(ddperror.SYN_EXPECTED_TYPENAME, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Listen-Typname"))
		}
	}
	p.consume(token.LISTE)

	return result
}

// parses tokens into a DDPType and returns wether the type is a reference type
// expects the next token to be the start of the type
// returns nil and errors if no typename was found
func (p *parser) parseReferenceType() (ddptypes.Type, bool) {
	if !p.matchAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER, token.VARIABLE, token.VARIABLEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil, false // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.VARIABLE:
		return p.tokenTypeToType(p.previous().Type), false
	case token.WAHRHEITSWERT, token.TEXT:
		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-2).Type)}, false
		} else if p.matchAny(token.LISTEN) {
			if !p.consume(token.REFERENZ) {
				// report the error on the REFERENZ token, but still advance
				// because there is a valid token afterwards
				p.advance()
			}
			return ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-3).Type)}, true
		} else if p.matchAny(token.REFERENZ) {
			return p.tokenTypeToType(p.peekN(-2).Type), true
		}
		return p.tokenTypeToType(p.previous().Type), false
	case token.ZAHLEN:
		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.ZAHL}, false
		} else if p.matchAny(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.ZAHL}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.ZAHL, true
	case token.KOMMAZAHLEN:
		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, false
		} else if p.matchAny(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.KOMMAZAHL, true
	case token.BUCHSTABEN:
		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, false
		} else if p.matchAny(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.BUCHSTABE, true
	case token.VARIABLEN:
		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.VARIABLE}, false
		} else if p.matchAny(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.VARIABLE}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.VARIABLE, true
	case token.IDENTIFIER:
		if Type, exists := p.scope().LookupType(p.previous().Literal); exists {
			if p.matchAny(token.LISTE) {
				return ddptypes.ListType{Underlying: Type}, false
			} else if p.matchAny(token.LISTEN) {
				if !p.consume(token.REFERENZ) {
					// report the error on the REFERENZ token, but still advance
					// because there is a valid token afterwards
					p.advance()
				}
				return ddptypes.ListType{Underlying: Type}, true
			} else if p.matchAny(token.REFERENZ) {
				return Type, true
			}

			return Type, false
		}
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peekN(-2).Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Listen-Typname"))
	}

	return nil, false // unreachable
}

// parses tokens into a DDPType
// unlike parseType it may return void
// the error return is ILLEGAL
func (p *parser) parseReturnType() ddptypes.Type {
	getArticle := func(gender ddptypes.GrammaticalGender) token.TokenType {
		switch gender {
		case ddptypes.MASKULIN:
			return token.EINEN
		case ddptypes.FEMININ:
			return token.EINE
		case ddptypes.NEUTRUM:
			return token.EIN
		}
		return token.ILLEGAL // unreachable
	}

	if p.matchAny(token.NICHTS) {
		return ddptypes.VoidType{}
	}
	p.consumeAny(token.EINEN, token.EINE, token.EIN)
	tok := p.previous()
	typ := p.parseType()
	if typ == nil {
		return typ // prevent the crash from the if below
	}
	if article := getArticle(typ.Gender()); article != tok.Type {
		p.err(ddperror.SYN_GENDER_MISMATCH, tok.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", article))
	}
	return typ
}
