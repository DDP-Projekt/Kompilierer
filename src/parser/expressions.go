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
	expr := p.boolOR()
	for p.matchN(token.COMMA, token.FALLS) {
		tok := p.previous()
		cond := p.ifExpression()
		p.consume(token.COMMA, token.ANSONSTEN)
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

func (p *parser) boolOR() ast.Expression {
	expr := p.boolAND()
	for p.match(token.ODER) {
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
	for p.match(token.UND) {
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
	for p.matchN(token.LOGISCH, token.ODER) {
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
	for p.matchN(token.LOGISCH, token.KONTRA) {
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
	for p.matchN(token.LOGISCH, token.UND) {
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
	for p.match(token.GLEICH, token.UNGLEICH) {
		tok := p.previous()
		rhs := p.comparison()
		operator := ast.BIN_EQUAL
		if tok.Type == token.UNGLEICH {
			operator = ast.BIN_UNEQUAL
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
		p.consume(token.IST)
	}
	return expr
}

func (p *parser) comparison() ast.Expression {
	expr := p.bitShift()
	for p.match(token.GRÖßER, token.KLEINER, token.ZWISCHEN) {
		tok := p.previous()
		if tok.Type == token.ZWISCHEN {
			mid := p.bitShift()
			p.consume(token.UND)
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
			p.consume(token.ALS)
			if p.match(token.COMMA) {
				p.consume(token.ODER)
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
		p.consume(token.IST)
	}
	return expr
}

func (p *parser) bitShift() ast.Expression {
	expr := p.term()
	for p.match(token.UM) {
		rhs := p.term()
		p.consume(token.BIT, token.NACH)
		if !p.match(token.LINKS, token.RECHTS) {
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
		p.consume(token.VERSCHOBEN)
	}
	return expr
}

func (p *parser) term() ast.Expression {
	expr := p.factor()
	for p.match(token.PLUS, token.MINUS, token.VERKETTET) {
		tok := p.previous()
		operator := ast.BIN_PLUS
		if tok.Type == token.VERKETTET { // string concatenation
			p.consume(token.MIT)
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
	for p.match(token.MAL, token.DURCH, token.MODULO) {
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
	if p.match(token.NICHT, token.BETRAG, token.GRÖßE, token.LÄNGE, token.STANDARDWERT, token.LOGISCH, token.DIE, token.DER, token.DEM) {
		start := p.previous()

		switch start.Type {
		case token.DIE:
			if !p.match(token.GRÖßE, token.LÄNGE) { // nominativ
				p.decrease() // DIE does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.DER:
			if !p.match(token.GRÖßE, token.LÄNGE, token.BETRAG, token.STANDARDWERT) { // Betrag: nominativ, Größe/Länge: dativ
				p.decrease() // DER does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.DEM:
			if !p.match(token.BETRAG, token.STANDARDWERT) { // dativ
				p.decrease() // DEM does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		case token.LOGISCH:
			if !p.match(token.NICHT) {
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
			p.consume(token.VON)
		case token.GRÖßE, token.STANDARDWERT:
			p.consume(token.VON)
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
			article := p.previous()
			_type := p.parseType()
			operator := ast.TYPE_SIZE
			if tok.Type == token.STANDARDWERT {
				operator = ast.TYPE_DEFAULT
			}

			// report grammar errors
			if _type != nil {
				switch _type.Gender() {
				case ddptypes.FEMININ:
					if article.Type != token.EINER {
						p.err(ddperror.SYN_GENDER_MISMATCH, article.Range, ddperror.MsgGotExpected(article.Literal, "einer"))
					}
				default:
					if article.Type != token.EINEM {
						p.err(ddperror.SYN_GENDER_MISMATCH, article.Range, ddperror.MsgGotExpected(article.Literal, "einem"))
					}
				}
			}

			return &ast.TypeOpExpr{
				Range:    token.NewRange(start, p.previous()),
				Tok:      *start,
				Operator: operator,
				Rhs:      _type,
			}
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
	for p.match(token.NEGATE) {
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
	if lhs == nil && p.match(token.DIE, token.DER) {
		if p.match(token.LOGARITHMUS) {
			tok := p.previous()
			p.consume(token.VON)
			numerus := p.expression()
			p.consume(token.ZUR, token.BASIS)
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
			p.consume(token.DOT, token.WURZEL)
			tok := p.previous()
			p.consume(token.VON)
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

	for p.match(token.HOCH) {
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
	for p.match(token.IM, token.BIS, token.AB) {
		switch p.previous().Type {
		// im Bereich von ... bis ...
		case token.IM:
			p.consume(token.BEREICH, token.VON)
			von := p.previous()
			mid := p.expression()
			p.consume(token.BIS)
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
			if !p.match(token.ZUM) {
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
			p.consume(token.DOT, token.ELEMENT)
			// t ab dem n. Element
		case token.AB:
			p.consume(token.DEM)
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
			p.consume(token.DOT, token.ELEMENT)
		}
	}
	return lhs
}

func (p *parser) indexing(lhs ast.Expression) ast.Expression {
	lhs = p.field_access(lhs)
	for p.match(token.AN) {
		p.consume(token.DER, token.STELLE)
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
	for p.match(token.VON) {
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
	for p.match(token.ALS) {
		Type := p.parseType()
		lhs = &ast.CastExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   token.NewEndPos(p.previous()),
			},
			Type: Type,
			Lhs:  lhs,
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
		if begin.Type == token.EINER && p.match(token.LEEREN) {
			typ := p.parseListType()
			lhs = &ast.ListLit{
				Tok:    *begin,
				Range:  token.NewRange(begin, p.previous()),
				Type:   typ,
				Values: nil,
			}
		} else if p.match(token.LEERE) {
			typ := p.parseListType()
			lhs = &ast.ListLit{
				Tok:    *begin,
				Range:  token.NewRange(begin, p.previous()),
				Type:   typ,
				Values: nil,
			}
		} else {
			p.consume(token.LISTE, token.COMMA, token.DIE, token.AUS)
			values := append(make([]ast.Expression, 0, 2), p.expression())
			for p.match(token.COMMA) {
				values = append(values, p.expression())
			}
			p.consume(token.BESTEHT)
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
	p.consume(token.RPAREN)

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
			p.consume(token.IDENTIFIER)
		}
		ident := &ast.Ident{
			Literal: *p.previous(),
		}
		var ass ast.Assigneable = ident

		for p.match(token.VON) {
			if p.match(token.IDENTIFIER) {
				rhs := assigneable_impl(true)
				ass = &ast.FieldAccess{
					Rhs:   rhs,
					Field: ident,
				}
			} else {
				p.consume(token.LPAREN)
				rhs := assigneable_impl(false)
				ass = &ast.FieldAccess{
					Rhs:   rhs,
					Field: ident,
				}
			}
		}

		if !isInFieldAcess {
			for p.match(token.AN) {
				p.consume(token.DER, token.STELLE)
				index := p.unary() // TODO: check if this can stay p.expression or if p.unary is better
				ass = &ast.Indexing{
					Lhs:   ass,
					Index: index,
				}
				if !p.match(token.COMMA) {
					break
				}
			}
		}

		if isParenthesized {
			p.consume(token.RPAREN)
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
