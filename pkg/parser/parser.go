package parser

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/pkg/ast"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast/resolver"
	"github.com/DDP-Projekt/Kompilierer/pkg/ast/typechecker"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

//go:embed inbuilt
var inbuilt embed.FS

// inbuilt functions are prefixed with § (e.g. §Schreibe_Zahl), that's also how the compiler/interpreter recognizes them as inbuilt
// the variables below hold the declarations of inbuilt functions and variables
// they are initialized in the init() func of this package (called when the package is imported)
var inbuiltSymbolTable *ast.SymbolTable         // stores inbuilt Symbols (func names and constants)
var inbuiltAliases []funcAlias                  // stores inbuilt aliases
var inbuiltTypechecker *typechecker.Typechecker // stores the types of inbuilt function arguments
var inbuiltDecls []*ast.DeclStmt                // stores inbuilt declarations to be evalueated first later on (you could compare it to the go runtime which is compiled into the binaries every time)
var initializing bool                           // flag if we are currently running the init() function

// load inbuilt functions
func init() {
	initializing = true
	errored := false
	// helper to set the errored flag on error
	Err := func(msg string) {
		errored = true
		fmt.Println(msg)
	}

	inbuiltSymbolTable = ast.NewSymbolTable(nil) // global SymbolTable
	inbuiltAliases = make([]funcAlias, 0)
	inbuiltTypechecker = typechecker.New(inbuiltSymbolTable, Err) // needs the global inbuiltSymbolTable to determine inbuilt function argument types
	inbuiltDecls = make([]*ast.DeclStmt, 0)

	// walk every .ddp file in the inbuilt directory and add its Declarations to the global state
	fs.WalkDir(inbuilt, ".", func(path string, entry fs.DirEntry, err error) error {
		if !entry.IsDir() {
			if filepath.Ext(path) == ".ddp" {
				file, err := inbuilt.ReadFile(path) // read the file
				if err != nil {
					Err(err.Error())
					return err
				}
				tokens, err := scanner.ScanSource(path, file, Err, scanner.ModeInitializing) // scan the file with the ModeInitializing flag to scan § correctly
				if err != nil {
					Err(err.Error())
					return err
				}
				parser := New(tokens, Err) // create the parser for this file
				Ast := parser.Parse()      // parse the file
				if Ast.Faulty {
					Err(err.Error())
					return err
				}
				// append all inbuilt function and variable declarations to the inbuildDecls
				// they are compiled into every ddp executable
				for _, stmt := range Ast.Statements {
					if decl, ok := stmt.(*ast.DeclStmt); ok {
						inbuiltDecls = append(inbuiltDecls, decl)
					}
				}
				inbuiltAliases = append(inbuiltAliases, parser.funcAliases...) // append the function aliases
				inbuiltSymbolTable.Merge(Ast.Symbols)                          // add the symbols (variable and function names) to the inbuildSymbolTable
			}
		}
		return nil
	})

	if errored {
		panic(errors.New("unable to load inbuilts")) // inbuilts MUST build
	}
	initializing = false
}

// wrapper for an alias
type funcAlias struct {
	Tokens []token.Token // tokens of the alias
	Func   string        // the function it refers to
}

// holds state when parsing a .ddp file into an AST
type Parser struct {
	tokens       []token.Token        // the tokens to parse
	cur          int                  // index of the current token
	errorHandler scanner.ErrorHandler // a function to which errors are passed

	funcAliases     []funcAlias              // all found aliases (+ inbuild aliases)
	currentFunction string                   // function which is currently being parsed
	panicMode       bool                     // flag to not report following errors
	errored         bool                     // wether the parser found an error
	resolver        *resolver.Resolver       // used to resolve every node directly after it has been parsed
	typechecker     *typechecker.Typechecker // used to typecheck every node directly after it has been parsed
}

// returns a new parser, ready to parse the provided tokens
func New(tokens []token.Token, errorHandler scanner.ErrorHandler) *Parser {
	if errorHandler == nil { // default error handler does nothing
		errorHandler = func(string) {}
	}
	if len(tokens) == 0 {
		tokens = []token.Token{{Type: token.EOF}} // we need at least one EOF at the end of the tokens slice
	}
	if tokens[len(tokens)-1].Type != token.EOF { // the last token must be EOF
		tokens = append(tokens, token.Token{Type: token.EOF})
	}
	aliases := make([]funcAlias, len(inbuiltAliases))
	// we don't want to change the inbuilt aliases incase we are in initializing mode,
	// so we copy them
	copy(aliases, inbuiltAliases)
	p := &Parser{
		tokens:       tokens,
		cur:          0,
		errorHandler: errorHandler,
		funcAliases:  aliases,
		panicMode:    false,
		errored:      false,
		resolver:     &resolver.Resolver{},
		typechecker:  &typechecker.Typechecker{},
	}
	return p
}

// parse the provided tokens into an Ast
func (p *Parser) Parse() *ast.Ast {
	Ast := &ast.Ast{
		Statements: make([]ast.Statement, 0, len(inbuiltDecls)), // every AST has at least the inbuild function and variable declarations
		Symbols:    inbuiltSymbolTable.Copy(),                   // every AST has at least the inbuild function and variable symbols
		Faulty:     false,
		File:       p.tokens[0].File,
	}

	if !initializing { // don't append inbuilt stuff if this is just a temporary ast during initialization
		for _, v := range inbuiltDecls {
			Ast.Statements = append(Ast.Statements, v)
		}
	}
	// prepare the resolver and typechecker with the inbuild symbols and types
	p.resolver = resolver.New(Ast, p.errorHandler)
	*p.typechecker = *inbuiltTypechecker // the typechecker needs the funcArgs field from the inbuilt SymbolTable
	p.typechecker.CurrentTable = Ast.Symbols

	// main parsing loop
	for !p.atEnd() {
		stmt := p.declaration()                       // parse the node
		p.resolver.ResolveNode(stmt)                  // resolve symbols in it (variables, functions, ...)
		p.typechecker.TypecheckNode(stmt)             // typecheck the node
		Ast.Statements = append(Ast.Statements, stmt) // add it to the ast
		if p.panicMode {                              // synchronize the parsing if we are in panic mode
			p.synchronize()
		}
	}

	// if any error occured, the AST is faulty
	if p.errored || p.resolver.Errored() || p.typechecker.Errored() {
		Ast.Faulty = true
	}

	return Ast
}

// if an error was encountered we synchronize to a point where correct parsing is possible again
func (p *Parser) synchronize() {
	p.panicMode = false
	p.errored = true

	p.advance()
	for !p.atEnd() {
		if p.previous().Type == token.DOT { // a . ends statements, so we can continue parsing
			return
		}
		// these tokens typically begin statements which begin a new node
		switch p.peek().Type {
		case token.DER, token.DIE, token.WENN, token.FÜR, token.GIB, token.SOLANGE, token.COLON:
			return
		}
		p.advance()
	}
}

// entry point for the recursive descent parsing
func (p *Parser) declaration() ast.Statement {
	if p.match(token.DER, token.DIE) { // might indicate a function or variable declaration
		switch p.previous().Type {
		case token.DER:
			switch p.peek().Type {
			case token.BETRAG:
				p.decrease() // decrease, so expressionStatement() can recognize it as operator
			case token.BOOLEAN, token.TEXT, token.BUCHSTABE:
				p.match(token.BOOLEAN, token.TEXT, token.BUCHSTABE) // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration()}      // parse the declaration
			}
		case token.DIE:
			switch p.peek().Type {
			case token.GRÖßE, token.LÄNGE:
				p.decrease() // decrease, so expressionStatement() can recognize it as operator
			case token.ZAHL, token.KOMMAZAHL:
				p.match(token.ZAHL, token.KOMMAZAHL)           // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration()} // parse the declaration
			case token.FUNKTION:
				p.match(token.FUNKTION)
				return &ast.DeclStmt{Decl: p.funcDeclaration()} // parse the function declaration
			}
		}
	}

	return p.statement() // no declaration, so it must be a statement
}

// helper for boolean assignments
func (p *Parser) assignRhs() ast.Expression {
	var expr ast.Expression               // the final expression
	if p.match(token.TRUE, token.FALSE) { // parse possible wahr/falsch wenn syntax
		if p.match(token.WENN) {
			if tok := p.tokens[p.cur-2]; tok.Type == token.FALSE { // if it is false, we add a unary bool-negate into the ast
				tok.Type = token.NICHT
				expr = &ast.UnaryExpr{
					Operator: tok,
					Rhs:      p.expression(), // the actual boolean expression after falsch wenn, which is negated
				}
			} else {
				expr = p.expression() // wahr wenn simply becomes a normal expression
			}
			p.consume(token.IST) // ist, after wahr/falsch wenn for grammar
		} else { // no wahr/falsch wenn, only a boolean literal
			p.decrease() // decrease, so expression() can recognize the literal
			expr = p.expression()
			if _, ok := expr.(*ast.BoolLit); !ok { // validate that nothing follows after the literal
				p.err(expr.Token(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
			}
		}
	} else {
		expr = p.expression() // no wahr/falsch, so a normal expression
	}
	return expr
}

func (p *Parser) varDeclaration() ast.Declaration {
	typ := p.previous()               // ZAHL, KOMMAZAHL, etc.
	if !p.consume(token.IDENTIFIER) { // we need a name, so bailout if none is provided
		return &ast.BadDecl{Tok: p.peek()}
	}
	name := p.previous()
	p.consume(token.IST)
	var expr ast.Expression
	if typ.Type == token.BOOLEAN {
		expr = p.assignRhs() // handle booleans seperately (wahr/falsch wenn)
	} else {
		expr = p.expression()
	}
	p.consume(token.DOT)
	return &ast.VarDecl{
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
	Funktion := p.previous()          // save the token
	if !p.consume(token.IDENTIFIER) { // we need a name, so bailout if none is provided
		return &ast.BadDecl{Tok: p.peek()}
	}
	name := p.previous()

	// parse the parameter declaration
	// parameter names and types are declared seperately
	var paramNames []token.Token = nil
	var paramTypes []token.Token = nil
	if p.match(token.MIT) { // the function takes at least 1 parameter
		validate(p.consumeN(token.DEN, token.PARAMETERN, token.IDENTIFIER) && p.previous().Type == token.IDENTIFIER)
		paramNames = append(make([]token.Token, 0), p.previous()) // append the first parameter name
		for p.match(token.COMMA) {                                // the function takes multiple parameters
			p.consume(token.IDENTIFIER)
			if containsLiteral(paramNames, p.previous().Literal) { // check that each parameter name is unique
				valid = false
				p.err(p.previous(), fmt.Sprintf("Ein Parameter mit dem Namen '%s' ist bereits vorhanden", p.previous().Literal))
			}
			paramNames = append(paramNames, p.previous()) // append the parameter name
		}
		// parse the types of the parameters
		p.consumeN(token.VOM, token.TYP)
		validate(p.consumeAny(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE)) // validate the first parameter type
		paramTypes = append(make([]token.Token, 0), p.previous())                                       // append the first parameter type
		for p.match(token.COMMA) {                                                                      // parse the other parameter types
			if p.check(token.GIBT) { // , gibt indicates the end of the parameter list
				break
			}
			// validate the parameter type and append it
			validate(p.consumeAny(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.TEXT, token.BUCHSTABE))
			paramTypes = append(paramTypes, p.previous())
		}
		if p.previous().Type != token.COMMA {
			p.err(p.previous(), fmt.Sprintf("Es wurde 'COMMA' erwartet aber '%s' gefunden", p.previous().String()))
		}
	}
	// we need as many parmeter names as types
	if len(paramNames) != len(paramTypes) {
		valid = false
		p.err(p.previous(), fmt.Sprintf("Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter vs %d Typen)", len(paramNames), len(paramTypes)))
	}

	// parse the return type declaration
	p.consume(token.GIBT)
	p.consumeAny(token.EINE, token.EINEN, token.NICHTS)
	switch p.previous().Type {
	case token.NICHTS:
	case token.EINE:
		validate(p.consumeAny(token.ZAHL, token.KOMMAZAHL))
	case token.EINEN:
		validate(p.consumeAny(token.BOOLEAN, token.TEXT, token.BUCHSTABE))
	}
	Typ := p.previous()

	p.consumeN(token.ZURÜCK, token.COMMA, token.MACHT, token.COLON)
	bodyStart := p.cur                            // save the body start-position for later, we first need to parse aliases to enable recursion
	indent := p.previous().Indent + 1             // indentation level of the function body
	for p.peek().Indent >= indent && !p.atEnd() { // advance to the alias definitions by checking the indentation
		p.advance()
	}

	// parse the alias definitions before the body to enable recursion
	p.consumeN(token.UND, token.KANN, token.SO, token.BENUTZT, token.WERDEN, token.COLON, token.STRING) // at least 1 alias is required
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

	// scan the raw aliases into tokens
	funcAliases := make([]funcAlias, 0)
	for _, v := range aliases {
		// scan the raw alias withouth the ""
		if alias, err := scanner.ScanAlias([]byte(v.Literal[1:len(v.Literal)-1]), p.errorHandler); err != nil {
			p.err(v, fmt.Sprintf("Der Funktions Alias ist ungültig (%s)", err.Error()))
		} else {
			if len(alias) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
				p.err(v, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if validateAlias(alias, paramNames) { // check that the alias fits the function
				if fun := p.aliasExists(alias); fun != nil { // check that the alias does not already exist for another function
					p.err(v, fmt.Sprintf("Der Alias steht bereits für die Funktion '%s'", *fun))
				} else { // the alias is valid so we append it
					funcAliases = append(funcAliases, funcAlias{Tokens: alias, Func: name.Literal})
				}
			} else {
				valid = false
				p.err(v, "Ein Funktions Alias muss jeden Funktions Parameter genau ein mal enthalten")
			}
		}
	}

	aliasEnd := p.cur // save the end of the function declaration for later

	if p.currentFunction != "" {
		valid = false
		p.err(Funktion, "Es können nur globale Funktionen deklariert werden")
	}

	if !valid {
		p.errored = true
		return &ast.BadDecl{Tok: Funktion}
	}

	p.funcAliases = append(p.funcAliases, funcAliases...)

	// parse the body after the aliases to enable recursion
	p.cur = bodyStart // go back to the body
	p.currentFunction = name.Literal
	body := p.blockStatement() // parse the body
	// check that the function has a return statement if it needs one
	if Typ.Type != token.NICHTS { // only if the function does not return void
		b := body.(*ast.BlockStmt)
		if len(b.Statements) < 1 { // at least the return statement is needed
			p.err(Funktion, "Am Ende einer Funktion die etwas zurück gibt muss eine Rückgabe stehen")
		} else {
			// the last statement must be a return statement
			lastStmt := b.Statements[len(b.Statements)-1]
			if _, ok := lastStmt.(*ast.ReturnStmt); !ok {
				p.err(lastStmt.Token(), "Am Ende einer Funktion die etwas zurück gibt muss eine Rückgabe stehen")
			}
		}
	}

	p.currentFunction = ""
	p.cur = aliasEnd // go back to the end of the function to continue parsing

	return &ast.FuncDecl{
		Func:       Funktion,
		Name:       name,
		ParamNames: paramNames,
		ParamTypes: paramTypes,
		Type:       Typ,
		Body:       body.(*ast.BlockStmt),
	}
}

// helper for funcDeclaration to check that every parameter is provided exactly once
func validateAlias(alias []token.Token, paramNames []token.Token) bool {
	isAliasExpr := func(t token.Token) bool { return t.Type == token.ALIAS_EXPRESSION } // helper to check for parameters
	if countElements(alias, isAliasExpr) != len(paramNames) {                           // validate that the alias contains as many parameters as the function
		return false
	}
	nameSet := map[string]bool{} // set that holds the parameter names contained in the alias
	for _, v := range paramNames {
		nameSet[v.Literal] = true
	}
	// validate that each parameter is contained in the alias exactly once
	for _, v := range selectElements(alias, isAliasExpr) {
		k := v.Literal[1:] // remove the * from *argname
		if _, ok := nameSet[k]; ok {
			delete(nameSet, k)
		} else {
			return false
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

// parse a single statement
func (p *Parser) statement() ast.Statement {
	// check for assignement
	if p.peek().Type == token.IDENTIFIER {
		p.consume(token.IDENTIFIER)
		if p.peek().Type == token.IST {
			return p.assignLiteral() // x ist ... assignements may only have literals, so we use this helper function
		} else {
			p.decrease() // no assignement, so probably an expressionStatement()
		}
	}

	// parse all possible statements
	switch p.peek().Type {
	case token.SPEICHERE:
		p.consume(token.SPEICHERE)
		return p.assignNoLiteral() // Speichere ... in x, where non-literal expressions are allowed
	case token.WENN:
		p.consume(token.WENN)
		return p.ifStatement()
	case token.SOLANGE:
		p.consume(token.SOLANGE)
		return p.whileStatement()
	case token.FÜR:
		p.consume(token.FÜR)
		return p.forStatement()
	case token.GIB:
		p.consume(token.GIB)
		return p.returnStatement()
	case token.COLON:
		p.consume(token.COLON)
		return p.blockStatement()
	}

	// no other statement was found, so interpret it as expression statement, whose result will be discarded
	return p.expressionStatement()
}

// helper to parse assignements which may only be literals
func (p *Parser) assignLiteral() ast.Statement {
	ident := p.previous() // name of the variable was already consumed
	p.consume(token.IST)
	expr := p.assignRhs() // parse the expression
	// validate that the expression is a literal
	switch expr.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.CharLit:
	default:
		if typ, _ := p.resolver.CurrentTable.LookupVar(ident.Literal); typ != token.BOOLEAN {
			p.err(expr.Token(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
		}
	}
	p.consume(token.DOT)
	return &ast.AssignStmt{
		Tok:  ident,
		Name: ident,
		Rhs:  expr,
	}
}

// helper to parse an Speichere expr in x Assignement
func (p *Parser) assignNoLiteral() ast.Statement {
	speichere := p.previous()             // Speichere token
	var expr ast.Expression = nil         // final expression
	if expr = p.funcCall(); expr == nil { // check for funcCall alias which might include "das Ergebnis von" to make that possible
		if p.match(token.DAS) { // if there is no alias, we check the optional "das Ergebnis von" syntax
			p.consumeN(token.ERGEBNIS, token.VON)
		}
		expr = p.expression() // and parse the expression
	}
	p.consumeN(token.IN, token.IDENTIFIER)
	name := p.previous() // name of the variable is the just consumed identifier
	// for booleans, the ist wahr/falsch wenn syntax should be used
	if typ, _ := p.resolver.CurrentTable.LookupVar(name.Literal); typ == token.BOOLEAN {
		p.err(name, "Variablen vom Typ 'BOOLEAN' sind hier nicht zulässig")
	}
	p.consume(token.DOT)
	return &ast.AssignStmt{
		Tok:  speichere,
		Name: name,
		Rhs:  expr,
	}
}

func (p *Parser) ifStatement() ast.Statement {
	If := p.previous()                 // the already consumed wenn token
	condition := p.expression()        // parse the condition
	p.consumeN(token.IST, token.COMMA) // must be boolean, so an ist is required for grammar
	var Then ast.Statement
	if p.match(token.DANN) { // with dann: the body is a block statement
		p.consume(token.COLON)
		Then = p.blockStatement()
	} else { // otherwise it is a single statement
		if p.peek().Type == token.COLON { // block statements are only allowed with the syntax above
			p.err(p.peek(), "In einer Wenn Anweisung, muss ein 'dann' vor einem ':' stehen")
		}
		Then = p.declaration() // parse the single (non-block) statement
	}
	var Else ast.Statement = nil
	// parse a possible sonst statement
	if p.match(token.SONST) {
		// TODO: test if this if is necessary
		if p.match(token.COLON) {
			Else = p.blockStatement() // with colon it is a block statement
		} else { // without it we just parse a single statement
			Else = p.declaration()
		}
	} else if p.match(token.WENN) { // if-else blocks are parsed as nested ifs where the else of the first if is an if-statement
		if p.peek().Type == token.ABER {
			p.consume(token.ABER)
			Else = p.ifStatement() // parse the wenn aber
		} else {
			p.decrease() // no if-else just if, so decrease to parse the next if seperately
		}
	}
	return &ast.IfStmt{
		If:        If,
		Condition: condition,
		Then:      Then,
		Else:      Else,
	}
}

func (p *Parser) whileStatement() ast.Statement {
	While := p.previous()
	condition := p.expression()
	p.consumeN(token.IST, token.COMMA)
	var Body ast.Statement
	// TODO: test if this if is necessery
	if p.match(token.MACHE) {
		p.consume(token.COLON)
		Body = p.blockStatement()
	} else {
		Body = p.declaration()
	}
	return &ast.WhileStmt{
		While:     While,
		Condition: condition,
		Body:      Body,
	}
}

func (p *Parser) forStatement() ast.Statement {
	For := p.previous()
	p.consumeN(token.JEDE, token.ZAHL) // currently we only support int for loops (no strings or lists)
	Zahl := p.previous()
	p.consume(token.IDENTIFIER)
	Ident := p.previous()
	p.consume(token.VON)
	from := p.expression() // start of the counter
	initializer := &ast.VarDecl{
		Type:    Zahl,
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
	p.consumeN(token.COMMA)
	var Body ast.Statement
	if p.match(token.MACHE) { // body is a block statement
		p.consume(token.COLON)
		Body = p.blockStatement()
	} else { // body is a single statement
		Colon := p.previous()
		stmts := make([]ast.Statement, 1)
		stmts[0] = p.declaration()
		// wrap the single statement in a block for variable-scoping of the counter variable in the resolver and typechecker
		Body = &ast.BlockStmt{
			Colon:      Colon,
			Statements: stmts,
			Symbols:    nil,
		}
	}
	return &ast.ForStmt{
		For:         For,
		Initializer: initializer,
		To:          to,
		StepSize:    step,
		Body:        Body,
	}
}

func (p *Parser) returnStatement() ast.Statement {
	Return := p.previous()
	expr := p.expression()
	p.consumeN(token.ZURÜCK, token.DOT)
	return &ast.ReturnStmt{
		Func:   p.currentFunction,
		Return: Return,
		Value:  expr,
	}
}

func (p *Parser) blockStatement() ast.Statement {
	colon := p.previous()
	if p.peek().Line <= colon.Line {
		p.err(p.peek(), "Nach einem Doppelpunkt muss eine neue Zeile beginnen")
	}
	statements := make([]ast.Statement, 0)
	indent := colon.Indent + 1
	for p.peek().Indent >= indent && !p.atEnd() {
		statements = append(statements, p.declaration())
		if p.panicMode { // a loop calling declaration or sub-rules needs this
			p.synchronize()
		}
	}
	return &ast.BlockStmt{
		Colon:      colon,
		Statements: statements,
		Symbols:    nil,
	}
}

func (p *Parser) expressionStatement() ast.Statement {
	stmt := &ast.ExprStmt{Expr: p.expression()}
	p.consume(token.DOT)
	return stmt
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
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) boolAND() ast.Expression {
	expr := p.equality()
	for p.match(token.UND) {
		operator := p.previous()
		rhs := p.equality()
		expr = &ast.BinaryExpr{
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
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
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
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) bitShift() ast.Expression {
	expr := p.term()
	for p.match(token.UM) {
		rhs := p.term()
		p.consumeN(token.BIT, token.NACH)
		if !p.match(token.LINKS, token.RECHTS) {
			p.err(p.previous(), fmt.Sprintf("Es wurde 'LINKS' oder 'RECHTS' erwartet aber '%s' gefunden", p.previous().Literal))
			return &ast.BadExpr{Tok: expr.Token()}
		}
		operator := p.previous()
		expr = &ast.BinaryExpr{
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
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *Parser) unary() ast.Expression {
	if expr := p.funcCall(); expr != nil { // first check for a function call to enable operator overloading
		return expr
	}
	// match the correct unary operator
	if p.match(token.NICHT, token.BETRAG, token.NEGATE, token.DIE, token.GRÖßE, token.LÄNGE, token.DER) {
		if p.previous().Type == token.DIE {
			if !p.match(token.GRÖßE, token.LÄNGE) {
				p.decrease() // DIE does not belong to a operator, so maybe it is a function call
				return p.primary()
			}
		} else if p.previous().Type == token.DER {
			if !p.match(token.BETRAG) {
				p.decrease() // DER does not belong to a operator, so maybe it is a function call
				return p.primary()
			}

		} else { // error handling
			switch p.previous().Type {
			case token.GRÖßE, token.LÄNGE:
				p.err(p.previous(), fmt.Sprintf("Vor '%s' muss 'DIE' stehen", p.previous().String()))
			case token.BETRAG:
				p.err(p.previous(), "Vor 'BETRAG' muss 'DER' stehen")
			}
		}
		operator := p.previous()
		switch operator.Type {
		case token.BETRAG, token.GRÖßE, token.LÄNGE:
			p.consume(token.VON)
		}
		return &ast.UnaryExpr{
			Operator: operator,
			Rhs:      p.unary(),
		}
	}
	expr := p.primary()

	// type-casting
	if p.match(token.ALS) {
		p.consumeAny(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE, token.TEXT)
		return &ast.UnaryExpr{
			Operator: p.previous(),
			Rhs:      expr,
		}
	}

	return expr
}

func (p *Parser) primary() ast.Expression {
	if expr := p.funcCall(); expr != nil { // funccall has the highest precedence (aliases + operator overloading)
		return expr
	}
	// literals
	if p.match(token.FALSE) {
		return &ast.BoolLit{Literal: p.previous(), Value: false}
	}
	if p.match(token.TRUE) {
		return &ast.BoolLit{Literal: p.previous(), Value: true}
	}
	if p.match(token.INT) {
		lit := p.previous()
		if val, err := strconv.ParseInt(lit.Literal, 10, 64); err == nil {
			return &ast.IntLit{Literal: lit, Value: val}
		} else {
			p.err(lit, fmt.Sprintf("Das Zahlen Literal '%s' kann nicht gelesen werden", lit.Literal))
			return &ast.IntLit{Literal: lit, Value: 0}
		}
	}
	if p.match(token.FLOAT) {
		lit := p.previous()
		if val, err := strconv.ParseFloat(strings.Replace(lit.Literal, ",", ".", 1), 64); err == nil {
			return &ast.FloatLit{Literal: lit, Value: val}
		} else {
			p.err(lit, fmt.Sprintf("Das Zahlen Literal '%s' kann nicht gelesen werden", lit.Literal))
			return &ast.FloatLit{Literal: lit, Value: 0}
		}
	}
	if p.match(token.CHAR) {
		lit := p.previous()
		return &ast.CharLit{Literal: lit, Value: p.parseChar(lit.Literal)}
	}
	if p.match(token.STRING) {
		lit := p.previous()
		return &ast.StringLit{Literal: lit, Value: p.parseString(lit.Literal)}
	}
	if p.match(token.LPAREN) {
		lParen := p.previous()
		expr := p.expression()
		p.consume(token.RPAREN)
		return &ast.Grouping{
			LParen: lParen,
			Expr:   expr,
		}
	}
	if p.match(token.IDENTIFIER) {
		return &ast.Ident{
			Literal: p.previous(),
		}
	}
	p.err(p.peek(), fmt.Sprintf("Es wurde ein Ausdruck erwartet aber '%s' gefunden", p.peek().Literal))
	return &ast.BadExpr{
		Tok: p.peek(),
	}
}

func (p *Parser) funcCall() ast.Expression {
	// stores an alias with the actual length of all tokens (expanded token.ALIAS_EXPRESSIONs)
	type matchedAlias struct {
		alias        *funcAlias // original alias
		actualLength int        // length of this occurence in the code
	}

	start := p.cur                            // save start position to restore the state if no alias was recognized
	matchedAliases := make([]matchedAlias, 0) // stors all matched aliases (can be multiple due to different lengths)

	// loop through all possible aliases (expensive, might change later)
outer:
	for i, l := 0, len(p.funcAliases); i < l; i++ {
		alias := &p.funcAliases[i]

		// loop through all the tokens in the alias
		for ii, ll := 0, len(alias.Tokens); ii < ll && alias.Tokens[ii].Type != token.EOF; ii++ {
			tok := &alias.Tokens[ii]

			// expand arguments
			if tok.Type == token.ALIAS_EXPRESSION {
				switch p.peek().Type {
				case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER:
					p.advance() // single-token so skip it
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
	if len(matchedAliases) != 0 { // check if any alias was matched
		var finalAlias *matchedAlias = nil // stores the resulting alias
		// use the longest possible alias
		// if some are of the same length use the one defined first
		length := 0
		for i, v := range matchedAliases {
			if len(v.alias.Tokens) > length {
				length = len(v.alias.Tokens)
				finalAlias = &matchedAliases[i]
			}
		}

		// parse the call arguments
		p.cur = start
		args := map[string]ast.Expression{}
		// go through the whole alias again, applying nearly the same algorithm as above (can I get this into a separate function?)
		for i, l := 0, len(finalAlias.alias.Tokens); i < l && finalAlias.alias.Tokens[i].Type != token.EOF; i++ {
			tok := &finalAlias.alias.Tokens[i]

			if tok.Type == token.ALIAS_EXPRESSION {
				exprStart := p.cur
				switch p.peek().Type {
				case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER:
					p.advance() // single-token argument
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
				tokens = append(tokens, token.Token{Type: token.EOF, Literal: "", Indent: 0, File: tok.File, Line: tok.Line, Column: tok.Column})
				parser := New(tokens, p.errorHandler) // create a new parser for this expression
				parser.funcAliases = p.funcAliases    // it needs the functions aliases
				arg := parser.expression()            // parse the argument
				args[tok.Literal[1:]] = arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}

		p.cur = start + finalAlias.actualLength
		return &ast.FuncCall{
			Tok:  p.tokens[start],
			Name: finalAlias.alias.Func,
			Args: args,
		}
	}

	return nil // no alias was matched -> we are not in a function call
}

/*** Helper functions ***/

// helper to parse ddp chars with escape sequences
func (p *Parser) parseChar(s string) (r rune) {
	lit := s[1 : len(s)-1] // remove the ''
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
			p.err(p.previous(), fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Buchstaben Literal", string(r)))
		}
		return r
	}
	p.err(p.previous(), "Invalides Buchstaben Literal")
	return -1
}

// helper to parse ddp strings with escape sequences
func (p *Parser) parseString(s string) string {
	str := s[1 : len(s)-1] // remove the ""
	for i, w := 0, 0; i < len(str); i += w {
		var r rune
		r, w = utf8.DecodeRuneInString(str[i:])
		if r == '\\' {
			r2, w2 := utf8.DecodeRuneInString(str[i+w:])
			switch r2 {
			case 'a':
				r2 = '\a'
			case 'b':
				r2 = '\b'
			case 'n':
				r2 = '\n'
			case 'r':
				r2 = '\r'
			case 't':
				r2 = '\t'
			case '"':
			case '\\':
			default:
				p.err(p.previous(), fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Text Literal", string(r2)))
				continue
			}
			str = str[0:i] + string(r2) + str[i+w+w2:]
		}
	}
	return str
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

// if the current token is of type t advance, otherwise error
func (p *Parser) consume(t token.TokenType) bool {
	if p.check(t) {
		p.advance()
		return true
	}
	p.err(p.peek(), fmt.Sprintf("Es wurde '%s' erwartet aber '%s' gefunden", t.String(), p.peek().Literal))
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
func (p *Parser) consumeAny(t ...token.TokenType) bool {
	for _, v := range t {
		if p.check(v) {
			p.advance()
			return true
		}
	}
	msg := "Es wurde "
	for i, v := range t {
		if i >= len(t)-1 {
			break
		}
		msg += "'" + v.String() + "', "
	}
	msg += "oder '" + t[len(t)-1].String() + "' erwartet aber '" + p.peek().Literal + "' gefunden"
	p.err(p.peek(), msg)
	return false
}

// helper to report errors and enter panic mode
func (p *Parser) err(t token.Token, msg string) {
	if !p.panicMode {
		p.panicMode = true
		p.errorHandler(fmt.Sprintf("Fehler in %s in Zeile %d, Spalte %d: %s", t.File, t.Line, t.Column, msg))
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
	}
	return p.previous()
}

// returns the current token without advancing
func (p *Parser) peek() token.Token {
	return p.tokens[p.cur]
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
	if t1.Type == token.IDENTIFIER {
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

// selects all elements in the slice which fulfill the provided test function
func selectElements[T any](elements []T, test func(T) bool) (result []T) {
	for _, v := range elements {
		if test(v) {
			result = append(result, v)
		}
	}
	return result
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
