/*
This file defines the functions used to parse expressions (except parser.alias() which can be found in alias.go)

The rules are roughly sorted by precedence in ascending order, meaning functions further down in the file have higher precedence than those higher up.
*/
package parser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// attempts to parse an expression but returns a possible error
func (p *parser) expressionOrErr() (ast.Expression, *ddperror.Error) {
	errHndl := p.errorHandler

	var err *ddperror.Error
	p.errorHandler = func(e ddperror.Error) {
		err = &e
	}
	expr := p.expression()
	p.errorHandler = errHndl
	return expr, err
}

// entry for expression parsing
func (p *parser) expression() ast.Expression {
	return p.ifExpression()
}

// <a> wenn <b>, sonst <c>
func (p *parser) ifExpression() ast.Expression {
	expr := p.boolXOR()
	for p.matchSeq(token.COMMA, token.FALLS) {
		tok := p.previous()
		cond := p.ifExpression()
		p.consumeSeq(token.COMMA, token.ANSONSTEN)
		other := p.ifExpression()
		expr = &ast.TernaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   other.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Mid:      cond,
			Rhs:      other,
			Operator: ast.TER_FALLS,
		}
	}
	return expr
}

// entweder <a> oder <b> ist
func (p *parser) boolXOR() ast.Expression {
	for p.matchAny(token.ENTWEDER) {
		tok := p.previous()
		lhs := p.boolOR()
		p.consumeSeq(token.COMMA, token.ODER)
		rhs := p.boolOR()
		return &ast.BinaryExpr{
			Range: token.Range{
				Start: tok.Range.Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      lhs,
			Operator: ast.BIN_XOR,
			Rhs:      rhs,
		}
	}
	return p.boolOR()
}

func (p *parser) boolOR() ast.Expression {
	expr := p.boolAND()
	for p.matchAny(token.ODER) {
		tok := p.previous()
		rhs := p.boolAND()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: ast.BIN_OR,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) boolAND() ast.Expression {
	expr := p.bitwiseOR()
	for p.matchAny(token.UND) {
		tok := p.previous()
		rhs := p.bitwiseOR()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: ast.BIN_AND,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) bitwiseOR() ast.Expression {
	expr := p.bitwiseXOR()
	for p.matchSeq(token.LOGISCH, token.ODER) {
		tok := p.previous()
		rhs := p.bitwiseXOR()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: ast.BIN_LOGIC_OR,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) bitwiseXOR() ast.Expression {
	expr := p.bitwiseAND()
	for p.matchSeq(token.LOGISCH, token.KONTRA) {
		tok := p.previous()
		rhs := p.bitwiseAND()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: ast.BIN_LOGIC_XOR,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) bitwiseAND() ast.Expression {
	expr := p.equality()
	for p.matchSeq(token.LOGISCH, token.UND) {
		tok := p.previous()
		rhs := p.equality()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: ast.BIN_LOGIC_AND,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) equality() ast.Expression {
	expr := p.comparison()
	for p.matchAny(token.GLEICH, token.UNGLEICH, token.EIN, token.EINE, token.KEIN, token.KEINE) {
		tok := p.previous()

		bin_operator := ast.BIN_EQUAL
		switch tok.Type {
		case token.UNGLEICH:
			bin_operator = ast.BIN_UNEQUAL
			fallthrough
		case token.GLEICH:
			rhs := p.comparison()
			expr = &ast.BinaryExpr{
				Range: token.Range{
					Start: expr.GetRange().Start,
					End:   rhs.GetRange().End,
				},
				Tok:      *tok,
				Lhs:      expr,
				Operator: bin_operator,
				Rhs:      rhs,
			}
		case token.EIN, token.EINE, token.KEIN, token.KEINE:
			checkType := p.parseType(false)
			expr = &ast.TypeCheck{
				Range: token.Range{
					Start: expr.GetRange().Start,
					End:   p.previous().Range.End,
				},
				Tok:       *tok,
				CheckType: checkType,
				Lhs:       expr,
			}
			if tok.Type == token.KEIN || tok.Type == token.KEINE {
				expr = &ast.UnaryExpr{
					Range:    expr.GetRange(),
					Tok:      expr.Token(),
					Operator: ast.UN_NOT,
					Rhs:      expr,
				}
			}

			// gender check
			if checkType != nil {
				if (tok.Type == token.EIN || tok.Type == token.KEIN) && checkType.Gender() == ddptypes.FEMININ {
					p.err(ddperror.SYN_GENDER_MISMATCH, token.NewRange(tok, tok), "Meintest du 'ein'?")
				} else if (tok.Type == token.EINE || tok.Type == token.KEINE) && checkType.Gender() != ddptypes.FEMININ {
					p.err(ddperror.SYN_GENDER_MISMATCH, token.NewRange(tok, tok), "Meintest du 'eine'?")
				}
			}
		}

		if p.previous().Type != token.IST {
			p.consumeSeq(token.IST)
		} else {
			p.matchAny(token.IST)
		}
	}
	return expr
}

func (p *parser) comparison() ast.Expression {
	expr := p.bitShift()
	for p.matchAny(token.GRÖßER, token.KLEINER, token.ZWISCHEN) {
		tok := p.previous()
		if tok.Type == token.ZWISCHEN {
			mid := p.bitShift()
			p.consumeSeq(token.UND)
			rhs := p.bitShift()

			// expr > mid && expr < rhs
			expr = &ast.TernaryExpr{
				Range: token.Range{
					Start: expr.GetRange().Start,
					End:   rhs.GetRange().End,
				},
				Lhs:      expr,
				Mid:      mid,
				Rhs:      rhs,
				Operator: ast.TER_BETWEEN,
			}
		} else {
			operator := ast.BIN_GREATER
			if tok.Type == token.KLEINER {
				operator = ast.BIN_LESS
			}
			p.consumeSeq(token.ALS)
			if p.matchAny(token.COMMA) {
				p.consumeSeq(token.ODER)
				if tok.Type == token.GRÖßER {
					operator = ast.BIN_GREATER_EQ
				} else {
					operator = ast.BIN_LESS_EQ
				}
			}

			rhs := p.bitShift()
			expr = &ast.BinaryExpr{
				Range: token.Range{
					Start: expr.GetRange().Start,
					End:   rhs.GetRange().End,
				},
				Tok:      *tok,
				Lhs:      expr,
				Operator: operator,
				Rhs:      rhs,
			}
		}
		if p.previous().Type != token.IST {
			p.consumeSeq(token.IST)
		} else {
			p.matchAny(token.IST)
		}
	}
	return expr
}

func (p *parser) bitShift() ast.Expression {
	expr := p.term()
	for p.matchAny(token.UM) {
		rhs := p.term()
		p.consumeSeq(token.BIT, token.NACH)
		if !p.matchAny(token.LINKS, token.RECHTS) {
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "Links", "Rechts"))
			return &ast.BadExpr{
				Err: p.lastError,
				Tok: expr.Token(),
			}
		}
		tok := p.previous()
		operator := ast.BIN_LEFT_SHIFT
		if tok.Type == token.RECHTS {
			operator = ast.BIN_RIGHT_SHIFT
		}
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
		p.consumeSeq(token.VERSCHOBEN)
	}
	return expr
}

func (p *parser) term() ast.Expression {
	expr := p.factor()
	for p.matchAny(token.PLUS, token.MINUS, token.VERKETTET) {
		tok := p.previous()
		operator := ast.BIN_PLUS
		if tok.Type == token.VERKETTET { // string concatenation
			p.consumeSeq(token.MIT)
			operator = ast.BIN_CONCAT
		} else if tok.Type == token.MINUS {
			operator = ast.BIN_MINUS
		}
		rhs := p.factor()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) factor() ast.Expression {
	expr := p.unary()
	for p.matchAny(token.MAL, token.DURCH, token.MODULO) {
		tok := p.previous()
		operator := ast.BIN_MULT
		if tok.Type == token.DURCH {
			operator = ast.BIN_DIV
		} else if tok.Type == token.MODULO {
			operator = ast.BIN_MOD
		}
		rhs := p.unary()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) unary() ast.Expression {
	if expr := p.alias(); expr != nil { // first check for a function call to enable operator overloading
		return p.power(expr)
	}
	// match the correct unary operator
	if p.matchAny(token.NICHT, token.BETRAG, token.GRÖßE, token.LÄNGE, token.STANDARDWERT, token.LOGISCH, token.DIE, token.DER, token.DEM, token.DEN) {
		start := p.previous()

		switch start.Type {
		case token.DIE:
			if !p.matchAny(token.GRÖßE, token.LÄNGE) { // nominativ
				p.decrease() // DIE does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.DER:
			if !p.matchAny(token.GRÖßE, token.LÄNGE, token.BETRAG, token.STANDARDWERT) { // Betrag: nominativ, Größe/Länge: dativ
				p.decrease() // DER does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.DEN:
			if !p.matchAny(token.BETRAG, token.STANDARDWERT) { // dativ
				p.decrease() // DEN does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.DEM:
			if !p.matchAny(token.BETRAG, token.STANDARDWERT) { // dativ
				p.decrease() // DEM does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.LOGISCH:
			if !p.matchAny(token.NICHT) {
				p.decrease() // LOGISCH does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.BETRAG, token.LÄNGE, token.GRÖßE, token.STANDARDWERT:
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, start.Range, fmt.Sprintf("Vor '%s' fehlt der Artikel", start))
		}

		tok := p.previous()
		operator := ast.UN_ABS
		switch tok.Type {
		case token.BETRAG, token.LÄNGE:
			p.consumeSeq(token.VON)
		case token.GRÖßE, token.STANDARDWERT:
			p.consumeSeq(token.VON)
			p.consumeAny(token.EINEM, token.EINER)
		case token.NICHT:
			if p.peekN(-2).Type == token.LOGISCH {
				operator = ast.UN_LOGIC_NOT
			}
		}
		switch tok.Type {
		case token.NICHT:
			if operator != ast.UN_LOGIC_NOT {
				operator = ast.UN_NOT
			}
		case token.GRÖßE, token.STANDARDWERT:
			article := p.previous().Type
			_type := p.parseType(false)
			operator := ast.TYPE_SIZE
			if tok.Type == token.STANDARDWERT {
				operator = ast.TYPE_DEFAULT
			}

			// report grammar errors
			if _type != nil {
				if !ddptypes.MatchesGender(_type, genderFromArticle2Dativ(article)...) {
					p.err(ddperror.SYN_GENDER_MISMATCH, tok.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", articleFromGender2Dativ(_type.Gender())))
				}
			}

			return p.power(&ast.TypeOpExpr{
				Range:    token.NewRange(start, p.previous()),
				Tok:      *start,
				Operator: operator,
				Rhs:      _type,
			})
		case token.LÄNGE:
			operator = ast.UN_LEN
		}
		rhs := p.unary()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(start),
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return p.negate()
}

func (p *parser) negate() ast.Expression {
	for p.matchAny(token.NEGATE) {
		tok := p.previous()
		rhs := p.negate()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(tok),
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Operator: ast.UN_NEGATE,
			Rhs:      rhs,
		}
	}
	return p.power(nil)
}

// when called from unary() lhs might be a funcCall
// TODO: check precedence
func (p *parser) power(lhs ast.Expression) ast.Expression {
	// TODO: grammar
	if lhs == nil && p.matchAny(token.DIE, token.DER) {
		if p.matchAny(token.LOGARITHMUS) {
			tok := p.previous()
			p.consumeSeq(token.VON)
			numerus := p.expression()
			p.consumeSeq(token.ZUR, token.BASIS)
			rhs := p.unary()

			lhs = &ast.BinaryExpr{
				Range: token.Range{
					Start: numerus.GetRange().Start,
					End:   rhs.GetRange().End,
				},
				Tok:      *tok,
				Lhs:      numerus,
				Operator: ast.BIN_LOG,
				Rhs:      rhs,
			}
		} else {
			lhs = p.unary()
			p.consumeSeq(token.DOT, token.WURZEL)
			tok := p.previous()
			p.consumeSeq(token.VON)
			// root is implemented as pow(degree, 1/radicant)
			expr := p.unary()

			lhs = &ast.BinaryExpr{
				Range: token.Range{
					Start: expr.GetRange().Start,
					End:   lhs.GetRange().End,
				},
				Tok:      *tok,
				Lhs:      expr,
				Operator: ast.BIN_POW,
				Rhs: &ast.BinaryExpr{
					Lhs: &ast.IntLit{
						Literal: lhs.Token(),
						Value:   1,
					},
					Tok:      *tok,
					Operator: ast.BIN_DIV,
					Rhs:      lhs,
				},
			}
		}
	}

	lhs = p.slicing(lhs) // make sure postfix operators after a function call are parsed

	for p.matchAny(token.HOCH) {
		tok := p.previous()
		rhs := p.unary()
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      lhs,
			Operator: ast.BIN_POW,
			Rhs:      rhs,
		}
	}
	return lhs
}

func (p *parser) slicing(lhs ast.Expression) ast.Expression {
	lhs = p.indexing(lhs)
	for p.matchAny(token.IM, token.BIS, token.AB) {
		switch p.previous().Type {
		// im Bereich von ... bis ...
		case token.IM:
			p.consumeSeq(token.BEREICH, token.VON)
			von := p.previous()
			mid := p.expression()
			p.consumeSeq(token.BIS)
			rhs := p.indexing(nil)
			lhs = &ast.TernaryExpr{
				Range: token.Range{
					Start: lhs.GetRange().Start,
					End:   rhs.GetRange().End,
				},
				Tok:      *von,
				Lhs:      lhs,
				Mid:      mid,
				Rhs:      rhs,
				Operator: ast.TER_SLICE,
			}
			// t bis zum n. Element
		case token.BIS:
			if !p.matchAny(token.ZUM) {
				p.decrease()
				return lhs
			}
			rhs := p.expression()
			lhs = &ast.BinaryExpr{
				Range: token.Range{
					Start: lhs.GetRange().Start,
					End:   token.NewEndPos(p.previous()),
				},
				Tok:      rhs.Token(),
				Lhs:      lhs,
				Rhs:      rhs,
				Operator: ast.BIN_SLICE_TO,
			}
			p.consumeSeq(token.DOT, token.ELEMENT)
			// t ab dem n. Element
		case token.AB:
			p.consumeSeq(token.DEM)
			rhs := p.expression()
			lhs = &ast.BinaryExpr{
				Range: token.Range{
					Start: lhs.GetRange().Start,
					End:   token.NewEndPos(p.previous()),
				},
				Tok:      rhs.Token(),
				Lhs:      lhs,
				Rhs:      rhs,
				Operator: ast.BIN_SLICE_FROM,
			}
			p.consumeSeq(token.DOT, token.ELEMENT)
		}
	}
	return lhs
}

func (p *parser) indexing(lhs ast.Expression) ast.Expression {
	lhs = p.field_access(lhs)
	for p.matchAny(token.AN) {
		p.consumeSeq(token.DER, token.STELLE)
		tok := p.previous()
		rhs := p.field_access(nil)
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *tok,
			Lhs:      lhs,
			Operator: ast.BIN_INDEX,
			Rhs:      rhs,
		}
	}
	return lhs
}

// x von y von z = x von (y von z)

func (p *parser) field_access(lhs ast.Expression) ast.Expression {
	lhs = p.type_cast(lhs)
	for p.matchAny(token.VON) {
		von := p.previous()
		rhs := p.field_access(nil) // recursive call to enable x von y von z (right-associative)
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      *von,
			Lhs:      lhs,
			Operator: ast.BIN_FIELD_ACCESS,
			Rhs:      rhs,
		}
	}
	return lhs
}

func (p *parser) type_cast(lhs ast.Expression) ast.Expression {
	lhs = p.primary(lhs)
	for p.matchAny(token.ALS) {
		Type := p.parseType(false)
		lhs = &ast.CastExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   token.NewEndPos(p.previous()),
			},
			TargetType: Type,
			Lhs:        lhs,
		}
	}

	return lhs
}

// when called from power() lhs might be a funcCall
func (p *parser) primary(lhs ast.Expression) ast.Expression {
	if lhs != nil {
		return lhs
	}

	// funccall has the highest precedence (aliases + operator overloading)
	lhs = p.alias()
	if lhs != nil {
		return lhs
	}

	switch tok := p.advance(); tok.Type {
	case token.FALSE:
		lhs = &ast.BoolLit{Literal: *p.previous(), Value: false}
	case token.TRUE:
		lhs = &ast.BoolLit{Literal: *p.previous(), Value: true}
	case token.INT:
		lhs = p.parseIntLit()
	case token.FLOAT:
		lit := p.previous()
		if val, err := strconv.ParseFloat(strings.Replace(lit.Literal, ",", ".", 1), 64); err == nil {
			lhs = &ast.FloatLit{Literal: *lit, Value: val}
		} else {
			p.err(ddperror.SYN_MALFORMED_LITERAL, lit.Range, fmt.Sprintf("Das Kommazahlen Literal '%s' kann nicht gelesen werden", lit.Literal))
			lhs = &ast.FloatLit{Literal: *lit, Value: 0}
		}
	case token.CHAR:
		lit := p.previous()
		lhs = &ast.CharLit{Literal: *lit, Value: p.parseChar(lit.Literal)}
	case token.STRING:
		lit := p.previous()
		lhs = &ast.StringLit{Literal: *lit, Value: p.parseString(lit.Literal)}
	case token.LPAREN:
		lhs = p.grouping()
	case token.IDENTIFIER:
		lhs = &ast.Ident{
			Literal: *p.previous(),
		}
	// TODO: grammar
	case token.EINE, token.EINER: // list literals
		begin := p.previous()
		if (begin.Type == token.EINER && p.matchAny(token.LEEREN)) || p.matchAny(token.LEERE) {
			typ := p.parseType(false)
			listType, isList := ddptypes.CastList(typ)
			if !isList {
				p.err(ddperror.SYN_EXPECTED_TYPENAME, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Listen-Typname"))
			}

			lhs = &ast.ListLit{
				Tok:    *begin,
				Range:  token.NewRange(begin, p.previous()),
				Type:   listType,
				Values: nil,
			}
		} else {
			p.consumeSeq(token.LISTE, token.COMMA, token.DIE, token.AUS)
			values := append(make([]ast.Expression, 0, 2), p.expression())
			for p.matchAny(token.COMMA) {
				values = append(values, p.expression())
			}
			p.consumeSeq(token.BESTEHT)
			lhs = &ast.ListLit{
				Tok:    *begin,
				Range:  token.NewRange(begin, p.previous()),
				Values: values,
			}
		}
	default:
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Literal", "ein Name"))
		lhs = &ast.BadExpr{
			Err: p.lastError,
			Tok: *tok,
		}
	}

	return lhs
}

// alias

func (p *parser) grouping() ast.Expression {
	lParen := p.previous()
	innerExpr := p.expression()
	p.consumeSeq(token.RPAREN)

	return &ast.Grouping{
		Range:  token.NewRange(lParen, p.previous()),
		LParen: *lParen,
		Expr:   innerExpr,
	}
}

// either ast.Ident, ast.Indexing or ast.FieldAccess
// p.previous() must be of Type token.IDENTIFIER or token.LPAREN
// TODO: fix precedence with braces
func (p *parser) assigneable() ast.Assigneable {
	var assigneable_impl func(bool) ast.Assigneable
	assigneable_impl = func(isInFieldAcess bool) ast.Assigneable {
		isParenthesized := p.previous().Type == token.LPAREN
		if isParenthesized {
			p.consumeSeq(token.IDENTIFIER)
		}
		ident := &ast.Ident{
			Literal: *p.previous(),
		}
		var ass ast.Assigneable = ident

		for p.matchAny(token.VON) {
			if p.matchAny(token.IDENTIFIER) {
				rhs := assigneable_impl(true)
				ass = &ast.FieldAccess{
					Rhs:   rhs,
					Field: ident,
				}
			} else {
				p.consumeSeq(token.LPAREN)
				rhs := assigneable_impl(false)
				ass = &ast.FieldAccess{
					Rhs:   rhs,
					Field: ident,
				}
			}
		}

		if !isInFieldAcess {
			for p.matchAny(token.AN) {
				p.consumeSeq(token.DER, token.STELLE)
				index := p.unary() // TODO: check if this can stay p.expression or if p.unary is better
				ass = &ast.Indexing{
					Lhs:   ass,
					Index: index,
				}
				if !p.matchAny(token.COMMA) {
					break
				}
			}
		}

		if isParenthesized {
			p.consumeSeq(token.RPAREN)
		}
		return ass
	}
	return assigneable_impl(false)
}

/*** Helper functions ***/

// helper to parse ddp chars with escape sequences
func (p *parser) parseChar(s string) (r rune) {
	lit := strings.TrimPrefix(strings.TrimSuffix(s, "'"), "'") // remove the ''
	switch utf8.RuneCountInString(lit) {
	case 1: // a single character can just be returned
		r, _ = utf8.DecodeRuneInString(lit)
		return r
	case 2: // two characters means \ something, the scanner would have errored otherwise
		r, _ := utf8.DecodeLastRuneInString(lit)
		switch r {
		case 'a':
			r = '\a'
		case 'b':
			r = '\b'
		case 'n':
			r = '\n'
		case 'r':
			r = '\r'
		case 't':
			r = '\t'
		case '\'':
			r = '\''
		case '\\':
		default:
			p.err(ddperror.SYN_MALFORMED_LITERAL, p.previous().Range, fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Buchstaben Literal", string(r)))
		}
		return r
	}
	return -1
}

// helper to parse ddp strings with escape sequences
func (p *parser) parseString(s string) string {
	str := strings.TrimPrefix(strings.TrimSuffix(s, "\""), "\"") // remove the ""

	for i, w := 0, 0; i < len(str); i += w {
		var r rune
		r, w = utf8.DecodeRuneInString(str[i:])
		if r == '\\' {
			seq, w2 := utf8.DecodeRuneInString(str[i+w:])
			switch seq {
			case 'a':
				seq = '\a'
			case 'b':
				seq = '\b'
			case 'n':
				seq = '\n'
			case 'r':
				seq = '\r'
			case 't':
				seq = '\t'
			case '"':
			case '\\':
			default:
				p.err(ddperror.SYN_MALFORMED_LITERAL, p.previous().Range, fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Text Literal", string(seq)))
				continue
			}

			str = str[:i] + string(seq) + str[i+w+w2:]
		}
	}

	return str
}

func (p *parser) parseIntLit() *ast.IntLit {
	lit := p.previous()
	if val, err := strconv.ParseInt(lit.Literal, 10, 64); err == nil {
		return &ast.IntLit{Literal: *lit, Value: val}
	} else {
		p.err(ddperror.SYN_MALFORMED_LITERAL, lit.Range, fmt.Sprintf("Das Zahlen Literal '%s' kann nicht gelesen werden", lit.Literal))
		return &ast.IntLit{Literal: *lit, Value: 0}
	}
}
