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
// if generic is true, unknown identifiers are treated as generic types
func (p *parser) parseType(generic bool) ddptypes.Type {
	if !p.matchAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER,
		token.VARIABLE, token.VARIABLEN, token.LPAREN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil
	}

	types := make([]ddptypes.Type, 0, 4)

	for ok := true; ok; ok = p.matchAny(token.NEGATE) {
		if p.previous().Type == token.NEGATE {
			p.consumeAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
				token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER,
				token.VARIABLE, token.VARIABLEN, token.LPAREN)
		}

		var typ ddptypes.Type
		switch p.previous().Type {
		case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.VARIABLE:
			typ = p.tokenTypeToType(p.previous().Type)
		case token.WAHRHEITSWERT, token.TEXT:
			if !p.matchAny(token.LISTE) {
				typ = p.tokenTypeToType(p.previous().Type)
				break
			}
			typ = ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-2).Type)}
		case token.ZAHLEN:
			p.consumeSeq(token.LISTE)
			typ = ddptypes.ListType{Underlying: ddptypes.ZAHL}
		case token.KOMMAZAHLEN:
			p.consumeSeq(token.LISTE)
			typ = ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}
		case token.BUCHSTABEN:
			if p.peekN(-2).Type == token.EINEN || p.peekN(-2).Type == token.JEDEN { // edge case in function return types and for-range loops
				typ = ddptypes.BUCHSTABE
				break
			}
			p.consumeSeq(token.LISTE)
			typ = ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}
		case token.VARIABLEN:
			p.consumeSeq(token.LISTE)
			typ = ddptypes.ListType{Underlying: ddptypes.VARIABLE}
		case token.LPAREN:
			typ = p.parseType(generic)
			p.consumeAny(token.RPAREN)
		case token.IDENTIFIER:
			if Type, exists := p.scope().LookupType(p.previous().Literal); exists || generic {
				if generic && !exists {
					Type = ddptypes.GenericType{Name: p.previous().Literal}
				}

				if p.matchAny(token.LISTE) {
					typ = ddptypes.ListType{Underlying: Type}
				} else {
					typ = Type
				}
				break
			}
			fallthrough
		default:
			p.err(ddperror.SYN_EXPECTED_TYPENAME, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Typname"))
		}

		if typ != nil {
			types = append(types, typ)
		}
	}

	if len(types) == 0 {
		return nil
	} else if len(types) == 1 {
		return types[0]
	}

	mainType := types[len(types)-1]
	if genericStruct, isGeneric := ddptypes.CastGenericStructType(mainType); isGeneric {
		return ddptypes.GetInstantiatedStructType(genericStruct, types[:len(types)-1])
	}

	p.err(ddperror.SEM_CANNOT_INSTANTIATE_NON_GENERIC_TYPE, p.previous().Range, "Ein nicht-generischer Typ kann keine Typparameter haben")

	return mainType
}

// parses tokens into a DDPType and returns wether the type is a reference type
// expects the next token to be the start of the type
// returns nil and errors if no typename was found
func (p *parser) parseReferenceType(generic bool) (ddptypes.Type, bool) {
	if !p.matchAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER,
		token.VARIABLE, token.VARIABLEN, token.LPAREN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil, false // void indicates error
	}

	types := make([]ddptypes.Type, 0, 4)
	isRef := false

	for ok := true; ok; ok = p.matchAny(token.NEGATE) {
		if isRef {
			p.decrease()
			p.err(ddperror.TYP_REFERENCE_TYPE_PARAM, p.previous().Range, "Ein Typparameter darf keine Referenz sein")
			break
		}

		if p.previous().Type == token.NEGATE {
			p.consumeAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
				token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER,
				token.VARIABLE, token.VARIABLEN, token.LPAREN)
		}

		var typ ddptypes.Type
		switch p.previous().Type {
		case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.VARIABLE:
			typ, isRef = p.tokenTypeToType(p.previous().Type), false
		case token.WAHRHEITSWERT, token.TEXT:
			if p.matchAny(token.LISTE) {
				typ, isRef = ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-2).Type)}, false
			} else if p.matchAny(token.LISTEN) {
				if !p.consumeSeq(token.REFERENZ) {
					// report the error on the REFERENZ token, but still advance
					// because there is a valid token afterwards
					p.advance()
				}
				typ, isRef = ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-3).Type)}, true
			} else if p.matchAny(token.REFERENZ) {
				typ, isRef = p.tokenTypeToType(p.peekN(-2).Type), true
			} else {
				typ, isRef = p.tokenTypeToType(p.previous().Type), false
			}
		case token.ZAHLEN:
			if p.matchAny(token.LISTE) {
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.ZAHL}, false
			} else if p.matchAny(token.LISTEN) {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.ZAHL}, true
			} else {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.ZAHL, true
			}
		case token.KOMMAZAHLEN:
			if p.matchAny(token.LISTE) {
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, false
			} else if p.matchAny(token.LISTEN) {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, true
			} else {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.KOMMAZAHL, true
			}
		case token.BUCHSTABEN:
			if p.matchAny(token.LISTE) {
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, false
			} else if p.matchAny(token.LISTEN) {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, true
			} else {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.BUCHSTABE, true
			}
		case token.VARIABLEN:
			if p.matchAny(token.LISTE) {
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.VARIABLE}, false
			} else if p.matchAny(token.LISTEN) {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.ListType{Underlying: ddptypes.VARIABLE}, true
			} else {
				p.consumeSeq(token.REFERENZ)
				typ, isRef = ddptypes.VARIABLE, true
			}
		case token.LPAREN:
			typ = p.parseType(generic)
			p.consumeAny(token.RPAREN)
		case token.IDENTIFIER:
			if Type, exists := p.scope().LookupType(p.previous().Literal); exists || generic {
				if generic && !exists {
					Type = ddptypes.GenericType{Name: p.previous().Literal}
				}

				if p.matchAny(token.LISTE) {
					typ, isRef = ddptypes.ListType{Underlying: Type}, false
				} else if p.matchAny(token.LISTEN) {
					if !p.consumeSeq(token.REFERENZ) {
						// report the error on the REFERENZ token, but still advance
						// because there is a valid token afterwards
						p.advance()
					}
					typ, isRef = ddptypes.ListType{Underlying: Type}, true
				} else if p.matchAny(token.REFERENZ) {
					typ, isRef = Type, true
				} else {
					typ, isRef = Type, false
				}
				break
			}
			fallthrough
		default:
			p.err(ddperror.SYN_EXPECTED_TYPENAME, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Typname"))
		}
		if typ != nil {
			types = append(types, typ)
		}
	}

	if len(types) == 0 {
		return nil, false
	} else if len(types) == 1 {
		return types[0], isRef
	}

	mainType := types[len(types)-1]
	if genericStruct, isGeneric := ddptypes.CastGenericStructType(mainType); isGeneric {
		return ddptypes.GetInstantiatedStructType(genericStruct, types[:len(types)-1]), isRef
	}

	p.err(ddperror.SEM_CANNOT_INSTANTIATE_NON_GENERIC_TYPE, p.previous().Range, "Ein Typ, der keine generische Kombination ist kann keine Typparameter haben")

	return mainType, isRef
}

// parses tokens into a DDPType
// unlike parseType it may return void
// the error return is ILLEGAL
func (p *parser) parseReturnType(genericTypes map[string]ddptypes.GenericType) ddptypes.Type {
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
	expectedGender := func(tok token.TokenType) ddptypes.GrammaticalGender {
		switch tok {
		case token.EINEN:
			return ddptypes.MASKULIN
		case token.EINE:
			return ddptypes.FEMININ
		case token.EIN:
			return ddptypes.NEUTRUM
		}
		return ddptypes.INVALID_GENDER // unreachable
	}

	if p.matchAny(token.NICHTS) {
		return ddptypes.VoidType{}
	}
	p.consumeAny(token.EINEN, token.EINE, token.EIN)
	tok := p.previous()

	var typ ddptypes.Type
	if len(genericTypes) > 0 && p.peek().Type == token.IDENTIFIER {
		var ok bool
		if typ, ok = genericTypes[p.peek().Literal]; !ok {
			typ = p.parseType(false)
		} else {
			p.advance()
			if p.matchAny(token.LISTE) {
				typ = ddptypes.ListType{Underlying: typ}
			}
		}
	} else {
		typ = p.parseType(false)
	}

	if typ == nil {
		return typ // prevent the crash from the if below
	}
	if !ddptypes.MatchesGender(typ, expectedGender(tok.Type)) {
		p.err(ddperror.SYN_GENDER_MISMATCH, tok.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", getArticle(typ.Gender())))
	}
	return typ
}
