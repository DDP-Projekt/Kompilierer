/*
This file defines functions to parse DDP statements.
*/
package parser

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// parse a single statement
func (p *parser) statement() ast.Statement {
	// check for assignement
	if p.matchAny(token.IDENTIFIER) {
		if p.peek().Type == token.IST || p.peek().Type == token.AN {
			return p.assignLiteral() // x ist ... assignements may only have literals, so we use this helper function
		} else {
			p.decrease() // no assignement, so probably an expressionStatement()
		}
	}

	// parse all possible statements
	switch p.peek().Type {
	case token.BINDE:
		p.consumeSeq(token.BINDE)
		return p.importStatement()
	case token.ERHÖHE, token.VERRINGERE,
		token.VERVIELFACHE, token.TEILE,
		token.VERSCHIEBE, token.NEGIERE:
		p.advance()
		return p.compoundAssignement()
	case token.SPEICHERE:
		p.consumeSeq(token.SPEICHERE)
		return p.assignNoLiteral() // Speichere ... in x, where non-literal expressions are allowed
	case token.WENN:
		p.consumeSeq(token.WENN)
		return p.ifStatement()
	case token.SOLANGE:
		p.consumeSeq(token.SOLANGE)
		return p.whileStatement()
	case token.MACHE:
		p.consumeSeq(token.MACHE)
		return p.doWhileStmt()
	case token.WIEDERHOLE:
		p.consumeSeq(token.WIEDERHOLE)
		return p.repeatStmt()
	case token.FÜR:
		p.consumeSeq(token.FÜR)
		return p.forStatement()
	case token.GIB:
		p.consumeSeq(token.GIB)
		return p.returnStatement()
	case token.VERLASSE:
		p.consumeSeq(token.VERLASSE)
		return p.voidReturnOrBreak()
	case token.FAHRE:
		p.consumeSeq(token.FAHRE)
		return p.continueStatement()
	case token.COLON:
		p.consumeSeq(token.COLON)
		return p.blockStatement(nil)
	case token.ELIPSIS:
		p.consumeSeq(token.ELIPSIS)
		return p.todoStmt()
	}

	// no other statement was found, so interpret it as expression statement, whose result will be discarded
	return p.expressionStatement()
}

func (p *parser) importStatement() ast.Statement {
	binde := p.previous()
	var stmt *ast.ImportStmt
	if p.matchAny(token.STRING) {
		stmt = &ast.ImportStmt{
			FileName:        *p.previous(),
			ImportedSymbols: nil,
		}
	} else if p.matchAny(token.IDENTIFIER) {
		importedSymbols := []token.Token{*p.previous()}
		if p.peek().Type != token.AUS {
			if p.matchAny(token.UND) {
				p.consumeSeq(token.IDENTIFIER)
				importedSymbols = append(importedSymbols, *p.previous())
			} else {
				for p.matchAny(token.COMMA) {
					if p.consumeSeq(token.IDENTIFIER) {
						importedSymbols = append(importedSymbols, *p.previous())
					}
				}
				if p.consumeSeq(token.UND) && p.consumeSeq(token.IDENTIFIER) {
					importedSymbols = append(importedSymbols, *p.previous())
				}
			}
		}
		p.consumeSeq(token.AUS)
		if p.consumeSeq(token.STRING) {
			stmt = &ast.ImportStmt{
				FileName:        *p.previous(),
				ImportedSymbols: importedSymbols,
			}
		} else {
			return &ast.BadStmt{
				Tok: *p.peek(),
				Err: p.lastError,
			}
		}

	} else if p.matchAny(token.ALLE, token.REKURSIV) {
		isRecursive := p.previous().Type == token.REKURSIV
		if isRecursive {
			p.consumeSeq(token.ALLE)
		}
		p.consumeSeq(token.MODULE, token.AUS)
		if p.consumeSeq(token.STRING) {
			stmt = &ast.ImportStmt{
				FileName:          *p.previous(),
				ImportedSymbols:   nil,
				IsDirectoryImport: true,
				IsRecursive:       isRecursive,
			}
		} else {
			return &ast.BadStmt{
				Tok: *p.peek(),
				Err: p.lastError,
			}
		}
	} else {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "ein Text Literal, ein Name, 'alle' oder 'rekursiv'"))
		return &ast.BadStmt{
			Tok: *p.peek(),
			Err: p.lastError,
		}
	}
	p.consumeSeq(token.EIN, token.DOT)
	stmt.Range = token.NewRange(binde, p.previous())
	return stmt
}

// either consumes the neccesery . or adds a postfix do-while or repeat
func (p *parser) finishStatement(stmt ast.Statement) ast.Statement {
	if p.matchAny(token.DOT) || p.panicMode {
		return stmt
	}
	// p.checkStatement(stmt)

	cur := p.cur
	count, err := p.expressionOrErr()
	p.panicMode = false
	if p.peek().Type == token.COUNT_MAL {
		p.checkStatement(stmt)
		if err != nil {
			p.err(err.Code, err.Range, err.Msg)
		}
	} else {
		p.cur = cur
		p.consumeSeq(token.DOT)
		return stmt
	}

	if !p.matchAny(token.COUNT_MAL) {
		count_tok := count.Token()
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, count.GetRange(),
			fmt.Sprintf("%s\nWolltest du vor %s vielleicht einen Punkt setzten?",
				ddperror.MsgGotExpected(p.previous(), token.COUNT_MAL), &count_tok,
			),
		)
	}
	tok := p.previous()
	tok.Type = token.WIEDERHOLE
	p.consumeSeq(token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: stmt.GetRange().Start,
			End:   token.NewEndPos(p.previous()),
		},
		While:     *tok,
		Condition: count,
		Body:      stmt,
	}
}

// += -= *= /=
// TODO: fix indexings as assignebles with 'um' after the index
func (p *parser) compoundAssignement() ast.Statement {
	// the many branches are here mostly because of different prepositons
	tok := p.previous()
	operator := ast.BIN_INVALID
	switch tok.Type {
	case token.ERHÖHE:
		operator = ast.BIN_PLUS
	case token.VERRINGERE:
		operator = ast.BIN_MINUS
	case token.VERVIELFACHE:
		operator = ast.BIN_MULT
	case token.TEILE:
		operator = ast.BIN_DIV
	}

	p.consumeAny(token.IDENTIFIER, token.LPAREN)
	varName := p.assigneable()

	// early return for negate as it does not need a second operand
	if tok.Type == token.NEGIERE {
		p.consumeSeq(token.DOT)
		typ := p.typechecker.EvaluateSilent(varName)
		operator := ast.UN_NEGATE
		if ddptypes.Equal(typ, ddptypes.WAHRHEITSWERT) {
			operator = ast.UN_NOT
		}
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   *tok,
			Var:   varName,
			Rhs: &ast.UnaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      *tok,
				Operator: operator,
				Rhs:      varName,
			},
		}
	}

	if tok.Type == token.TEILE {
		p.consumeSeq(token.DURCH)
	} else {
		p.consumeSeq(token.UM)
	}

	operand := p.expression()

	if tok.Type == token.VERSCHIEBE {
		p.consumeSeq(token.BIT, token.NACH)
		p.consumeAny(token.LINKS, token.RECHTS)
		assign_token := tok
		tok = p.previous()
		operator := ast.BIN_LEFT_SHIFT
		if tok.Type == token.RECHTS {
			operator = ast.BIN_RIGHT_SHIFT
		}
		p.consumeSeq(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   *assign_token,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      *tok,
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	} else {
		p.consumeSeq(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   *tok,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      *tok,
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	}
}

// helper to parse assignements which may only be literals
func (p *parser) assignLiteral() ast.Statement {
	ident := p.assigneable() // name of the variable was already consumed
	p.consumeSeq(token.IST)
	expr := p.assignRhs(false) // parse the expression
	// validate that the expression is a literal
	if _, isLiteral := expr.(ast.Literal); !isLiteral {
		if typ := p.typechecker.Evaluate(ident); !ddptypes.Equal(typ, ddptypes.WAHRHEITSWERT) {
			p.err(ddperror.SYN_EXPECTED_LITERAL, expr.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
		}
	}
	ident_tok := ident.Token()
	return p.finishStatement(
		&ast.AssignStmt{
			Range: token.NewRange(&ident_tok, p.peek()),
			Tok:   ident.Token(),
			Var:   ident,
			Rhs:   expr,
		},
	)
}

// helper to parse an Speichere expr in x Assignement
func (p *parser) assignNoLiteral() ast.Statement {
	speichere := p.previous() // Speichere token
	expr := p.expression()
	p.consumeSeq(token.IN)
	p.consumeAny(token.IDENTIFIER, token.LPAREN)
	name := p.assigneable() // name of the variable is the just consumed identifier
	return p.finishStatement(
		&ast.AssignStmt{
			Range: token.NewRange(speichere, p.peek()),
			Tok:   *speichere,
			Var:   name,
			Rhs:   expr,
		},
	)
}

func (p *parser) ifStatement() ast.Statement {
	If := p.previous()          // the already consumed wenn token
	condition := p.expression() // parse the condition
	p.consumeSeq(token.COMMA)   // must be boolean, so an ist is required for grammar
	var Then ast.Statement
	thenScope := p.newScope()
	if p.matchAny(token.DANN) { // with dann: the body is a block statement
		p.consumeSeq(token.COLON)
		Then = p.blockStatement(thenScope)
	} else { // otherwise it is a single statement
		if p.peek().Type == token.COLON { // block statements are only allowed with the syntax above
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, "In einer Wenn Anweisung, muss ein 'dann' vor dem ':' stehen")
		}
		comma := p.previous()
		p.setScope(thenScope)
		Then = p.checkedDeclaration() // parse the single (non-block) statement
		p.exitScope()
		Then = &ast.BlockStmt{
			Range:      Then.GetRange(),
			Colon:      *comma,
			Statements: []ast.Statement{Then},
			Symbols:    thenScope,
		}
	}
	var Else ast.Statement = nil
	// parse a possible sonst statement
	if p.matchAny(token.SONST) {
		if p.previous().Indent == If.Indent {
			elseScope := p.newScope()
			if p.matchAny(token.COLON) {
				Else = p.blockStatement(elseScope) // with colon it is a block statement
			} else { // without it we just parse a single statement
				_else := p.previous()
				p.setScope(elseScope)
				Else = p.checkedDeclaration()
				p.exitScope()
				Else = &ast.BlockStmt{
					Range:      Else.GetRange(),
					Colon:      *_else,
					Statements: []ast.Statement{Else},
					Symbols:    elseScope,
				}
			}
		} else {
			p.decrease()
		}
	} else if p.matchAny(token.WENN) { // if-else blocks are parsed as nested ifs where the else of the first if is an if-statement
		if p.previous().Indent == If.Indent && p.peek().Type == token.ABER {
			p.consumeSeq(token.ABER)
			Else = p.ifStatement() // parse the wenn aber
		} else {
			p.decrease() // no if-else just if, so decrease to parse the next if seperately
		}
	}
	var endPos token.Position
	if Else != nil {
		endPos = Else.GetRange().End
	} else {
		endPos = Then.GetRange().End
	}
	return &ast.IfStmt{
		Range: token.Range{
			Start: token.NewStartPos(If),
			End:   endPos,
		},
		If:        *If,
		Condition: condition,
		Then:      Then,
		Else:      Else,
	}
}

func (p *parser) whileStatement() ast.Statement {
	While := p.previous()
	condition := p.expression()
	p.consumeSeq(token.COMMA)
	var Body ast.Statement
	bodyTable := p.newScope()
	p.resolver.LoopDepth++
	if p.matchAny(token.MACHE) {
		p.consumeSeq(token.COLON)
		Body = p.blockStatement(bodyTable)
	} else {
		is := p.previous()
		p.setScope(bodyTable)
		Body = p.checkedDeclaration()
		p.exitScope()
		Body = &ast.BlockStmt{
			Range:      Body.GetRange(),
			Colon:      *is,
			Statements: []ast.Statement{Body},
			Symbols:    bodyTable,
		}
	}
	p.resolver.LoopDepth--
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(While),
			End:   Body.GetRange().End,
		},
		While:     *While,
		Condition: condition,
		Body:      Body,
	}
}

func (p *parser) doWhileStmt() ast.Statement {
	Do := p.previous()
	p.consumeSeq(token.COLON)
	p.resolver.LoopDepth++
	body := p.blockStatement(nil)
	p.resolver.LoopDepth--
	p.consumeSeq(token.SOLANGE)
	condition := p.expression()
	p.consumeSeq(token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(Do),
			End:   token.NewEndPos(p.previous()),
		},
		While:     *Do,
		Condition: condition,
		Body:      body,
	}
}

func (p *parser) repeatStmt() ast.Statement {
	repeat := p.previous()
	p.consumeSeq(token.COLON)
	p.resolver.LoopDepth++
	body := p.blockStatement(nil)
	p.resolver.LoopDepth--
	count := p.expression()
	p.consumeSeq(token.COUNT_MAL, token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(repeat),
			End:   body.GetRange().End,
		},
		While:     *repeat,
		Condition: count,
		Body:      body,
	}
}

func (p *parser) forStatement() ast.Statement {
	For := p.previous()
	p.consumeAny(token.JEDE, token.JEDEN, token.JEDES)
	pronoun_tok := p.previous()
	TypeTok := p.peek()
	Typ := p.parseType(false)
	if Typ != nil {
		if !ddptypes.MatchesGender(Typ, genderFromForPronoun(pronoun_tok.Type)) {
			p.err(ddperror.SYN_GENDER_MISMATCH, pronoun_tok.Range, fmt.Sprintf("Falsches Pronomen, meintest du %s?", forPronounFromGender(Typ.Gender())))
		}
	}

	p.consumeSeq(token.IDENTIFIER)
	Ident := p.previous()
	iteratorComment := p.getLeadingOrTrailingComment()
	if p.matchAny(token.VON) {
		from := p.expression() // start of the counter
		initializer := &ast.VarDecl{
			Range: token.Range{
				Start: token.NewStartPos(TypeTok),
				End:   from.GetRange().End,
			},
			CommentTok: iteratorComment,
			Type:       Typ,
			NameTok:    *Ident,
			IsPublic:   false,
			Mod:        p.module,
			InitVal:    from,
		}
		p.consumeSeq(token.BIS)
		to := p.expression()                            // end of the counter
		var step ast.Expression = &ast.IntLit{Value: 1} // step-size (default = 1)
		if ddptypes.Equal(Typ, ddptypes.KOMMAZAHL) {
			step = &ast.FloatLit{Value: 1.0}
		}
		if p.matchAny(token.MIT) {
			p.consumeSeq(token.SCHRITTGRÖßE)
			step = p.expression() // custom specified step-size
		}
		p.consumeSeq(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		p.resolver.LoopDepth++
		if p.matchAny(token.MACHE) { // body is a block statement
			p.consumeSeq(token.COLON)
			Body = p.blockStatement(bodyTable).(*ast.BlockStmt)
		} else { // body is a single statement
			Colon := p.previous()
			p.setScope(bodyTable)
			stmt := p.checkedDeclaration()
			p.exitScope()
			// wrap the single statement in a block for variable-scoping of the counter variable in the resolver and typechecker
			Body = &ast.BlockStmt{
				Range: token.Range{
					Start: token.NewStartPos(Colon),
					End:   stmt.GetRange().End,
				},
				Colon:      *Colon,
				Statements: []ast.Statement{stmt},
				Symbols:    bodyTable,
			}
		}
		p.resolver.LoopDepth--
		return &ast.ForStmt{
			Range: token.Range{
				Start: token.NewStartPos(For),
				End:   Body.GetRange().End,
			},
			For:         *For,
			Initializer: initializer,
			To:          to,
			StepSize:    step,
			Body:        Body,
		}
	} else if p.matchAny(token.IN, token.MIT) {
		var index *ast.VarDecl
		if p.previous().Type == token.MIT {
			indexTok := p.peek()
			p.consumeSeq(token.INDEX, token.IDENTIFIER)
			indexComment := p.getLeadingOrTrailingComment()

			index = &ast.VarDecl{
				Range: token.Range{
					Start: token.NewStartPos(indexTok),
					End:   p.previous().Range.End,
				},
				CommentTok: indexComment,
				Type:       ddptypes.ZAHL,
				NameTok:    *p.previous(),
				IsPublic:   false,
				Mod:        p.module,
				InitVal: &ast.IntLit{
					Literal: *indexTok,
					Value:   1,
				},
			}

			p.consumeAny(token.IN)
		}

		In := p.expression()
		initializer := &ast.VarDecl{
			Range: token.Range{
				Start: token.NewStartPos(TypeTok),
				End:   In.GetRange().End,
			},
			CommentTok: iteratorComment,
			Type:       Typ,
			NameTok:    *Ident,
			IsPublic:   false,
			Mod:        p.module,
			InitVal:    In,
		}
		p.consumeSeq(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		if index != nil {
			bodyTable.InsertDecl(index.Name(), index) // add the loop variable to the table
		}
		p.resolver.LoopDepth++
		if p.matchAny(token.MACHE) { // body is a block statement
			p.consumeSeq(token.COLON)
			Body = p.blockStatement(bodyTable).(*ast.BlockStmt)
		} else { // body is a single statement
			Colon := p.previous()
			p.setScope(bodyTable)
			stmt := p.checkedDeclaration()
			p.exitScope()
			// wrap the single statement in a block for variable-scoping of the counter variable in the resolver and typechecker
			Body = &ast.BlockStmt{
				Range: token.Range{
					Start: token.NewStartPos(Colon),
					End:   stmt.GetRange().End,
				},
				Colon:      *Colon,
				Statements: []ast.Statement{stmt},
				Symbols:    bodyTable,
			}
		}
		p.resolver.LoopDepth--
		return &ast.ForRangeStmt{
			Range: token.Range{
				Start: token.NewStartPos(For),
				End:   Body.GetRange().End,
			},
			For:         *For,
			Initializer: initializer,
			Index:       index,
			In:          In,
			Body:        Body,
		}
	}
	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "'von'", "'in'"))
	return &ast.BadStmt{
		Err: p.lastError,
		Tok: *p.previous(),
	}
}

func (p *parser) returnStatement() ast.Statement {
	Return := p.previous()
	var expr ast.Expression
	if p.isCurrentFunctionBool {
		expr = p.assignRhs(true)
	} else {
		expr = p.expression()
	}

	p.consumeSeq(token.ZURÜCK, token.DOT)
	rnge := token.NewRange(Return, p.previous())
	if p.currentFunction == nil {
		p.err(ddperror.SEM_GLOBAL_RETURN, rnge, ddperror.MSG_GLOBAL_RETURN)
	}
	return &ast.ReturnStmt{
		Range:  rnge,
		Func:   p.currentFunction,
		Return: *Return,
		Value:  expr,
	}
}

func (p *parser) voidReturnOrBreak() ast.Statement {
	Leave := p.previous()
	p.consumeSeq(token.DIE)
	if p.matchAny(token.SCHLEIFE) {
		p.consumeSeq(token.DOT)
		return &ast.BreakContinueStmt{
			Range: token.NewRange(Leave, p.previous()),
			Tok:   *Leave,
		}
	}

	p.consumeSeq(token.FUNKTION, token.DOT)
	rnge := token.NewRange(Leave, p.previous())
	if p.currentFunction == nil {
		p.err(ddperror.SEM_GLOBAL_RETURN, rnge, ddperror.MSG_GLOBAL_RETURN)
	}
	return &ast.ReturnStmt{
		Range:  token.NewRange(Leave, p.previous()),
		Func:   p.currentFunction,
		Return: *Leave,
		Value:  nil,
	}
}

func (p *parser) continueStatement() ast.Statement {
	Continue := p.previous()
	p.consumeSeq(token.MIT, token.DER, token.SCHLEIFE, token.FORT, token.DOT)
	return &ast.BreakContinueStmt{
		Range: token.NewRange(Continue, p.previous()),
		Tok:   *Continue,
	}
}

func (p *parser) blockStatement(symbols ast.SymbolTable) ast.Statement {
	colon := p.previous()
	if p.peek().Line() <= colon.Line() {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, "Nach einem Doppelpunkt muss eine neue Zeile beginnen")
	}
	statements := make([]ast.Statement, 0)
	indent := colon.Indent + 1

	if symbols == nil {
		symbols = p.newScope()
	}
	p.setScope(symbols)
	for p.peek().Indent >= indent && !p.atEnd() {
		if stmt := p.checkedDeclaration(); stmt != nil {
			statements = append(statements, stmt)
		}
	}
	p.exitScope()

	return &ast.BlockStmt{
		Range:      token.NewRange(colon, p.previous()),
		Colon:      *colon,
		Statements: statements,
		Symbols:    symbols,
	}
}

func (p *parser) todoStmt() ast.Statement {
	p.warn(ddperror.SEM_TODO_STMT_FOUND, p.previous().Range, "Für diesen Teil des Programms fehlt eine Implementierung und es wird ein Laufzeitfehler ausgelöst")
	return &ast.TodoStmt{
		Tok: *p.previous(),
	}
}

func (p *parser) expressionStatement() ast.Statement {
	return p.finishStatement(&ast.ExprStmt{Expr: p.expression()})
}
