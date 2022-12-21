package parser

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast/resolver"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast/typechecker"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddperror"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state when parsing a .ddp file into an AST
type Parser struct {
	tokens       []token.Token    // the tokens to parse (without comments)
	comments     []token.Token    // all the comments from the original tokens slice
	cur          int              // index of the current token
	errorHandler ddperror.Handler // a function to which errors are passed

	funcAliases     []ast.FuncAlias          // all found aliases (+ inbuild aliases)
	currentFunction string                   // function which is currently being parsed
	panicMode       bool                     // flag to not report following errors
	Errored         bool                     // wether the parser found an error
	resolver        *resolver.Resolver       // used to resolve every node directly after it has been parsed
	typechecker     *typechecker.Typechecker // used to typecheck every node directly after it has been parsed
}

// returns a new parser, ready to parse the provided tokens
func New(tokens []token.Token, errorHandler ddperror.Handler) *Parser {
	// default error handler does nothing
	if errorHandler == nil {
		errorHandler = ddperror.EmptyHandler
	}

	if len(tokens) == 0 {
		tokens = []token.Token{{Type: token.EOF}} // we need at least one EOF at the end of the tokens slice
	}

	// the last token must be EOF
	if tokens[len(tokens)-1].Type != token.EOF {
		tokens = append(tokens, token.Token{Type: token.EOF})
	}

	pTokens := make([]token.Token, 0, len(tokens))
	pComments := make([]token.Token, 0)
	// filter the comments out
	for i := range tokens {
		if tokens[i].Type == token.COMMENT {
			pComments = append(pComments, tokens[i])
		} else {
			pTokens = append(pTokens, tokens[i])
		}
	}

	aliases := make([]ast.FuncAlias, 0)
	parser := &Parser{
		tokens:       pTokens,
		comments:     pComments,
		cur:          0,
		errorHandler: errorHandler,
		funcAliases:  aliases,
		panicMode:    false,
		Errored:      false,
		resolver:     &resolver.Resolver{},
		typechecker:  &typechecker.Typechecker{},
	}

	return parser
}

// parse the provided tokens into an Ast
func (p *Parser) Parse() *ast.Ast {
	Ast := &ast.Ast{
		Statements: make([]ast.Statement, 0),
		Symbols:    ast.NewSymbolTable(nil),
		Faulty:     false,
		File:       p.tokens[0].File,
	}

	// prepare the resolver and typechecker with the inbuild symbols and types
	p.resolver = resolver.New(Ast, p.errorHandler)
	p.typechecker = typechecker.New(Ast.Symbols, p.errorHandler)

	// main parsing loop
	for !p.atEnd() {
		if stmt := p.checkedDeclaration(); stmt != nil {
			Ast.Statements = append(Ast.Statements, stmt)
		}
	}

	// if any error occured, the AST is faulty
	if p.Errored || p.resolver.Errored || p.typechecker.Errored {
		Ast.Faulty = true
	}

	return Ast
}

// if an error was encountered we synchronize to a point where correct parsing is possible again
func (p *Parser) synchronize() {
	p.panicMode = false
	p.Errored = true

	//p.advance() // maybe this needs to stay?
	for !p.atEnd() {
		if p.previous().Type == token.DOT { // a . ends statements, so we can continue parsing
			return
		}
		// these tokens typically begin statements which begin a new node
		switch p.peek().Type {
		case token.DER, token.DIE, token.WENN, token.FÜR, token.GIB, token.SOLANGE, token.COLON, token.MACHE, token.WIEDERHOLE:
			return
		}
		p.advance()
	}
}

func (p *Parser) checkedDeclaration() ast.Statement {
	stmt := p.declaration() // parse the node
	if stmt != nil {        // nil check, for alias declarations that aren't Ast Nodes
		p.resolver.ResolveNode(stmt)      // resolve symbols in it (variables, functions, ...)
		p.typechecker.TypecheckNode(stmt) // typecheck the node
	}
	if p.panicMode { // synchronize the parsing if we are in panic mode
		p.synchronize()
	}
	return stmt
}

// entry point for the recursive descent parsing
func (p *Parser) declaration() ast.Statement {
	if p.match(token.DER, token.DIE) { // might indicate a function or variable declaration
		switch p.previous().Type {
		case token.DER:
			switch p.peek().Type {
			case token.BOOLEAN, token.TEXT, token.BUCHSTABE:
				p.advance()                                    // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration()} // parse the declaration
			case token.ALIAS:
				p.advance()
				return p.aliasDecl() // TODO: fix errors/crashes if this returns nil
			default:
				p.decrease() // decrease, so expressionStatement() can recognize it as expression
			}
		case token.DIE:
			switch p.peek().Type {
			case token.ZAHL, token.KOMMAZAHL, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.TEXT, token.BOOLEAN:
				p.advance()                                    // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration()} // parse the declaration
			case token.FUNKTION:
				p.advance()
				return &ast.DeclStmt{Decl: p.funcDeclaration()} // parse the function declaration
			default:
				p.decrease() // decrease, so expressionStatement() can recognize it as expression
			}
		}
	}

	return p.statement() // no declaration, so it must be a statement
}

// helper for boolean assignments
func (p *Parser) assignRhs() ast.Expression {
	var expr ast.Expression // the final expression

	if p.match(token.TRUE, token.FALSE) {
		// parse possible wahr/falsch wenn syntax
		if p.match(token.WENN) {
			// if it is false, we add a unary bool-negate into the ast
			if tok := p.tokens[p.cur-2]; tok.Type == token.FALSE {
				tok.Type = token.NICHT
				rhs := p.expression() // the actual boolean expression after falsch wenn, which is negated
				expr = &ast.UnaryExpr{
					Range: token.Range{
						Start: token.NewStartPos(tok),
						End:   rhs.GetRange().End,
					},
					Operator: tok,
					Rhs:      rhs,
				}
			} else {
				expr = p.expression() // wahr wenn simply becomes a normal expression
			}
		} else { // no wahr/falsch wenn, only a boolean literal
			p.decrease() // decrease, so expression() can recognize the literal
			expr = p.expression()

			// validate that nothing follows after the literal
			if _, ok := expr.(*ast.BoolLit); !ok {
				p.err(expr.Token(), expr.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
			}
		}
	} else {
		expr = p.expression() // no wahr/falsch, so a normal expression
	}

	return expr
}

func (p *Parser) varDeclaration() ast.Declaration {
	begin := p.peekN(-2)
	comment := p.commentBeforePos(begin.Range.Start, begin.File)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}
	p.decrease()
	typ := p.parseType()

	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: &ParserError{
				rang: token.NewRange(p.peekN(-2), p.peek()),
				file: p.peek().File,
				msg:  "Es wurde ein Variablen Name erwartet",
			},
			Tok: p.peek(),
		}
	}

	name := p.previous()
	p.consume(token.IST)
	var expr ast.Expression

	if typ != ddptypes.Bool() && typ.IsList { // TODO: fix this with function calls and groupings
		expr = p.expression()
		if p.match(token.COUNT_MAL) {
			value := p.expression()
			expr = &ast.ListLit{
				Tok:    expr.Token(),
				Range:  token.NewRange(expr.Token(), p.previous()),
				Type:   typ,
				Values: nil,
				Count:  expr,
				Value:  value,
			}
		}
	} else {
		expr = p.assignRhs()
	}

	p.consume(token.DOT)
	// prefer trailing comments as long as they are on the same line
	if trailingComment := p.commentAfterPos(p.previous().Range.End, p.previous().File); trailingComment != nil && trailingComment.Range.Start.Line == p.previous().Range.End.Line {
		comment = trailingComment
	}
	return &ast.VarDecl{
		Range:   token.NewRange(begin, p.previous()),
		Comment: comment,
		Type:    typ,
		Name:    name,
		InitVal: expr,
	}
}

func (p *Parser) funcDeclaration() ast.Declaration {
	valid := true              // checks if the function is valid and may be appended to the parser state as p.errored = !valid
	validate := func(b bool) { // helper for setting the valid flag (to simplify some big boolean expressions)
		if !b {
			valid = false
		}
	}

	errorMessage := ""
	perr := func(tok token.Token, rnge token.Range, msg string) {
		p.err(tok, rnge, msg)
		valid = false
		if errorMessage == "" {
			errorMessage = msg
		}
	}

	begin := p.peekN(-2)
	comment := p.commentBeforePos(begin.Range.Start, begin.File)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}
	Funktion := p.previous() // save the token
	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: &ParserError{
				rang: token.NewRange(begin, p.peek()),
				file: p.peek().File,
				msg:  "Es wurde ein Funktions Name erwartet",
			},
			Tok: p.peek(),
		}
	}
	name := p.previous()

	// parse the parameter declaration
	// parameter names and types are declared seperately
	var paramNames []token.Token = nil
	var paramTypes []ddptypes.ParameterType = nil
	var paramComments []*token.Token = nil
	if p.match(token.MIT) { // the function takes at least 1 parameter
		singleParameter := true
		if p.matchN(token.DEN, token.PARAMETERN) {
			singleParameter = false
		} else if !p.matchN(token.DEM, token.PARAMETER) {
			valid = false
			p.err(p.peek(), p.peek().Range, "Es wurde 'de[n/m] Parameter[n]' erwartet aber '%s' gefunden", p.peek())
		}
		validate(p.consume(token.IDENTIFIER))
		paramNames = append(make([]token.Token, 0), p.previous()) // append the first parameter name
		paramComments = append(make([]*token.Token, 0), p.getLeadingOrTrailingComment())
		if !singleParameter {
			addParamName := func(name token.Token) {
				if containsLiteral(paramNames, name.Literal) { // check that each parameter name is unique
					valid = false
					perr(name, name.Range, fmt.Sprintf("Ein Parameter mit dem Namen '%s' ist bereits vorhanden", name.Literal))
				}
				paramNames = append(paramNames, name)                                  // append the parameter name
				paramComments = append(paramComments, p.getLeadingOrTrailingComment()) // addParamName is always being called with name == p.previous()
			}
			if p.match(token.UND) {
				validate(p.consume(token.IDENTIFIER))
				addParamName(p.previous())
			} else {
				for p.match(token.COMMA) { // the function takes multiple parameters
					if !p.consume(token.IDENTIFIER) {
						break
					}
					addParamName(p.previous())
				}
				if !p.consumeN(token.UND, token.IDENTIFIER) {
					perr(p.peek(), p.peek().Range, fmt.Sprintf("Es wurde 'und <letzter Parameter>' erwartet aber %s gefunden\nMeintest du vielleicht 'dem Parameter' anstatt 'den Parametern'?", p.peek().Literal))
				}
				addParamName(p.previous())
			}
		}
		// parse the types of the parameters
		validate(p.consumeN(token.VOM, token.TYP))
		firstType, ref := p.parseReferenceType()
		validate(firstType.Primitive != ddptypes.ILLEGAL)
		paramTypes = append(make([]ddptypes.ParameterType, 0), ddptypes.ParameterType{Type: firstType, IsReference: ref}) // append the first parameter type
		if !singleParameter {
			addType := func() {
				// validate the parameter type and append it
				typ, ref := p.parseReferenceType()
				validate(typ.Primitive != ddptypes.ILLEGAL)
				paramTypes = append(paramTypes, ddptypes.ParameterType{Type: typ, IsReference: ref})
			}
			if p.match(token.UND) {
				addType()
			} else {
				for p.match(token.COMMA) { // parse the other parameter types
					if p.check(token.GIBT) { // , gibt indicates the end of the parameter list
						break
					}
					addType()
				}
				p.consume(token.UND)
				addType()
			}
		}
		p.consume(token.COMMA)
	}
	// we need as many parmeter names as types
	if len(paramNames) != len(paramTypes) {
		valid = false
		perr(p.previous(), p.previous().Range, fmt.Sprintf("Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)", len(paramNames), len(paramTypes)))
	}

	// parse the return type declaration
	validate(p.consume(token.GIBT))
	p.match(token.EINE, token.EINEN) // not neccessary
	Typ := p.parseTypeOrVoid()
	if Typ.Primitive == ddptypes.ILLEGAL {
		valid = false
	}

	validate(p.consumeN(token.ZURÜCK, token.COMMA))
	bodyStart := -1
	definedIn := token.Token{Type: token.ILLEGAL}
	if p.matchN(token.MACHT, token.COLON) {
		bodyStart = p.cur                             // save the body start-position for later, we first need to parse aliases to enable recursion
		indent := p.previous().Indent + 1             // indentation level of the function body
		for p.peek().Indent >= indent && !p.atEnd() { // advance to the alias definitions by checking the indentation
			p.advance()
		}
	} else {
		validate(p.consumeN(token.IST, token.IN, token.STRING, token.DEFINIERT))
		definedIn = p.peekN(-2)
		switch filepath.Ext(strings.Trim(definedIn.Literal, "\"")) {
		case ".c", ".lib", ".a", ".o":
		default:
			perr(definedIn, definedIn.Range, fmt.Sprintf("Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden", definedIn.Literal))
		}
	}

	// parse the alias definitions before the body to enable recursion
	validate(p.consumeN(token.UND, token.KANN, token.SO, token.BENUTZT, token.WERDEN, token.COLON, token.STRING)) // at least 1 alias is required
	aliases := make([]token.Token, 0)
	if p.previous().Type == token.STRING {
		aliases = append(aliases, p.previous())
	} else {
		valid = false
	}
	// append the raw aliases
	for (p.match(token.COMMA) || p.match(token.ODER)) && p.peek().Indent > 0 && !p.atEnd() {
		if p.consume(token.STRING) {
			aliases = append(aliases, p.previous())
		}
	}

	// map function parameters to their type (given to the alias if it is valid)
	paramTypesMap := map[string]ddptypes.ParameterType{}
	for i, v := range paramNames {
		if i < len(paramTypes) {
			paramTypesMap[v.Literal] = paramTypes[i]
		}
	}

	// scan the raw aliases into tokens
	funcAliases := make([]ast.FuncAlias, 0)
	for _, v := range aliases {
		// scan the raw alias withouth the ""
		didError := false
		errHandleWrapper := func(err ddperror.Error) { didError = true; p.errorHandler(err) }
		if alias, err := scanner.ScanAlias(v, errHandleWrapper); err == nil && !didError {
			if len(alias) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
				perr(v, v.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if validateAlias(alias, paramNames, paramTypes) { // check that the alias fits the function
				if fun := p.aliasExists(alias); fun != nil { // check that the alias does not already exist for another function
					perr(v, v.Range, fmt.Sprintf("Der Alias steht bereits für die Funktion '%s'", *fun))
				} else { // the alias is valid so we append it
					funcAliases = append(funcAliases, ast.FuncAlias{Tokens: alias, Original: v, Func: name.Literal, Args: paramTypesMap})
				}
			} else {
				valid = false
				perr(v, v.Range, "Ein Funktions Alias muss jeden Funktions Parameter genau ein mal enthalten")
			}
		}
	}

	aliasEnd := p.cur // save the end of the function declaration for later

	if p.currentFunction != "" {
		valid = false
		perr(begin, begin.Range, "Es können nur globale Funktionen deklariert werden")
	}

	if !valid {
		p.Errored = true
		p.cur = aliasEnd
		return &ast.BadDecl{
			Err: &ParserError{
				rang: token.NewRange(begin, p.previous()),
				file: p.previous().File,
				msg:  errorMessage,
			},
			Tok: Funktion,
		}
	}

	p.funcAliases = append(p.funcAliases, funcAliases...)

	decl := &ast.FuncDecl{
		Range:         token.NewRange(begin, p.previous()),
		Comment:       comment,
		Tok:           begin,
		Name:          name,
		ParamNames:    paramNames,
		ParamTypes:    paramTypes,
		ParamComments: paramComments,
		Type:          Typ,
		Body:          nil,
		ExternFile:    definedIn,
		Aliases:       funcAliases,
	}

	// parse the body after the aliases to enable recursion
	var body *ast.BlockStmt = nil
	if bodyStart != -1 {
		p.cur = bodyStart // go back to the body
		p.currentFunction = name.Literal

		bodyTable := p.newScope() // temporary symbolTable for the function parameters
		globalScope := bodyTable.Enclosing
		if existed := globalScope.InsertFunc(p.currentFunction, decl); existed { // insert the name of the current function
			p.err(decl.Name, decl.Name.Range, "Die Funktion '%s' existiert bereits", decl.Name.Literal) // functions may only be declared once
		}
		// add the parameters to the table
		for i, l := 0, len(paramNames); i < l; i++ {
			bodyTable.InsertVar(paramNames[i].Literal, &ast.VarDecl{Name: paramNames[i], Type: paramTypes[i].Type, Range: token.NewRange(paramNames[i], paramNames[i]), Comment: paramComments[i]})
		}
		body = p.blockStatement(bodyTable).(*ast.BlockStmt) // parse the body with the parameters in the current table
		decl.Body = body

		// check that the function has a return statement if it needs one
		if Typ != ddptypes.Void() { // only if the function does not return void
			if len(body.Statements) < 1 { // at least the return statement is needed
				perr(Funktion, body.Range, "Am Ende einer Funktion die etwas zurück gibt muss eine Rückgabe stehen")
			} else {
				// the last statement must be a return statement
				lastStmt := body.Statements[len(body.Statements)-1]
				if _, ok := lastStmt.(*ast.ReturnStmt); !ok {
					perr(lastStmt.Token(), lastStmt.GetRange(), "Am Ende einer Funktion die etwas zurück gibt muss eine Rückgabe stehen")
				}
			}
		}
	} else {
		if existed := p.resolver.CurrentTable.InsertFunc(name.Literal, decl); existed { // insert the name of the current function
			p.err(decl.Name, decl.Name.Range, "Die Funktion '%s' existiert bereits", decl.Name.Literal) // functions may only be declared once
		}
	}

	p.currentFunction = ""
	p.cur = aliasEnd // go back to the end of the function to continue parsing

	return decl
}

// helper for funcDeclaration to check that every parameter is provided exactly once
func validateAlias(alias []token.Token, paramNames []token.Token, paramTypes []ddptypes.ParameterType) bool {
	isAliasExpr := func(t token.Token) bool { return t.Type == token.ALIAS_PARAMETER } // helper to check for parameters
	if countElements(alias, isAliasExpr) != len(paramNames) {                          // validate that the alias contains as many parameters as the function
		return false
	}
	nameSet := map[string]ddptypes.ParameterType{} // set that holds the parameter names contained in the alias and their corresponding type
	for i, v := range paramNames {
		if i < len(paramTypes) {
			nameSet[v.Literal] = paramTypes[i]
		}
	}
	// validate that each parameter is contained in the alias exactly once
	// and fill in the AliasInfo
	for i, v := range alias {
		if isAliasExpr(v) {
			k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
			if argTyp, ok := nameSet[k]; ok {
				alias[i].AliasInfo = &argTyp
				delete(nameSet, k)
			} else {
				return false
			}
		}
	}
	return true
}

// helper to check if an alias already exists for a function
// returns functions name or nil
func (p *Parser) aliasExists(alias []token.Token) *string {
	for i := range p.funcAliases {
		if slicesEqual(alias, p.funcAliases[i].Tokens, tokenEqual) {
			return &p.funcAliases[i].Func
		}
	}
	return nil
}

func (p *Parser) aliasDecl() ast.Statement {
	begin := p.peekN(-2)
	p.consume(token.STRING)
	aliasTok := p.previous()
	p.consumeN(token.STEHT, token.FÜR, token.DIE, token.FUNKTION, token.IDENTIFIER)
	fun := p.previous()

	funDecl, ok := p.resolver.CurrentTable.LookupFunc(fun.Literal)
	if !ok {
		p.err(fun, fun.Range, "Die Funktion %s existiert nicht", fun.Literal)
		return nil
	}

	// map function parameters to their type (given to the alias if it is valid)
	paramTypes := map[string]ddptypes.ParameterType{}
	for i, v := range funDecl.ParamNames {
		if i < len(funDecl.ParamTypes) {
			paramTypes[v.Literal] = funDecl.ParamTypes[i]
		}
	}

	// scan the raw alias withouth the ""
	var alias *ast.FuncAlias
	if aliasTokens, err := scanner.ScanAlias(aliasTok, p.errorHandler); err != nil {
		p.err(aliasTok, aliasTok.Range, fmt.Sprintf("Der Funktions Alias ist ungültig (%s)", err.Error()))
	} else {
		if len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
			p.err(aliasTok, aliasTok.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
		} else if validateAlias(aliasTokens, funDecl.ParamNames, funDecl.ParamTypes) { // check that the alias fits the function
			if fun := p.aliasExists(aliasTokens); fun != nil { // check that the alias does not already exist for another function
				p.err(aliasTok, aliasTok.Range, fmt.Sprintf("Der Alias steht bereits für die Funktion '%s'", *fun))
			} else { // the alias is valid so we append it
				alias = &ast.FuncAlias{Tokens: aliasTokens, Original: aliasTok, Func: funDecl.Name.Literal, Args: paramTypes}
			}
		} else {
			p.err(aliasTok, aliasTok.Range, "Ein Funktions Alias muss jeden Funktions Parameter genau ein mal enthalten")
		}
	}

	p.consume(token.DOT)

	if begin.Indent > 0 {
		msg := "Ein Alias darf nur im globalen Bereich deklariert werden!"
		rnge := token.NewRange(begin, p.previous())
		p.err(begin, rnge, msg)
		return &ast.BadStmt{
			Err: &ParserError{
				rang: rnge,
				file: begin.File,
				msg:  msg,
			},
			Tok: begin,
		}
	} else if alias != nil {
		p.funcAliases = append(p.funcAliases, *alias)
		funDecl.Aliases = append(funDecl.Aliases, *alias)
	}
	return nil
}

// parse a single statement
func (p *Parser) statement() ast.Statement {
	// check for assignement
	if p.match(token.IDENTIFIER) {
		if p.peek().Type == token.IST || p.peek().Type == token.AN {
			return p.assignLiteral() // x ist ... assignements may only have literals, so we use this helper function
		} else {
			p.decrease() // no assignement, so probably an expressionStatement()
		}
	}

	// parse all possible statements
	switch p.peek().Type {
	case token.ADDIERE, token.ERHÖHE, token.SUBTRAHIERE, token.VERRINGERE,
		token.MULTIPLIZIERE, token.VERVIELFACHE, token.DIVIDIERE, token.TEILE,
		token.VERSCHIEBE, token.NEGIERE:
		p.advance()
		return p.compoundAssignement()
	case token.SPEICHERE:
		p.consume(token.SPEICHERE)
		return p.assignNoLiteral() // Speichere ... in x, where non-literal expressions are allowed
	case token.WENN:
		p.consume(token.WENN)
		return p.ifStatement()
	case token.SOLANGE:
		p.consume(token.SOLANGE)
		return p.whileStatement()
	case token.MACHE:
		p.consume(token.MACHE)
		return p.doWhileStmt()
	case token.WIEDERHOLE:
		p.consume(token.WIEDERHOLE)
		return p.repeatStmt()
	case token.FÜR:
		p.consume(token.FÜR)
		return p.forStatement()
	case token.GIB:
		p.consume(token.GIB)
		return p.returnStatement()
	case token.VERLASSE:
		p.consume(token.VERLASSE)
		return p.voidReturn()
	case token.COLON:
		p.consume(token.COLON)
		return p.blockStatement(nil)
	}

	// no other statement was found, so interpret it as expression statement, whose result will be discarded
	return p.expressionStatement()
}

// either consumes the neccesery . or adds a postfix do-while or repeat
func (p *Parser) finishStatement(stmt ast.Statement) ast.Statement {
	if p.match(token.DOT) {
		return stmt
	}
	count := p.expression()
	p.consume(token.COUNT_MAL)
	tok := p.previous()
	tok.Type = token.WIEDERHOLE
	p.consume(token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: stmt.GetRange().Start,
			End:   token.NewEndPos(p.previous()),
		},
		While:     tok,
		Condition: count,
		Body:      stmt,
	}
}

// += -= *= /=
// TODO: fix indexings as assignebles with 'um' after the index
func (p *Parser) compoundAssignement() ast.Statement {
	// the many branches are here mostly because of different prepositons
	operator := p.previous()
	var operand ast.Expression
	var varName ast.Assigneable
	if operator.Type == token.SUBTRAHIERE { // subtrahiere VON, so the operands are reversed
		operand = p.primary(nil)
	} else {
		p.consume(token.IDENTIFIER)
		if p.match(token.LPAREN) { // indexings may be enclosed in parens to prevent the 'um' from being interpretetd as bitshift
			varName = p.assigneable()
			p.consume(token.RPAREN)
		} else {
			varName = p.assigneable()
		}

		// early return for negate
		if operator.Type == token.NEGIERE {
			p.consume(token.DOT)
			return &ast.AssignStmt{
				Range: token.NewRange(operator, p.previous()),
				Tok:   operator,
				Var:   varName,
				Rhs: &ast.UnaryExpr{
					Range:    token.NewRange(operator, p.previous()),
					Operator: operator,
					Rhs:      varName,
				},
			}
		}
	}
	switch operator.Type {
	case token.ADDIERE, token.MULTIPLIZIERE:
		p.consume(token.MIT)
	case token.ERHÖHE, token.VERRINGERE, token.VERVIELFACHE, token.VERSCHIEBE:
		p.consume(token.UM)
	case token.SUBTRAHIERE:
		p.consume(token.VON)
	case token.DIVIDIERE, token.TEILE:
		p.consume(token.DURCH)
	}
	if operator.Type == token.SUBTRAHIERE { // order of operands is reversed
		p.consume(token.IDENTIFIER)
		varName = p.assigneable()
	} else {
		operand = p.primary(nil)
	}

	switch operator.Type {
	case token.ADDIERE, token.SUBTRAHIERE, token.MULTIPLIZIERE, token.DIVIDIERE:
		p.consumeN(token.UND, token.SPEICHERE, token.DAS, token.ERGEBNIS, token.IN, token.IDENTIFIER)
		targetName := p.assigneable()
		p.consume(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(operator, p.previous()),
			Tok:   operator,
			Var:   targetName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(operator, p.previous()),
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	case token.ERHÖHE, token.VERRINGERE, token.VERVIELFACHE, token.TEILE:
		p.consume(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(operator, p.previous()),
			Tok:   operator,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(operator, p.previous()),
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	case token.VERSCHIEBE:
		p.consumeN(token.BIT, token.NACH)
		p.consumeAny(token.LINKS, token.RECHTS)
		tok := operator
		operator = p.previous()
		p.consume(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(operator, p.previous()),
			Tok:   tok,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(operator, p.previous()),
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	default: // unreachable
		return &ast.BadStmt{}
	}
}

// helper to parse assignements which may only be literals
func (p *Parser) assignLiteral() ast.Statement {
	ident := p.assigneable() // name of the variable was already consumed
	p.consume(token.IST)
	expr := p.assignRhs() // parse the expression
	// validate that the expression is a literal
	switch expr := expr.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.CharLit, *ast.ListLit:
	default:
		if typ := p.typechecker.Evaluate(ident); typ != ddptypes.Bool() {
			p.err(expr.Token(), expr.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
		}
	}
	return p.finishStatement(
		&ast.AssignStmt{
			Range: token.NewRange(ident.Token(), p.peek()),
			Tok:   ident.Token(),
			Var:   ident,
			Rhs:   expr,
		},
	)
}

// helper to parse an Speichere expr in x Assignement
func (p *Parser) assignNoLiteral() ast.Statement {
	speichere := p.previous() // Speichere token
	expr := p.expression()
	p.consumeN(token.IN, token.IDENTIFIER)
	name := p.assigneable() // name of the variable is the just consumed identifier
	return p.finishStatement(
		&ast.AssignStmt{
			Range: token.NewRange(speichere, p.peek()),
			Tok:   speichere,
			Var:   name,
			Rhs:   expr,
		},
	)
}

func (p *Parser) ifStatement() ast.Statement {
	If := p.previous()          // the already consumed wenn token
	condition := p.expression() // parse the condition
	p.consume(token.COMMA)      // must be boolean, so an ist is required for grammar
	var Then ast.Statement
	thenScope := p.newScope()
	if p.match(token.DANN) { // with dann: the body is a block statement
		p.consume(token.COLON)
		Then = p.blockStatement(thenScope)
	} else { // otherwise it is a single statement
		if p.peek().Type == token.COLON { // block statements are only allowed with the syntax above
			p.err(p.peek(), p.peek().Range, "In einer Wenn Anweisung, muss ein 'dann' vor einem ':' stehen")
		}
		comma := p.previous()
		p.setScope(thenScope)
		Then = p.checkedDeclaration() // parse the single (non-block) statement
		p.exitScope()
		Then = &ast.BlockStmt{
			Range:      Then.GetRange(),
			Colon:      comma,
			Statements: []ast.Statement{Then},
			Symbols:    thenScope,
		}
	}
	var Else ast.Statement = nil
	// parse a possible sonst statement
	if p.match(token.SONST) {
		if p.previous().Indent == If.Indent {
			elseScope := p.newScope()
			if p.match(token.COLON) {
				Else = p.blockStatement(elseScope) // with colon it is a block statement
			} else { // without it we just parse a single statement
				_else := p.previous()
				p.setScope(elseScope)
				Else = p.checkedDeclaration()
				p.exitScope()
				Else = &ast.BlockStmt{
					Range:      Else.GetRange(),
					Colon:      _else,
					Statements: []ast.Statement{Else},
					Symbols:    elseScope,
				}
			}
		} else {
			p.decrease()
		}
	} else if p.match(token.WENN) { // if-else blocks are parsed as nested ifs where the else of the first if is an if-statement
		if p.previous().Indent == If.Indent && p.peek().Type == token.ABER {
			p.consume(token.ABER)
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
		If:        If,
		Condition: condition,
		Then:      Then,
		Else:      Else,
	}
}

func (p *Parser) whileStatement() ast.Statement {
	While := p.previous()
	condition := p.expression()
	p.consume(token.COMMA)
	var Body ast.Statement
	bodyTable := p.newScope()
	if p.match(token.MACHE) {
		p.consume(token.COLON)
		Body = p.blockStatement(bodyTable)
	} else {
		is := p.previous()
		p.setScope(bodyTable)
		Body = p.checkedDeclaration()
		p.exitScope()
		Body = &ast.BlockStmt{
			Range:      Body.GetRange(),
			Colon:      is,
			Statements: []ast.Statement{Body},
			Symbols:    bodyTable,
		}
	}
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(While),
			End:   Body.GetRange().End,
		},
		While:     While,
		Condition: condition,
		Body:      Body,
	}
}

func (p *Parser) doWhileStmt() ast.Statement {
	Do := p.previous()
	p.consume(token.COLON)
	body := p.blockStatement(nil)
	p.consume(token.SOLANGE)
	condition := p.expression()
	p.consume(token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(Do),
			End:   token.NewEndPos(p.previous()),
		},
		While:     Do,
		Condition: condition,
		Body:      body,
	}
}

func (p *Parser) repeatStmt() ast.Statement {
	repeat := p.previous()
	p.consume(token.COLON)
	body := p.blockStatement(nil)
	count := p.expression()
	p.consumeN(token.COUNT_MAL, token.DOT)
	return &ast.WhileStmt{
		Range: token.Range{
			Start: token.NewStartPos(repeat),
			End:   body.GetRange().End,
		},
		While:     repeat,
		Condition: count,
		Body:      body,
	}
}

func (p *Parser) forStatement() ast.Statement {
	For := p.previous()
	p.consumeAny(token.JEDE, token.JEDEN)
	TypeTok := p.peek()
	Typ := p.parseType()
	p.consume(token.IDENTIFIER)
	Ident := p.previous()
	iteratorComment := p.getLeadingOrTrailingComment()
	if p.match(token.VON) {
		from := p.expression() // start of the counter
		initializer := &ast.VarDecl{
			Range: token.Range{
				Start: token.NewStartPos(TypeTok),
				End:   from.GetRange().End,
			},
			Comment: iteratorComment,
			Type:    Typ,
			Name:    Ident,
			InitVal: from,
		}
		p.consume(token.BIS)
		to := p.expression()                            // end of the counter
		var step ast.Expression = &ast.IntLit{Value: 1} // step-size (default = 1)
		if p.match(token.MIT) {
			p.consume(token.SCHRITTGRÖßE)
			step = p.expression() // custom specified step-size
		}
		p.consume(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                       // temporary symbolTable for the loop variable
		bodyTable.InsertVar(Ident.Literal, initializer) // add the loop variable to the table
		if p.match(token.MACHE) {                       // body is a block statement
			p.consume(token.COLON)
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
				Colon:      Colon,
				Statements: []ast.Statement{stmt},
				Symbols:    bodyTable,
			}
		}
		return &ast.ForStmt{
			Range: token.Range{
				Start: token.NewStartPos(For),
				End:   Body.GetRange().End,
			},
			For:         For,
			Initializer: initializer,
			To:          to,
			StepSize:    step,
			Body:        Body,
		}
	} else if p.match(token.IN) {
		In := p.expression()
		initializer := &ast.VarDecl{
			Range: token.Range{
				Start: token.NewStartPos(TypeTok),
				End:   In.GetRange().End,
			},
			Type:    Typ,
			Name:    Ident,
			InitVal: In,
		}
		p.consume(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                       // temporary symbolTable for the loop variable
		bodyTable.InsertVar(Ident.Literal, initializer) // add the loop variable to the table
		if p.match(token.MACHE) {                       // body is a block statement
			p.consume(token.COLON)
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
				Colon:      Colon,
				Statements: []ast.Statement{stmt},
				Symbols:    bodyTable,
			}
		}
		return &ast.ForRangeStmt{
			Range: token.Range{
				Start: token.NewStartPos(For),
				End:   Body.GetRange().End,
			},
			For:         For,
			Initializer: initializer,
			In:          In,
			Body:        Body,
		}
	}
	msg := fmt.Sprintf("Es wurde VON oder IN erwartet, aber '%s' gefunden", p.previous())
	p.err(p.previous(), p.peek().Range, msg)
	return &ast.BadStmt{
		Err: &ParserError{
			rang: token.NewRange(For, p.previous()),
			file: p.previous().File,
			msg:  msg,
		},
		Tok: p.previous(),
	}
}

func (p *Parser) returnStatement() ast.Statement {
	Return := p.previous()
	expr := p.expression()
	p.consumeN(token.ZURÜCK, token.DOT)
	return &ast.ReturnStmt{
		Range:  token.NewRange(Return, p.previous()),
		Func:   p.currentFunction,
		Return: Return,
		Value:  expr,
	}
}

func (p *Parser) voidReturn() ast.Statement {
	Leave := p.previous()
	p.consumeN(token.DIE, token.FUNKTION, token.DOT)
	return &ast.ReturnStmt{
		Range:  token.NewRange(Leave, p.previous()),
		Func:   p.currentFunction,
		Return: Leave,
		Value:  nil,
	}
}

func (p *Parser) blockStatement(symbols *ast.SymbolTable) ast.Statement {
	colon := p.previous()
	if p.peek().Line() <= colon.Line() {
		p.err(p.peek(), p.peek().Range, "Nach einem Doppelpunkt muss eine neue Zeile beginnen")
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
		Colon:      colon,
		Statements: statements,
		Symbols:    symbols,
	}
}

func (p *Parser) expressionStatement() ast.Statement {
	return p.finishStatement(&ast.ExprStmt{Expr: p.expression()})
}

// entry for expression parsing
func (p *Parser) expression() ast.Expression {
	return p.boolOR()
}

func (p *Parser) boolOR() ast.Expression {
	expr := p.boolAND()
	for p.match(token.ODER) {
		operator := p.previous()
		rhs := p.boolAND()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) boolAND() ast.Expression {
	expr := p.bitwiseOR()
	for p.match(token.UND) {
		operator := p.previous()
		rhs := p.bitwiseOR()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) bitwiseOR() ast.Expression {
	expr := p.bitwiseXOR()
	for p.matchN(token.LOGISCH, token.ODER) {
		operator := p.previous()
		operator.Type = token.LOGISCHODER
		rhs := p.bitwiseXOR()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) bitwiseXOR() ast.Expression {
	expr := p.bitwiseAND()
	for p.matchN(token.LOGISCH, token.KONTRA) {
		operator := p.previous()
		rhs := p.bitwiseAND()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) bitwiseAND() ast.Expression {
	expr := p.equality()
	for p.matchN(token.LOGISCH, token.UND) {
		operator := p.previous()
		operator.Type = token.LOGISCHUND
		rhs := p.equality()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) equality() ast.Expression {
	expr := p.comparison()
	for p.match(token.GLEICH, token.UNGLEICH) {
		operator := p.previous()
		rhs := p.comparison()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
		p.consume(token.IST)
	}
	return expr
}

func (p *Parser) comparison() ast.Expression {
	expr := p.bitShift()
	for p.match(token.GRÖßER, token.KLEINER) {
		operator := p.previous()
		p.consume(token.ALS)
		if p.match(token.COMMA) {
			p.consume(token.ODER)
			if operator.Type == token.GRÖßER {
				operator.Type = token.GRÖßERODER
			} else {
				operator.Type = token.KLEINERODER
			}
		}

		rhs := p.bitShift()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
		p.consume(token.IST)
	}
	return expr
}

func (p *Parser) bitShift() ast.Expression {
	expr := p.term()
	for p.match(token.UM) {
		rhs := p.term()
		p.consumeN(token.BIT, token.NACH)
		if !p.match(token.LINKS, token.RECHTS) {
			msg := fmt.Sprintf("Es wurde 'LINKS' oder 'RECHTS' erwartet aber '%s' gefunden", p.previous().Literal)
			p.err(p.previous(), p.previous().Range, msg)
			return &ast.BadExpr{
				Err: &ParserError{
					rang: token.Range{
						Start: expr.GetRange().Start,
						End:   token.NewEndPos(p.peek()),
					},
					file: expr.Token().File,
					msg:  msg,
				},
				Tok: expr.Token(),
			}
		}
		operator := p.previous()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
		p.consume(token.VERSCHOBEN)
	}
	return expr
}

func (p *Parser) term() ast.Expression {
	expr := p.factor()
	for p.match(token.PLUS, token.MINUS, token.VERKETTET) {
		operator := p.previous()
		if operator.Type == token.VERKETTET { // string concatenation
			p.consume(token.MIT)
		}
		rhs := p.factor()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) factor() ast.Expression {
	expr := p.unary()
	for p.match(token.MAL, token.DURCH, token.MODULO) {
		operator := p.previous()
		rhs := p.unary()
		expr = &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) unary() ast.Expression {
	if expr := p.funcCall(); expr != nil { // first check for a function call to enable operator overloading
		return p.power(expr)
	}
	var start token.Token
	// match the correct unary operator
	if p.match(token.NICHT, token.BETRAG, token.DIE, token.GRÖßE, token.LÄNGE, token.DER, token.LOGISCH) {
		if p.previous().Type == token.DIE {
			start = p.previous()
			if !p.match(token.GRÖßE, token.LÄNGE) {
				p.decrease() // DIE does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		} else if p.previous().Type == token.DER {
			start = p.previous()
			if !p.match(token.BETRAG) {
				p.decrease() // DER does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		} else if p.previous().Type == token.LOGISCH {
			start = p.previous()
			if !p.match(token.NICHT) {
				p.decrease() // LOGISCH does not belong to a operator, so maybe it is a function call
				return p.negate()
			}
		} else { // error handling
			switch p.previous().Type {
			case token.GRÖßE, token.LÄNGE:
				p.err(p.previous(), p.previous().Range, fmt.Sprintf("Vor '%s' muss 'DIE' stehen", p.previous()))
			case token.BETRAG:
				p.err(p.previous(), p.previous().Range, "Vor 'BETRAG' muss 'DER' stehen")
			}
			start = p.previous()
		}
		operator := p.previous()
		switch operator.Type {
		case token.BETRAG, token.GRÖßE, token.LÄNGE:
			p.consume(token.VON)
		case token.NICHT:
			if p.peekN(-2).Type == token.LOGISCH {
				operator.Type = token.LOGISCHNICHT
			}
		}
		rhs := p.unary()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(start),
				End:   rhs.GetRange().End,
			},
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return p.negate()
}

func (p *Parser) negate() ast.Expression {
	if p.match(token.NEGATE) {
		op := p.previous()
		rhs := p.unary()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(op),
				End:   rhs.GetRange().End,
			},
			Operator: op,
			Rhs:      rhs,
		}
	}
	return p.power(nil)
}

// when called from unary() lhs might be a funcCall
// TODO: check precedence
func (p *Parser) power(lhs ast.Expression) ast.Expression {
	if p.match(token.DIE) {
		lhs := p.unary()
		p.consumeN(token.DOT, token.WURZEL)
		operator := p.previous()
		operator.Type = token.HOCH
		op2 := operator
		op2.Type = token.DURCH
		p.consume(token.VON)
		// root is implemented as pow(degree, 1/radicant)
		expr := p.unary()

		return &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   lhs.GetRange().End,
			},
			Lhs:      expr,
			Operator: operator,
			Rhs: &ast.BinaryExpr{
				Lhs: &ast.IntLit{
					Literal: lhs.Token(),
					Value:   1,
				},
				Operator: op2,
				Rhs:      lhs,
			},
		}
	}

	if p.matchN(token.DER, token.LOGARITHMUS) {
		operator := p.previous()
		p.consume(token.VON)
		numerus := p.expression()
		p.consumeN(token.ZUR, token.BASIS)
		rhs := p.unary()

		return &ast.BinaryExpr{
			Range: token.Range{
				Start: numerus.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      numerus,
			Operator: operator,
			Rhs:      rhs,
		}
	}

	lhs = p.primary(lhs)
	for p.match(token.HOCH) {
		operator := p.previous()
		rhs := p.unary()
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      lhs,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return lhs
}

// when called from power() lhs might be a funcCall
func (p *Parser) primary(lhs ast.Expression) ast.Expression {
	if lhs == nil {
		lhs = p.funcCall()
	}
	if lhs == nil { // funccall has the highest precedence (aliases + operator overloading)
		switch tok := p.advance(); tok.Type {
		case token.FALSE:
			lhs = &ast.BoolLit{Literal: p.previous(), Value: false}
		case token.TRUE:
			lhs = &ast.BoolLit{Literal: p.previous(), Value: true}
		case token.PI:
			lhs = &ast.FloatLit{Literal: p.previous(), Value: 3.141592654}
		case token.E:
			lhs = &ast.FloatLit{Literal: p.previous(), Value: 2.718281828}
		case token.TAU:
			lhs = &ast.FloatLit{Literal: p.previous(), Value: 6.283185307}
		case token.PHI:
			lhs = &ast.FloatLit{Literal: p.previous(), Value: 1.618033989}
		case token.INT:
			lhs = p.parseIntLit()
		case token.FLOAT:
			lit := p.previous()
			if val, err := strconv.ParseFloat(strings.Replace(lit.Literal, ",", ".", 1), 64); err == nil {
				lhs = &ast.FloatLit{Literal: lit, Value: val}
			} else {
				p.err(lit, lit.Range, "Das Zahlen Literal kann nicht gelesen werden")
				lhs = &ast.FloatLit{Literal: lit, Value: 0}
			}
		case token.CHAR:
			lit := p.previous()
			lhs = &ast.CharLit{Literal: lit, Value: p.parseChar(lit.Literal)}
		case token.STRING:
			lit := p.previous()
			lhs = &ast.StringLit{Literal: lit, Value: p.parseString(lit.Literal)}
		case token.LPAREN:
			lhs = p.grouping()
		case token.IDENTIFIER:
			lhs = &ast.Ident{
				Literal: p.previous(),
			}
		case token.EINE, token.EINER: // list literals
			begin := p.previous()
			if begin.Type == token.EINER && p.match(token.LEEREN) {
				typ := p.parseListType()
				lhs = &ast.ListLit{
					Tok:    begin,
					Range:  token.NewRange(begin, p.previous()),
					Type:   typ,
					Values: nil,
				}
			} else if p.match(token.LEERE) {
				typ := p.parseListType()
				lhs = &ast.ListLit{
					Tok:    begin,
					Range:  token.NewRange(begin, p.previous()),
					Type:   typ,
					Values: nil,
				}
			} else {
				p.consumeN(token.LISTE, token.COMMA, token.DIE, token.AUS)
				values := append(make([]ast.Expression, 0, 2), p.expression())
				for p.match(token.COMMA) {
					values = append(values, p.expression())
				}
				p.consume(token.BESTEHT)
				lhs = &ast.ListLit{
					Tok:    begin,
					Range:  token.NewRange(begin, p.previous()),
					Values: values,
				}
			}
		default:
			msg := fmt.Sprintf("Es wurde ein Literal oder ein Name erwartet aber '%s' gefunden", p.previous().Literal)
			p.err(p.previous(), p.previous().Range, msg)
			lhs = &ast.BadExpr{
				Err: &ParserError{
					rang: tok.Range,
					file: tok.File,
					msg:  msg,
				},
				Tok: tok,
			}
		}
	}

	// indexing
	if p.match(token.AN) {
		p.consumeN(token.DER, token.STELLE)
		operator := p.previous()
		rhs := p.primary(nil)
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      lhs,
			Operator: operator,
			Rhs:      rhs,
		}
	} else if p.match(token.VON) {
		operator := p.previous()
		operator.Type = token.VONBIS
		operand := lhs
		mid := p.expression()
		p.consume(token.BIS)
		rhs := p.primary(nil)
		lhs = &ast.TernaryExpr{
			Range: token.Range{
				Start: operand.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Lhs:      operand,
			Mid:      mid,
			Rhs:      rhs,
			Operator: operator,
		}
	}

	// type-casting
	if p.match(token.ALS) { // TODO: enable nested casting
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

// either ast.Ident or ast.Indexing
// p.previous() must be of Type token.IDENTIFIER
func (p *Parser) assigneable() ast.Assigneable {
	var ass ast.Assigneable = &ast.Ident{
		Literal: p.previous(),
	}

	for p.match(token.AN) {
		p.consumeN(token.DER, token.STELLE)
		index := p.unary() // TODO: check if this can stay p.expression or if p.unary is better
		ass = &ast.Indexing{
			Lhs:   ass,
			Index: index,
		}
		if !p.match(token.COMMA) {
			break
		}
	}
	return ass
}

func (p *Parser) grouping() ast.Expression {
	lParen := p.previous()
	innerExpr := p.expression()
	p.consume(token.RPAREN)

	return &ast.Grouping{
		Range:  token.NewRange(lParen, p.previous()),
		LParen: lParen,
		Expr:   innerExpr,
	}
}

func (p *Parser) funcCall() ast.Expression {
	// stores an alias with the actual length of all tokens (expanded token.ALIAS_EXPRESSIONs)
	type matchedAlias struct {
		alias        *ast.FuncAlias // original alias
		actualLength int            // length of this occurence in the code (considers the token length of the passed arguments)
	}

	start := p.cur                            // save start position to restore the state if no alias was recognized
	matchedAliases := make([]matchedAlias, 0) // stors all matched aliases (can be multiple due to different lengths)

	// loop through all possible aliases (expensive, might change later)
outer:
	for i, l := 0, len(p.funcAliases); i < l; i++ {
		alias := &p.funcAliases[i]
		p.cur = start

		// loop through all the tokens in the alias
		for ii, ll := 0, len(alias.Tokens); ii < ll && alias.Tokens[ii].Type != token.EOF; ii++ {
			tok := &alias.Tokens[ii]

			// expand arguments
			if tok.Type == token.ALIAS_PARAMETER {
				switch t := p.peek(); t.Type {
				case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER,
					token.PI, token.E, token.TAU, token.PHI:
					p.advance() // single-token so skip it
					continue
				case token.NEGATE:
					p.advance()
					if !p.match(token.INT, token.FLOAT, token.PI, token.E, token.TAU, token.PHI, token.IDENTIFIER) {
						p.cur = start
						continue outer
					}
					continue
				case token.LPAREN:
					p.advance()
					numLparens := 1 // used to enable groupings and multi-token expressions inside the argument
					for numLparens > 0 && !p.atEnd() {
						switch p.advance().Type {
						case token.LPAREN:
							numLparens++
						case token.RPAREN:
							numLparens--
						}
					}
					if p.atEnd() {
						p.cur = start
						continue outer
					}
					continue
				}
			}

			// validate that the alias matches
			if !tokenEqual(p.peek(), *tok) {
				p.cur = start
				continue outer // try the next alias otherwise
			}
			p.advance()
		}

		// alias was matched so append it to the list of possible aliases
		matchedAliases = append(matchedAliases, matchedAlias{alias: alias, actualLength: p.cur - start})
	}

	if len(matchedAliases) == 0 { // check if any alias was matched
		return nil // no alias -> no function call
	}

	// attempts to evaluate the arguments for the passed alias and checks if types match
	// returns nil if argument and parameter types don't match
	// similar to the alogrithm above
	checkAlias := func(mAlias *matchedAlias, typeSensitive bool) map[string]ast.Expression {
		p.cur = start
		args := map[string]ast.Expression{}

		for i, l := 0, len(mAlias.alias.Tokens); i < l && mAlias.alias.Tokens[i].Type != token.EOF; i++ {
			tok := &mAlias.alias.Tokens[i]

			if tok.Type == token.ALIAS_PARAMETER {
				argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
				paramType := mAlias.alias.Args[argName]    // type of the current parameter

				pType := p.peek().Type
				// early return if a non-identifier expression is passed as reference
				if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
					return nil
				}

				exprStart := p.cur
				switch pType {
				case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER,
					token.PI, token.E, token.TAU, token.PHI:
					p.advance() // single-token argument
				case token.NEGATE:
					p.advance()
					p.match(token.INT, token.FLOAT, token.PI, token.E, token.TAU, token.PHI, token.IDENTIFIER)
				case token.LPAREN: // multiple-token arguments must be wrapped in parentheses
					p.advance()
					numLparens := 1
					for numLparens > 0 && !p.atEnd() {
						switch p.advance().Type {
						case token.LPAREN:
							numLparens++
						case token.RPAREN:
							numLparens--
						}
					}
				}
				tokens := make([]token.Token, p.cur-exprStart)
				copy(tokens, p.tokens[exprStart:p.cur]) // copy all the tokens of the expression to be able to append the EOF
				// append the EOF needed for the parser
				eof := token.Token{Type: token.EOF, Literal: "", Indent: 0, File: tok.File, Range: tok.Range, AliasInfo: nil}
				tokens = append(tokens, eof)
				argParser := New(tokens, p.errorHandler) // create a new parser for this expression
				argParser.funcAliases = p.funcAliases    // it needs the functions aliases
				argParser.resolver = p.resolver
				argParser.typechecker = p.typechecker
				var arg ast.Expression
				if paramType.IsReference {
					if tokens[0].Type == token.LPAREN {
						tokens = append(tokens[1:len(tokens)-2], eof)
						argParser.tokens = tokens
					}
					argParser.advance() // consume the identifier for assigneable() to work
					arg = argParser.assigneable()
				} else {
					arg = argParser.expression() // parse the argument
				}

				// check if the argument type matches the prameter type

				// we are in the for loop below, so the types must match
				// otherwise it doesn't matter
				if typeSensitive {
					typecheckErrored := p.typechecker.Errored
					p.typechecker.ErrorHandler = ddperror.EmptyHandler // silence errors
					typ := p.typechecker.Evaluate(arg)

					if typ != paramType.Type {
						arg = nil // arg and param types don't match
					} else if                                                     // string-indexings may not be passed as char-reference
					ass, ok := arg.(*ast.Indexing);                               // evaluate the argunemt
					paramType.IsReference && paramType.Type == ddptypes.Char() && // if the parameter is a char-reference
						ok { // and the argument is a indexing
						lhs := p.typechecker.Evaluate(ass.Lhs)
						if lhs.Primitive == ddptypes.TEXT { // check if the lhs is a string
							arg = nil
						}
					}

					p.typechecker.ErrorHandler = p.errorHandler // turn errors on again
					p.typechecker.Errored = typecheckErrored

					if arg == nil {
						return nil
					}
				}

				args[argName] = arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}
		p.cur = start + mAlias.actualLength
		return args
	}

	// sort the aliases in descending order
	// Stable so equal aliases stay in the order they were defined
	sort.SliceStable(matchedAliases, func(i, j int) bool {
		return len(matchedAliases[i].alias.Tokens) > len(matchedAliases[j].alias.Tokens)
	})

	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		if args := checkAlias(&matchedAliases[i], true); args != nil {
			return &ast.FuncCall{
				Range: token.NewRange(p.tokens[start], p.previous()),
				Tok:   p.tokens[start],
				Name:  matchedAliases[i].alias.Func,
				Args:  args,
			}
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments
	mostFitting := &matchedAliases[0]
	args := checkAlias(mostFitting, false)

	return &ast.FuncCall{
		Range: token.NewRange(p.tokens[start], p.previous()),
		Tok:   p.tokens[start],
		Name:  mostFitting.alias.Func,
		Args:  args,
	}
}

/*** Helper functions ***/

// helper to parse ddp chars with escape sequences
func (p *Parser) parseChar(s string) (r rune) {
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
			p.err(p.previous(), p.previous().Range, "Ungültige Escape Sequenz '\\%s' im Buchstaben Literal", string(r))
		}
		return r
	}
	p.err(p.previous(), p.previous().Range, "Invalides Buchstaben Literal")
	return -1
}

// helper to parse ddp strings with escape sequences
func (p *Parser) parseString(s string) string {
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
				p.err(p.previous(), p.previous().Range, "Ungültige Escape Sequenz '\\%s' im Text Literal", string(seq))
				continue
			}

			str = str[:i] + string(seq) + str[i+w+w2:]
		}
	}

	return str
}

func (p *Parser) parseIntLit() *ast.IntLit {
	lit := p.previous()
	if val, err := strconv.ParseInt(lit.Literal, 10, 64); err == nil {
		return &ast.IntLit{Literal: lit, Value: val}
	} else {
		p.err(lit, lit.Range, "Das Zahlen Literal '%s' kann nicht gelesen werden", lit.Literal)
		return &ast.IntLit{Literal: lit, Value: 0}
	}
}

// converts a TokenType to a Type
func tokenTypeToType(t token.TokenType) ddptypes.Type {
	switch t {
	case token.NICHTS:
		return ddptypes.Void()
	case token.ZAHL:
		return ddptypes.Int()
	case token.KOMMAZAHL:
		return ddptypes.Float()
	case token.BOOLEAN:
		return ddptypes.Bool()
	case token.BUCHSTABE:
		return ddptypes.Char()
	case token.TEXT:
		return ddptypes.String()
	}
	panic(fmt.Sprintf("invalid TokenType (%d)", t))
}

// parses tokens into a DDPType
// expects the next token to be the start of the type
// returns ILLEGAL and errors if no typename was found
func (p *Parser) parseType() ddptypes.Type {
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(p.peek(), p.peek().Range, "Es wurde ein Typname erwartet aber '%s' gefunden", p.peek().Literal)
		return ddptypes.Illegal() // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return tokenTypeToType(p.previous().Type)
	case token.BOOLEAN, token.TEXT:
		if !p.match(token.LISTE) {
			return tokenTypeToType(p.previous().Type)
		}
		return ddptypes.NewList(tokenTypeToType(p.peekN(-2).Type).Primitive)
	case token.ZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.ZAHL)
	case token.KOMMAZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.KOMMAZAHL)
	case token.BUCHSTABEN:
		if p.peekN(-2).Type == token.EINEN || p.peekN(-2).Type == token.JEDEN { // edge case in function return types and for-range loops
			return ddptypes.Char()
		}
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.BUCHSTABE)
	}

	return ddptypes.Illegal() // unreachable
}

// parses tokens into a DDPType which must be a list type
// expects the next token to be the start of the type
// returns ILLEGAL and errors if no typename was found
func (p *Parser) parseListType() ddptypes.Type {
	if !p.match(token.BOOLEAN, token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(p.peek(), p.peek().Range, "Es wurde ein Listen-Typname erwartet aber '%s' gefunden", p.peek().Literal)
		return ddptypes.Illegal() // void indicates error
	}

	switch p.previous().Type {
	case token.BOOLEAN, token.TEXT:
		p.consume(token.LISTE)
		return ddptypes.NewList(tokenTypeToType(p.peekN(-2).Type).Primitive)
	case token.ZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.ZAHL)
	case token.KOMMAZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.KOMMAZAHL)
	case token.BUCHSTABEN:
		p.consume(token.LISTE)
		return ddptypes.NewList(ddptypes.BUCHSTABE)
	}

	return ddptypes.Illegal() // unreachable
}

// parses tokens into a DDPType and returns wether the type is a reference type
// expects the next token to be the start of the type
// returns ILLEGAL and errors if no typename was found
func (p *Parser) parseReferenceType() (ddptypes.Type, bool) {
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(p.peek(), p.peek().Range, "Es wurde ein Typname erwartet aber '%s' gefunden", p.peek().Literal)
		return ddptypes.Illegal(), false // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return tokenTypeToType(p.previous().Type), false
	case token.BOOLEAN, token.TEXT:
		if p.match(token.LISTE) {
			return ddptypes.NewList(tokenTypeToType(p.peekN(-2).Type).Primitive), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.NewList(tokenTypeToType(p.peekN(-3).Type).Primitive), true
		} else if p.match(token.REFERENZ) {
			return ddptypes.NewPrimitive(tokenTypeToType(p.peekN(-2).Type).Primitive), true
		}
		return tokenTypeToType(p.previous().Type), false
	case token.ZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.NewList(ddptypes.ZAHL), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.NewList(ddptypes.ZAHL), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Int(), true
	case token.KOMMAZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.NewList(ddptypes.KOMMAZAHL), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.NewList(ddptypes.KOMMAZAHL), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Float(), true
	case token.BUCHSTABEN:
		if p.match(token.LISTE) {
			return ddptypes.NewList(ddptypes.BUCHSTABE), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.NewList(ddptypes.BUCHSTABE), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Char(), true
	}

	return ddptypes.Illegal(), false // unreachable
}

// parses tokens into a DDPType
// unlike parseType it may return void
// the error return is ILLEGAL
func (p *Parser) parseTypeOrVoid() ddptypes.Type {
	if p.match(token.NICHTS) {
		return ddptypes.Void()
	}
	return p.parseType()
}

// create a sub-scope of the current scope
func (p *Parser) newScope() *ast.SymbolTable {
	return ast.NewSymbolTable(p.resolver.CurrentTable)
}

// set the current scope for the resolver and typechecker
func (p *Parser) setScope(symbols *ast.SymbolTable) {
	p.resolver.CurrentTable, p.typechecker.CurrentTable = symbols, symbols
}

// exit the current scope of the resolver and typechecker
func (p *Parser) exitScope() {
	p.resolver.CurrentTable, p.typechecker.CurrentTable = p.resolver.CurrentTable.Enclosing, p.typechecker.CurrentTable.Enclosing
}

// if the current tokenType is contained in types, advance
// returns wether we advanced or not
func (p *Parser) match(types ...token.TokenType) bool {
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
func (p *Parser) matchN(types ...token.TokenType) bool {
	for i, t := range types {
		if p.peekN(i).Type != t {
			return false
		}
	}

	for i := range types {
		_ = i
		p.advance()
	}

	return true
}

// if the current token is of type t advance, otherwise error
func (p *Parser) consume(t token.TokenType) bool {
	if p.check(t) {
		p.advance()
		return true
	}

	p.err(p.peek(), p.peek().Range, "Es wurde '%s' erwartet aber '%s' gefunden", t, p.peek().Literal)
	return false
}

// consume a series of tokens
func (p *Parser) consumeN(t ...token.TokenType) bool {
	for _, v := range t {
		if !p.consume(v) {
			return false
		}
	}
	return true
}

// same as consume but tolerates multiple tokenTypes
func (p *Parser) consumeAny(tokenTypes ...token.TokenType) bool {
	for _, v := range tokenTypes {
		if p.check(v) {
			p.advance()
			return true
		}
	}

	msg := "Es wurde "
	for i, v := range tokenTypes {
		if i >= len(tokenTypes)-1 {
			break
		}
		msg += fmt.Sprintf("'%s', ", v)
	}
	msg += fmt.Sprintf("oder '%s' erwartet aber '%s' gefunden", tokenTypes[len(tokenTypes)-1], p.peek().Literal)

	p.err(p.peek(), p.peek().Range, msg)
	return false
}

// helper to report errors and enter panic mode
func (p *Parser) err(token token.Token, rnge token.Range, msg string, args ...any) {
	if !p.panicMode {
		p.panicMode = true
		p.errorHandler(&ParserError{
			file: token.File,
			rang: rnge,
			msg:  fmt.Sprintf(msg, args...),
		})
	}
}

// check if the current token is of type t without advancing
func (p *Parser) check(t token.TokenType) bool {
	if p.atEnd() {
		return false
	}
	return p.peek().Type == t
}

// check if the current token is EOF
func (p *Parser) atEnd() bool {
	return p.peek().Type == token.EOF
}

// return the current token and advance p.cur
func (p *Parser) advance() token.Token {
	if !p.atEnd() {
		p.cur++
		return p.previous()
	}
	return p.peek() // return EOF
}

// returns the current token without advancing
func (p *Parser) peek() token.Token {
	return p.tokens[p.cur]
}

// returns the n'th token starting from current without advancing
func (p *Parser) peekN(n int) token.Token {
	if p.cur+n >= len(p.tokens) || p.cur+n < 0 {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[p.cur+n]
}

// returns the token before peek()
func (p *Parser) previous() token.Token {
	if p.cur < 1 {
		return token.Token{Type: token.ILLEGAL}
	}
	return p.tokens[p.cur-1]
}

// opposite of advance
func (p *Parser) decrease() {
	if p.cur > 0 {
		p.cur--
	}
}

// retrives the last comment which comes before pos
// if their are no comments before pos nil is returned
func (p *Parser) commentBeforePos(pos token.Position, file string) (result *token.Token) {
	if len(p.comments) == 0 {
		return nil
	}

	for i := range p.comments {
		if p.comments[i].File != file {
			continue
		}
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
func (p *Parser) commentAfterPos(pos token.Position, file string) (result *token.Token) {
	if len(p.comments) == 0 {
		return nil
	}

	for i := range p.comments {
		if p.comments[i].File != file {
			continue
		}
		if p.comments[i].Range.End.IsBehind(pos) {
			return &p.comments[i]
		}
	}
	return result
}

// retreives a leading or trailing comment of p.previous()
// prefers leading comments
// may return nil
func (p *Parser) getLeadingOrTrailingComment() (result *token.Token) {
	tok := p.previous()
	comment := p.commentBeforePos(tok.Range.Start, tok.File)
	// the comment must be between the identifier and the last token of the type
	if comment != nil && !comment.Range.Start.IsBehind(p.peekN(-2).Range.End) {
		comment = nil
	}
	// a trailing comment must be the next token after the identifier
	if trailingComment := p.commentAfterPos(tok.Range.End, tok.File); comment == nil && trailingComment != nil &&
		trailingComment.Range.End.IsBefore(p.peek().Range.Start) {
		comment = trailingComment
	}

	return comment
}

// check if a slice of tokens contains a literal
func containsLiteral(tokens []token.Token, literal string) bool {
	for _, v := range tokens {
		if v.Literal == literal {
			return true
		}
	}
	return false
}

// check if two tokens are equal
func tokenEqual(t1 token.Token, t2 token.Token) bool {
	if t1.Type != t2.Type {
		return false
	}

	switch t1.Type {
	case token.IDENTIFIER:
		return t1.Literal == t2.Literal
	case token.ALIAS_PARAMETER:
		return *t1.AliasInfo == *t2.AliasInfo
	case token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal == t2.Literal
	}

	return true
}

// counts all elements in the slice which fulfill the provided test function
func countElements[T any](elements []T, test func(T) bool) (count int) {
	for _, v := range elements {
		if test(v) {
			count++
		}
	}
	return count
}

// checks wether two slices are equal using the provided comparison function
func slicesEqual[T any](s1 []T, s2 []T, equal func(T, T) bool) bool {
	if len(s1) != len(s2) {
		return false
	}

	for i := range s1 {
		if !equal(s1[i], s2[i]) {
			return false
		}
	}

	return true
}
