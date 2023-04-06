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
	"github.com/DDP-Projekt/Kompilierer/pkg/ddppath"
	"github.com/DDP-Projekt/Kompilierer/pkg/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/pkg/scanner"
	"github.com/DDP-Projekt/Kompilierer/pkg/token"
)

// holds state when parsing a .ddp file into an AST
type parser struct {
	tokens       []token.Token    // the tokens to parse (without comments)
	comments     []token.Token    // all the comments from the original tokens slice
	cur          int              // index of the current token
	errorHandler ddperror.Handler // a function to which errors are passed
	lastError    ddperror.Error   // latest reported error

	module            *ast.Module
	predefinedModules map[string]*ast.Module   // modules that were passed as environment, might not all be used
	funcAliases       []ast.FuncAlias          // all found aliases (+ inbuild aliases)
	currentFunction   string                   // function which is currently being parsed
	panicMode         bool                     // flag to not report following errors
	errored           bool                     // wether the parser found an error
	resolver          *resolver.Resolver       // used to resolve every node directly after it has been parsed
	typechecker       *typechecker.Typechecker // used to typecheck every node directly after it has been parsed
}

// returns a new parser, ready to parse the provided tokens
func newParser(tokens []token.Token, modules map[string]*ast.Module, errorHandler ddperror.Handler) *parser {
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

	path, err := filepath.Abs(pTokens[0].File)
	if err != nil {
		path = ""
	}

	aliases := make([]ast.FuncAlias, 0)
	parser := &parser{
		tokens:       pTokens,
		comments:     pComments,
		cur:          0,
		errorHandler: nil,
		module: &ast.Module{
			FileName:             path,
			Imports:              make([]*ast.ImportStmt, 0),
			ExternalDependencies: make(map[string]struct{}),
			Ast: &ast.Ast{
				Statements: make([]ast.Statement, 0),
				Comments:   pComments,
				Symbols:    ast.NewSymbolTable(nil),
				Faulty:     false,
			},
			PublicDecls: make(map[string]ast.Declaration),
		},
		predefinedModules: modules,
		funcAliases:       aliases,
		panicMode:         false,
		errored:           false,
		resolver:          &resolver.Resolver{},
		typechecker:       &typechecker.Typechecker{},
	}

	// wrap the errorHandler to set the parsers Errored variable
	// if it is called
	parser.errorHandler = func(err ddperror.Error) {
		parser.errored = true
		errorHandler(err)
	}

	// prepare the resolver and typechecker with the inbuild symbols and types
	parser.resolver = resolver.New(parser.module.Ast, parser.errorHandler)
	parser.typechecker = typechecker.New(parser.module.Ast.Symbols, parser.errorHandler)

	return parser
}

// parse the provided tokens into an Ast
func (p *parser) Parse() *ast.Module {
	// main parsing loop
	for !p.atEnd() {
		if stmt := p.checkedDeclaration(); stmt != nil {
			p.module.Ast.Statements = append(p.module.Ast.Statements, stmt)
		}
	}

	// if any error occured, the AST is faulty
	if p.errored || p.resolver.Errored || p.typechecker.Errored {
		p.module.Ast.Faulty = true
	}

	return p.module
}

// if an error was encountered we synchronize to a point where correct parsing is possible again
func (p *parser) synchronize() {
	p.panicMode = false

	//p.advance() // maybe this needs to stay?
	for !p.atEnd() {
		if p.previous().Type == token.DOT { // a . ends statements, so we can continue parsing
			return
		}
		// these tokens typically begin statements which begin a new node
		switch p.peek().Type {
		// DIE/DER does not always indicate a declaration
		// so we only return if the next token fits
		case token.DIE:
			switch p.peekN(1).Type {
			case token.ZAHL, token.KOMMAZAHL, token.ZAHLEN, token.KOMMAZAHLEN,
				token.BUCHSTABEN, token.TEXT, token.BOOLEAN, token.FUNKTION:
				{
					return
				}
			}
		case token.DER:
			switch p.peekN(1).Type {
			case token.BOOLEAN, token.TEXT, token.BUCHSTABE, token.ALIAS:
				{
					return
				}
			}
		case token.WENN, token.FÜR, token.GIB, token.VERLASSE, token.SOLANGE,
			token.COLON, token.MACHE, token.DANN, token.WIEDERHOLE:
			return
		}
		p.advance()
	}
}

// fils out importStmt.Module and updates the parser state accordingly
func (p *parser) resolveModuleImport(importStmt *ast.ImportStmt) {
	p.module.Imports = append(p.module.Imports, importStmt) // add the import to the module

	rawPath := ast.TrimStringLit(importStmt.FileName)
	inclPath := ""

	// resolve the actual file path
	var err error
	if strings.HasPrefix(rawPath, "Duden") {
		inclPath = filepath.Join(ddppath.InstallDir, rawPath) + ".ddp"
	} else {
		inclPath, err = filepath.Abs(filepath.Join(filepath.Dir(p.module.FileName), rawPath+".ddp"))
	}

	if err != nil {
		p.err(ddperror.SYN_MALFORMED_INCLUDE_PATH, importStmt.FileName.Range, fmt.Sprintf("Fehlerhafter Dateipfad '%s': \"%s\"", rawPath+".ddp", err.Error()), p.module.FileName)
		return
	} else if module, ok := p.predefinedModules[inclPath]; !ok { // the module is new
		p.predefinedModules[inclPath] = nil // already add the name to the map to not import it infinetly
		// parse the new module
		importStmt.Module, err = Parse(Options{
			FileName:     inclPath,
			Source:       nil,
			Tokens:       nil,
			Modules:      p.predefinedModules,
			ErrorHandler: p.errorHandler,
		})

		// add the module to the list and to the importStmt
		// or report the error
		if err != nil {
			p.err(ddperror.MISC_INCLUDE_ERROR, importStmt.Range, fmt.Sprintf("Fehler beim einbinden von '%s': %s", rawPath, err.Error()), importStmt.FileName.File)
			return // return early on error
		} else {
			p.predefinedModules[inclPath] = importStmt.Module
		}
	} else { // we already included the module
		importStmt.Module = module
	}

	// helper to add an alias slice to the parser
	// only adds aliases that are not already defined
	// and errors otherwise
	addAliases := func(aliases []ast.FuncAlias, errRange token.Range) {
		// add all the aliases
		for _, alias := range aliases {
			if funcName := p.aliasExists(alias.Tokens); funcName != nil {
				p.err(ddperror.SEM_ALIAS_ALREADY_DEFINED, errRange, ddperror.MsgAliasAlreadyExists(alias.Original.Literal, *funcName), importStmt.FileName.File)
			} else {
				p.funcAliases = append(p.funcAliases, alias)
			}
		}
	}

	if len(importStmt.ImportedSymbols) == 0 {
		// add all public aliases
		for name, decl := range importStmt.Module.PublicDecls {
			funcDecl, isFunc := decl.(*ast.FuncDecl) // skip VarDecls
			// skip functions that are already defined
			// the resolver will error here
			_, exists, _ := p.resolver.CurrentTable.LookupDecl(name)
			// continue if the conditions are not met
			if !isFunc || exists {
				continue
			}

			// add all the aliases
			addAliases(funcDecl.Aliases, importStmt.Range)
		}
	} else {
		// only add the imported ones
		for _, tok := range importStmt.ImportedSymbols {
			name := tok.Literal
			// check that the name exists as a public declaration
			_, exists := importStmt.Module.PublicDecls[name]
			funcDecl, isFunc := importStmt.Module.PublicDecls[name].(*ast.FuncDecl)
			if !isFunc || !exists {
				continue
			}

			// add all the aliases
			addAliases(funcDecl.Aliases, tok.Range)
		}
	}
}

func (p *parser) checkedDeclaration() ast.Statement {
	stmt := p.declaration() // parse the node
	// TODO: maybe introduce a NOOP node to handle such cases
	if stmt != nil { // nil check, for alias declarations that aren't Ast Nodes
		if importStmt, ok := stmt.(*ast.ImportStmt); ok {
			p.resolveModuleImport(importStmt)
		}
		p.resolver.ResolveNode(stmt)      // resolve symbols in it (variables, functions, ...)
		p.typechecker.TypecheckNode(stmt) // typecheck the node
	}
	if p.panicMode { // synchronize the parsing if we are in panic mode
		p.synchronize()
	}
	return stmt
}

// entry point for the recursive descent parsing
func (p *parser) declaration() ast.Statement {
	if p.match(token.DER, token.DIE) { // might indicate a function or variable declaration
		n := -1
		if p.match(token.OEFFENTLICHE) {
			n = -2
		}
		switch p.peekN(n).Type {
		case token.DER:
			switch p.peek().Type {
			case token.BOOLEAN, token.TEXT, token.BUCHSTABE:
				p.advance()                                         // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration(n - 1)} // parse the declaration
			case token.ALIAS:
				p.advance()
				return p.aliasDecl()
			default:
				p.decrease() // decrease, so expressionStatement() can recognize it as expression
			}
		case token.DIE:
			switch p.peek().Type {
			case token.ZAHL, token.KOMMAZAHL, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.TEXT, token.BOOLEAN:
				p.advance()                                         // consume the type
				return &ast.DeclStmt{Decl: p.varDeclaration(n - 1)} // parse the declaration
			case token.FUNKTION:
				p.advance()
				return &ast.DeclStmt{Decl: p.funcDeclaration(n - 1)} // parse the function declaration
			default:
				p.decrease() // decrease, so expressionStatement() can recognize it as expression
			}
		}
		if n == -2 {
			p.decrease()
		}
	}

	return p.statement() // no declaration, so it must be a statement
}

// helper for boolean assignments
func (p *parser) assignRhs() ast.Expression {
	var expr ast.Expression // the final expression

	if p.match(token.TRUE, token.FALSE) {
		// parse possible wahr/falsch wenn syntax
		if p.match(token.COMMA) {
			p.consume(token.WENN)
			// if it is false, we add a unary bool-negate into the ast
			if tok := p.tokens[p.cur-2]; tok.Type == token.FALSE {
				rhs := p.expression() // the actual boolean expression after falsch wenn, which is negated
				expr = &ast.UnaryExpr{
					Range: token.Range{
						Start: token.NewStartPos(tok),
						End:   rhs.GetRange().End,
					},
					Tok:      tok,
					Operator: ast.UN_NOT,
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
				p.err(ddperror.SYN_EXPECTED_LITERAL, expr.GetRange(), ddperror.MsgGotExpected("ein Ausdruck", "ein Literal"), expr.Token().File)
			}
		}
	} else {
		expr = p.expression() // no wahr/falsch, so a normal expression
	}

	return expr
}

// parses a variable declaration
// startDepth is the int passed to p.peekN(n) to get to the DER/DIE token of the declaration
func (p *parser) varDeclaration(startDepth int) ast.Declaration {
	begin := p.peekN(startDepth)
	comment := p.commentBeforePos(begin.Range.Start, begin.File)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}
	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE
	p.decrease()
	typ := p.parseType()

	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.peek().File,
				Msg:   "Es wurde ein Variablen Name erwartet",
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
	decl := &ast.VarDecl{
		Range:    token.NewRange(begin, p.previous()),
		Comment:  comment,
		Type:     typ,
		Name:     name,
		IsPublic: isPublic,
		InitVal:  expr,
	}
	if _, alreadyExists := p.module.PublicDecls[decl.Name.Literal]; decl.IsPublic && !alreadyExists {
		p.module.PublicDecls[decl.Name.Literal] = decl
	}
	return decl
}

// parses a function declaration
// startDepth is the int passed to p.peekN(n) to get to the DIE token of the declaration
func (p *parser) funcDeclaration(startDepth int) ast.Declaration {
	valid := true              // checks if the function is valid and may be appended to the parser state as p.errored = !valid
	validate := func(b bool) { // helper for setting the valid flag (to simplify some big boolean expressions)
		if !b {
			valid = false
		}
	}

	perr := func(code ddperror.Code, Range token.Range, msg string, file string) {
		p.err(code, Range, msg, file)
		valid = false
	}

	begin := p.peekN(startDepth)
	comment := p.commentBeforePos(begin.Range.Start, begin.File)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE

	Funktion := p.previous() // save the token
	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.New(ddperror.SYN_EXPECTED_IDENTIFIER, token.NewRange(begin, p.peek()), "Es wurde ein Funktions Name erwartet", p.peek().File),
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
			perr(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "'de[n/m] Parameter[n]'"), p.peek().File)
		}
		validate(p.consume(token.IDENTIFIER))
		paramNames = append(make([]token.Token, 0), p.previous()) // append the first parameter name
		paramComments = append(make([]*token.Token, 0), p.getLeadingOrTrailingComment())
		if !singleParameter {
			// helper function to avoid too much repitition
			addParamName := func(name token.Token) {
				if containsLiteral(paramNames, name.Literal) { // check that each parameter name is unique
					perr(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, fmt.Sprintf("Ein Parameter mit dem Namen '%s' ist bereits vorhanden", name.Literal), name.File)
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
					perr(ddperror.SYN_EXPECTED_IDENTIFIER, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "der letzte Parameter (und <Name>)")+"\nMeintest du vorher vielleicht 'dem Parameter' anstatt 'den Parametern'?", p.peek().File)
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
			// helper function to avoid too much repitition
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
		perr(ddperror.SEM_PARAM_NAME_TYPE_COUNT_MISMATCH, token.NewRange(paramNames[0], p.previous()), fmt.Sprintf("Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)", len(paramNames), len(paramTypes)), paramNames[0].File)
	}

	// parse the return type declaration
	validate(p.consume(token.GIBT))
	Typ := p.parseReturnType()
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
		switch filepath.Ext(ast.TrimStringLit(definedIn)) {
		case ".c", ".lib", ".a", ".o":
		default:
			perr(ddperror.SEM_EXPECTED_LINKABLE_FILEPATH, definedIn.Range, fmt.Sprintf("Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden", definedIn.Literal), definedIn.File)
		}
	}

	// parse the alias definitions before the body to enable recursion
	validate(p.consumeN(token.UND, token.KANN, token.SO, token.BENUTZT, token.WERDEN, token.COLON, token.STRING)) // at least 1 alias is required
	aliases := make([]token.Token, 0)
	if p.previous().Type == token.STRING {
		aliases = append(aliases, p.previous())
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
				perr(ddperror.SEM_MALFORMED_ALIAS, v.Range, "Ein Alias muss mindestens 1 Symbol enthalten", v.File)
			} else if validateAlias(alias, paramNames, paramTypes) { // check that the alias fits the function
				if fun := p.aliasExists(alias); fun != nil { // check that the alias does not already exist for another function
					perr(ddperror.SEM_ALIAS_ALREADY_TAKEN, v.Range, ddperror.MsgAliasAlreadyExists(v.Literal, *fun), v.File)
				} else { // the alias is valid so we append it
					funcAliases = append(funcAliases, ast.FuncAlias{Tokens: alias, Original: v, Func: name.Literal, Args: paramTypesMap})
				}
			} else {
				perr(ddperror.SEM_MALFORMED_ALIAS, v.Range, "Ein Funktions Alias muss jeden Funktions Parameter genau ein mal enthalten", v.File)
			}
		}
	}

	aliasEnd := p.cur // save the end of the function declaration for later

	if p.currentFunction != "" {
		perr(ddperror.SEM_NON_GLOBAL_FUNCTION, begin.Range, "Es können nur globale Funktionen deklariert werden", begin.File)
	}

	if !valid {
		p.cur = aliasEnd
		return &ast.BadDecl{
			Err: p.lastError,
			Tok: Funktion,
		}
	}

	p.funcAliases = append(p.funcAliases, funcAliases...)

	decl := &ast.FuncDecl{
		Range:         token.NewRange(begin, p.previous()),
		Comment:       comment,
		Tok:           begin,
		Name:          name,
		IsPublic:      isPublic,
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
		if existed := globalScope.InsertDecl(p.currentFunction, decl); existed { // insert the name of the current function
			p.err(ddperror.SEM_NAME_ALREADY_DEFINED, decl.Name.Range, ddperror.MsgNameAlreadyExists(decl.Name.Literal), decl.Tok.File)
		} else if decl.IsPublic {
			p.module.PublicDecls[decl.Name.Literal] = decl
		}
		// add the parameters to the table
		for i, l := 0, len(paramNames); i < l; i++ {
			bodyTable.InsertDecl(paramNames[i].Literal, &ast.VarDecl{Name: paramNames[i], Type: paramTypes[i].Type, Range: token.NewRange(paramNames[i], paramNames[i]), Comment: paramComments[i]})
		}
		body = p.blockStatement(bodyTable).(*ast.BlockStmt) // parse the body with the parameters in the current table
		decl.Body = body

		// check that the function has a return statement if it needs one
		if Typ != ddptypes.Void() { // only if the function does not return void
			if len(body.Statements) < 1 { // at least the return statement is needed
				perr(ddperror.SEM_MISSING_RETURN, body.Range, ddperror.MSG_MISSING_RETURN, body.Token().File)
			} else {
				// the last statement must be a return statement
				lastStmt := body.Statements[len(body.Statements)-1]
				if _, ok := lastStmt.(*ast.ReturnStmt); !ok {
					perr(ddperror.SEM_MISSING_RETURN, token.NewRange(p.previous(), p.previous()), ddperror.MSG_MISSING_RETURN, lastStmt.Token().File)
				}
			}
		}
	} else { // the function is defined in an extern file
		if existed := p.resolver.CurrentTable.InsertDecl(name.Literal, decl); existed { // insert the name of the current function
			p.err(ddperror.SEM_NAME_ALREADY_DEFINED, decl.Name.Range, ddperror.MsgNameAlreadyExists(decl.Name.Literal), decl.Tok.File)
		} else if decl.IsPublic {
			p.module.PublicDecls[decl.Name.Literal] = decl
		}
		p.module.ExternalDependencies[ast.TrimStringLit(decl.ExternFile)] = struct{}{} // add the extern declaration
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
func (p *parser) aliasExists(alias []token.Token) *string {
	for i := range p.funcAliases {
		if slicesEqual(alias, p.funcAliases[i].Tokens, tokenEqual) {
			return &p.funcAliases[i].Func
		}
	}
	return nil
}

func (p *parser) aliasDecl() ast.Statement {
	begin := p.peekN(-2)
	p.consume(token.STRING)
	aliasTok := p.previous()
	p.consumeN(token.STEHT, token.FÜR, token.DIE, token.FUNKTION, token.IDENTIFIER)
	fun := p.previous()

	decl, ok, isVar := p.resolver.CurrentTable.LookupDecl(fun.Literal)
	if !ok {
		p.err(ddperror.SEM_NAME_UNDEFINED, fun.Range, fmt.Sprintf("Der Name %s wurde noch nicht deklariert", fun.Literal), fun.File)
		return nil
	} else if isVar {
		p.err(ddperror.SEM_BAD_NAME_CONTEXT, fun.Range, fmt.Sprintf("Der Name %s steht für eine Variable und nicht für eine Funktion", fun.Literal), fun.File)
		return nil
	}
	funDecl := decl.(*ast.FuncDecl)

	// map function parameters to their type (given to the alias if it is valid)
	paramTypes := map[string]ddptypes.ParameterType{}
	for i, v := range funDecl.ParamNames {
		if i < len(funDecl.ParamTypes) {
			paramTypes[v.Literal] = funDecl.ParamTypes[i]
		}
	}

	// scan the raw alias withouth the ""
	var alias *ast.FuncAlias
	if aliasTokens, err := scanner.ScanAlias(aliasTok, func(err ddperror.Error) { p.err(err.Code, err.Range, err.Msg, err.File) }); err == nil && len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
		p.err(ddperror.SEM_MALFORMED_ALIAS, aliasTok.Range, "Ein Alias muss mindestens 1 Symbol enthalten", aliasTok.File)
	} else if validateAlias(aliasTokens, funDecl.ParamNames, funDecl.ParamTypes) { // check that the alias fits the function
		if fun := p.aliasExists(aliasTokens); fun != nil { // check that the alias does not already exist for another function
			p.err(ddperror.SEM_ALIAS_ALREADY_DEFINED, aliasTok.Range, ddperror.MsgAliasAlreadyExists(aliasTok.Literal, *fun), aliasTok.File)
		} else { // the alias is valid so we append it
			alias = &ast.FuncAlias{Tokens: aliasTokens, Original: aliasTok, Func: funDecl.Name.Literal, Args: paramTypes}
		}
	} else {
		p.err(ddperror.SEM_MALFORMED_ALIAS, aliasTok.Range, "Ein Funktions Alias muss jeden Funktions Parameter genau ein mal enthalten", aliasTok.File)
	}

	p.consume(token.DOT)

	if begin.Indent > 0 {
		p.err(ddperror.SEM_ALIAS_MUST_BE_GLOBAL, token.NewRange(begin, p.previous()), "Ein Alias darf nur im globalen Bereich deklariert werden!", begin.File)
		return &ast.BadStmt{
			Err: p.lastError,
			Tok: begin,
		}
	} else if alias != nil {
		p.funcAliases = append(p.funcAliases, *alias)
		funDecl.Aliases = append(funDecl.Aliases, *alias)
	}
	return nil
}

// parse a single statement
func (p *parser) statement() ast.Statement {
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
	case token.BINDE:
		p.consume(token.BINDE)
		return p.importStatement()
	case token.ERHÖHE, token.VERRINGERE,
		token.VERVIELFACHE, token.TEILE,
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

func (p *parser) importStatement() ast.Statement {
	binde := p.previous()
	var stmt *ast.ImportStmt
	if p.match(token.STRING) {
		stmt = &ast.ImportStmt{
			FileName:        p.previous(),
			ImportedSymbols: nil,
		}
	} else if p.match(token.IDENTIFIER) {
		importedSymbols := []token.Token{p.previous()}
		if p.peek().Type != token.AUS {
			if p.match(token.UND) {
				p.consume(token.IDENTIFIER)
				importedSymbols = append(importedSymbols, p.previous())
			} else {
				for p.match(token.COMMA) {
					p.consume(token.IDENTIFIER)
					importedSymbols = append(importedSymbols, p.previous())
				}
				p.consume(token.UND)
				p.consume(token.IDENTIFIER)
				importedSymbols = append(importedSymbols, p.previous())
			}
		}
		p.consumeN(token.AUS, token.STRING)
		stmt = &ast.ImportStmt{
			FileName:        p.previous(),
			ImportedSymbols: importedSymbols,
		}
	} else {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "ein Text Literal oder ein Name"), p.peek().File)
		return &ast.BadStmt{
			Tok: p.peek(),
			Err: p.lastError,
		}
	}
	p.consumeN(token.EIN, token.DOT)
	stmt.Range = token.NewRange(binde, p.previous())
	return stmt
}

// either consumes the neccesery . or adds a postfix do-while or repeat
func (p *parser) finishStatement(stmt ast.Statement) ast.Statement {
	if p.match(token.DOT) {
		return stmt
	}
	count := p.expression()
	if !p.match(token.COUNT_MAL) {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, count.GetRange(),
			fmt.Sprintf("%s\nWolltest du vor %s vielleicht einen Punkt setzten?", ddperror.MsgGotExpected(p.previous(), token.COUNT_MAL), count.Token()),
			count.Token().File,
		)
	}
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

	var operand ast.Expression
	var varName ast.Assigneable
	p.consume(token.IDENTIFIER)
	if p.match(token.LPAREN) { // indexings may be enclosed in parens to prevent the 'um' from being interpretetd as bitshift
		varName = p.assigneable()
		p.consume(token.RPAREN)
	} else {
		varName = p.assigneable()
	}

	// early return for negate as it does not need a second operand
	if tok.Type == token.NEGIERE {
		p.consume(token.DOT)
		typ := p.typechecker.EvaluateSilent(varName)
		operator := ast.UN_NEGATE
		if typ == ddptypes.Bool() {
			operator = ast.UN_NOT
		}
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   tok,
			Var:   varName,
			Rhs: &ast.UnaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      tok,
				Operator: operator,
				Rhs:      varName,
			},
		}
	}

	if tok.Type == token.TEILE {
		p.consume(token.DURCH)
	} else {
		p.consume(token.UM)
	}

	operand = p.expression()

	if tok.Type == token.VERSCHIEBE {
		p.consumeN(token.BIT, token.NACH)
		p.consumeAny(token.LINKS, token.RECHTS)
		assign_token := tok
		tok = p.previous()
		operator := ast.BIN_LEFT_SHIFT
		if tok.Type == token.RECHTS {
			operator = ast.BIN_RIGHT_SHIFT
		}
		p.consume(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   assign_token,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      tok,
				Lhs:      varName,
				Operator: operator,
				Rhs:      operand,
			},
		}
	} else {
		p.consume(token.DOT)
		return &ast.AssignStmt{
			Range: token.NewRange(tok, p.previous()),
			Tok:   tok,
			Var:   varName,
			Rhs: &ast.BinaryExpr{
				Range:    token.NewRange(tok, p.previous()),
				Tok:      tok,
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
	p.consume(token.IST)
	expr := p.assignRhs() // parse the expression
	// validate that the expression is a literal
	switch expr := expr.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.CharLit, *ast.ListLit:
	default:
		if typ := p.typechecker.Evaluate(ident); typ != ddptypes.Bool() {
			p.err(ddperror.SYN_EXPECTED_LITERAL, expr.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden", expr.Token().File)
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
func (p *parser) assignNoLiteral() ast.Statement {
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

func (p *parser) ifStatement() ast.Statement {
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
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, "In einer Wenn Anweisung, muss ein 'dann' vor dem ':' stehen", p.peek().File)
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

func (p *parser) whileStatement() ast.Statement {
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

func (p *parser) doWhileStmt() ast.Statement {
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

func (p *parser) repeatStmt() ast.Statement {
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

func (p *parser) forStatement() ast.Statement {
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
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		if p.match(token.MACHE) {                        // body is a block statement
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
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		if p.match(token.MACHE) {                        // body is a block statement
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
	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "'von'", "'in'"), p.peek().File)
	return &ast.BadStmt{
		Err: p.lastError,
		Tok: p.previous(),
	}
}

func (p *parser) returnStatement() ast.Statement {
	Return := p.previous()
	expr := p.expression()
	p.consumeN(token.ZURÜCK, token.DOT)
	rnge := token.NewRange(Return, p.previous())
	if p.currentFunction == "" {
		p.err(ddperror.SEM_GLOBAL_RETURN, rnge, ddperror.MSG_GLOBAL_RETURN, Return.File)
	}
	return &ast.ReturnStmt{
		Range:  rnge,
		Func:   p.currentFunction,
		Return: Return,
		Value:  expr,
	}
}

func (p *parser) voidReturn() ast.Statement {
	Leave := p.previous()
	p.consumeN(token.DIE, token.FUNKTION, token.DOT)
	rnge := token.NewRange(Leave, p.previous())
	if p.currentFunction == "" {
		p.err(ddperror.SEM_GLOBAL_RETURN, rnge, ddperror.MSG_GLOBAL_RETURN, Leave.File)
	}
	return &ast.ReturnStmt{
		Range:  token.NewRange(Leave, p.previous()),
		Func:   p.currentFunction,
		Return: Leave,
		Value:  nil,
	}
}

func (p *parser) blockStatement(symbols *ast.SymbolTable) ast.Statement {
	colon := p.previous()
	if p.peek().Line() <= colon.Line() {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, "Nach einem Doppelpunkt muss eine neue Zeile beginnen", p.peek().File)
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

func (p *parser) expressionStatement() ast.Statement {
	return p.finishStatement(&ast.ExprStmt{Expr: p.expression()})
}

// entry for expression parsing
func (p *parser) expression() ast.Expression {
	return p.boolOR()
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
			Tok:      tok,
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
			Tok:      tok,
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
			Tok:      tok,
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
			Tok:      tok,
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
			Tok:      tok,
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
			Tok:      tok,
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
	for p.match(token.GRÖßER, token.KLEINER) {
		tok := p.previous()
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
			Tok:      tok,
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
		p.consume(token.IST)
	}
	return expr
}

func (p *parser) bitShift() ast.Expression {
	expr := p.term()
	for p.match(token.UM) {
		rhs := p.term()
		p.consumeN(token.BIT, token.NACH)
		if !p.match(token.LINKS, token.RECHTS) {
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "Links", "Rechts"), p.peek().File)
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
			Tok:      tok,
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
			Tok:      tok,
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
			Tok:      tok,
			Lhs:      expr,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return expr
}

func (p *parser) unary() ast.Expression {
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
				p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.previous().Range, fmt.Sprintf("Vor '%s' muss 'die' stehen", p.previous()), p.previous().File)
			case token.BETRAG:
				p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.previous().Range, "Vor 'Betrag' muss 'der' stehen", p.previous().File)
			}
			start = p.previous()
		}
		tok := p.previous()
		operator := ast.UN_ABS
		switch tok.Type {
		case token.BETRAG, token.GRÖßE, token.LÄNGE:
			p.consume(token.VON)
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
		case token.GRÖßE:
			operator = ast.UN_SIZE
		case token.LÄNGE:
			operator = ast.UN_LEN
		}
		rhs := p.unary()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(start),
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Operator: operator,
			Rhs:      rhs,
		}
	}
	return p.negate()
}

func (p *parser) negate() ast.Expression {
	if p.match(token.NEGATE) {
		tok := p.previous()
		rhs := p.unary()
		return &ast.UnaryExpr{
			Range: token.Range{
				Start: token.NewStartPos(tok),
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Operator: ast.UN_NEGATE,
			Rhs:      rhs,
		}
	}
	return p.power(nil)
}

// when called from unary() lhs might be a funcCall
// TODO: check precedence
func (p *parser) power(lhs ast.Expression) ast.Expression {
	if p.match(token.DIE) {
		lhs := p.unary()
		p.consumeN(token.DOT, token.WURZEL)
		tok := p.previous()
		p.consume(token.VON)
		// root is implemented as pow(degree, 1/radicant)
		expr := p.unary()

		return &ast.BinaryExpr{
			Range: token.Range{
				Start: expr.GetRange().Start,
				End:   lhs.GetRange().End,
			},
			Tok:      tok,
			Lhs:      expr,
			Operator: ast.BIN_POW,
			Rhs: &ast.BinaryExpr{
				Lhs: &ast.IntLit{
					Literal: lhs.Token(),
					Value:   1,
				},
				Tok:      tok,
				Operator: ast.BIN_DIV,
				Rhs:      lhs,
			},
		}
	}

	if p.matchN(token.DER, token.LOGARITHMUS) {
		tok := p.previous()
		p.consume(token.VON)
		numerus := p.expression()
		p.consumeN(token.ZUR, token.BASIS)
		rhs := p.unary()

		return &ast.BinaryExpr{
			Range: token.Range{
				Start: numerus.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Lhs:      numerus,
			Operator: ast.BIN_LOG,
			Rhs:      rhs,
		}
	}

	lhs = p.primary(lhs)
	for p.match(token.HOCH) {
		tok := p.previous()
		rhs := p.unary()
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Lhs:      lhs,
			Operator: ast.BIN_POW,
			Rhs:      rhs,
		}
	}
	return lhs
}

// when called from power() lhs might be a funcCall
func (p *parser) primary(lhs ast.Expression) ast.Expression {
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
				p.err(ddperror.SYN_MALFORMED_LITERAL, lit.Range, fmt.Sprintf("Das Kommazahlen Literal '%s' kann nicht gelesen werden", lit.Literal), lit.File)
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
			p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.previous().Range, ddperror.MsgGotExpected(p.previous().Literal, "ein Literal", "ein Name"), p.previous().File)
			lhs = &ast.BadExpr{
				Err: p.lastError,
				Tok: tok,
			}
		}
	}

	// TODO: check this with precedence and the else-if
	// 		 parsing might be incorrect
	// 		 remember to also check p.assigneable()
	// indexing
	if p.match(token.AN) {
		p.consumeN(token.DER, token.STELLE)
		tok := p.previous()
		rhs := p.primary(nil)
		lhs = &ast.BinaryExpr{
			Range: token.Range{
				Start: lhs.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Lhs:      lhs,
			Operator: ast.BIN_INDEX,
			Rhs:      rhs,
		}
	} else if p.match(token.VON) {
		tok := p.previous()
		operand := lhs
		mid := p.expression()
		p.consume(token.BIS)
		rhs := p.primary(nil)
		lhs = &ast.TernaryExpr{
			Range: token.Range{
				Start: operand.GetRange().Start,
				End:   rhs.GetRange().End,
			},
			Tok:      tok,
			Lhs:      operand,
			Mid:      mid,
			Rhs:      rhs,
			Operator: ast.TER_SLICE,
		}
	}

	// type-casting
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

// either ast.Ident or ast.Indexing
// p.previous() must be of Type token.IDENTIFIER
func (p *parser) assigneable() ast.Assigneable {
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

func (p *parser) grouping() ast.Expression {
	lParen := p.previous()
	innerExpr := p.expression()
	p.consume(token.RPAREN)

	return &ast.Grouping{
		Range:  token.NewRange(lParen, p.previous()),
		LParen: lParen,
		Expr:   innerExpr,
	}
}

func (p *parser) funcCall() ast.Expression {
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
	// it also returns all errors that might have occured while doing so
	checkAlias := func(mAlias *matchedAlias, typeSensitive bool) (map[string]ast.Expression, []ddperror.Error) {
		p.cur = start
		args := map[string]ast.Expression{}
		reported_errors := make([]ddperror.Error, 0)
		// this error handler collects its errors in reported_errors
		error_collector := func(err ddperror.Error) {
			reported_errors = append(reported_errors, err)
		}

		for i, l := 0, len(mAlias.alias.Tokens); i < l && mAlias.alias.Tokens[i].Type != token.EOF; i++ {
			tok := &mAlias.alias.Tokens[i]

			if tok.Type == token.ALIAS_PARAMETER {
				argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
				paramType := mAlias.alias.Args[argName]    // type of the current parameter

				pType := p.peek().Type
				// early return if a non-identifier expression is passed as reference
				if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
					return nil, reported_errors
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
				argParser := newParser(tokens, nil, error_collector) // create a new parser for this expression
				argParser.funcAliases = p.funcAliases                // it needs the functions aliases
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
					typ := p.typechecker.EvaluateSilent(arg) // evaluate the argument

					if typ != paramType.Type {
						arg = nil // arg and param types don't match
					} else if ass, ok := arg.(*ast.Indexing);                     // string-indexings may not be passed as char-reference
					paramType.IsReference && paramType.Type == ddptypes.Char() && // if the parameter is a char-reference
						ok { // and the argument is a indexing
						lhs := p.typechecker.EvaluateSilent(ass.Lhs)
						if lhs.Primitive == ddptypes.TEXT { // check if the lhs is a string
							arg = nil
						}
					}

					if arg == nil {
						return nil, reported_errors
					}
				}

				args[argName] = arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}
		p.cur = start + mAlias.actualLength
		return args, reported_errors
	}

	// sort the aliases in descending order
	// Stable so equal aliases stay in the order they were defined
	sort.SliceStable(matchedAliases, func(i, j int) bool {
		return len(matchedAliases[i].alias.Tokens) > len(matchedAliases[j].alias.Tokens)
	})

	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		if args, errs := checkAlias(&matchedAliases[i], true); args != nil {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
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
	args, errs := checkAlias(mostFitting, false)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return &ast.FuncCall{
		Range: token.NewRange(p.tokens[start], p.previous()),
		Tok:   p.tokens[start],
		Name:  mostFitting.alias.Func,
		Args:  args,
	}
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
			p.err(ddperror.SYN_MALFORMED_LITERAL, p.previous().Range, fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Buchstaben Literal", string(r)), p.previous().File)
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
				p.err(ddperror.SYN_MALFORMED_LITERAL, p.previous().Range, fmt.Sprintf("Ungültige Escape Sequenz '\\%s' im Text Literal", string(seq)), p.previous().File)
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
		return &ast.IntLit{Literal: lit, Value: val}
	} else {
		p.err(ddperror.SYN_MALFORMED_LITERAL, lit.Range, fmt.Sprintf("Das Zahlen Literal '%s' kann nicht gelesen werden", lit.Literal), lit.File)
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
func (p *parser) parseType() ddptypes.Type {
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"), p.peek().File)
		return ddptypes.Illegal() // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return tokenTypeToType(p.previous().Type)
	case token.BOOLEAN, token.TEXT:
		if !p.match(token.LISTE) {
			return tokenTypeToType(p.previous().Type)
		}
		return ddptypes.List(tokenTypeToType(p.peekN(-2).Type).Primitive)
	case token.ZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.List(ddptypes.ZAHL)
	case token.KOMMAZAHLEN:
		p.consume(token.LISTE)
		return ddptypes.List(ddptypes.KOMMAZAHL)
	case token.BUCHSTABEN:
		if p.peekN(-2).Type == token.EINEN || p.peekN(-2).Type == token.JEDEN { // edge case in function return types and for-range loops
			return ddptypes.Char()
		}
		p.consume(token.LISTE)
		return ddptypes.List(ddptypes.BUCHSTABE)
	}

	return ddptypes.Illegal() // unreachable
}

// parses tokens into a DDPType which must be a list type
// expects the next token to be the start of the type
// returns ILLEGAL and errors if no typename was found
func (p *parser) parseListType() ddptypes.Type {
	if !p.match(token.BOOLEAN, token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Listen-Typname"), p.peek().File)
		return ddptypes.Illegal()
	}

	if !p.consume(token.LISTE) {
		// report the error on the LISTE token, but still advance
		// because there is a valid token afterwards
		p.advance()
	}
	switch p.peekN(-2).Type {
	case token.BOOLEAN, token.TEXT:
		return ddptypes.List(tokenTypeToType(p.peekN(-2).Type).Primitive)
	case token.ZAHLEN:
		return ddptypes.List(ddptypes.ZAHL)
	case token.KOMMAZAHLEN:
		return ddptypes.List(ddptypes.KOMMAZAHL)
	case token.BUCHSTABEN:
		return ddptypes.List(ddptypes.BUCHSTABE)
	}

	return ddptypes.Illegal() // unreachable
}

// parses tokens into a DDPType and returns wether the type is a reference type
// expects the next token to be the start of the type
// returns ILLEGAL and errors if no typename was found
func (p *parser) parseReferenceType() (ddptypes.Type, bool) {
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.BOOLEAN, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"), p.peek().File)
		return ddptypes.Illegal(), false // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return tokenTypeToType(p.previous().Type), false
	case token.BOOLEAN, token.TEXT:
		if p.match(token.LISTE) {
			return ddptypes.List(tokenTypeToType(p.peekN(-2).Type).Primitive), false
		} else if p.match(token.LISTEN) {
			if !p.consume(token.REFERENZ) {
				// report the error on the REFERENZ token, but still advance
				// because there is a valid token afterwards
				p.advance()
			}
			return ddptypes.List(tokenTypeToType(p.peekN(-3).Type).Primitive), true
		} else if p.match(token.REFERENZ) {
			return ddptypes.Primitive(tokenTypeToType(p.peekN(-2).Type).Primitive), true
		}
		return tokenTypeToType(p.previous().Type), false
	case token.ZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.List(ddptypes.ZAHL), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.List(ddptypes.ZAHL), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Int(), true
	case token.KOMMAZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.List(ddptypes.KOMMAZAHL), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.List(ddptypes.KOMMAZAHL), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Float(), true
	case token.BUCHSTABEN:
		if p.match(token.LISTE) {
			return ddptypes.List(ddptypes.BUCHSTABE), false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.List(ddptypes.BUCHSTABE), true
		}
		p.consume(token.REFERENZ)
		return ddptypes.Char(), true
	}

	return ddptypes.Illegal(), false // unreachable
}

// parses tokens into a DDPType
// unlike parseType it may return void
// the error return is ILLEGAL
func (p *parser) parseReturnType() ddptypes.Type {
	if p.match(token.NICHTS) {
		return ddptypes.Void()
	}
	p.consumeAny(token.EINEN, token.EINE)
	return p.parseType()
}

// create a sub-scope of the current scope
func (p *parser) newScope() *ast.SymbolTable {
	return ast.NewSymbolTable(p.resolver.CurrentTable)
}

// set the current scope for the resolver and typechecker
func (p *parser) setScope(symbols *ast.SymbolTable) {
	p.resolver.CurrentTable, p.typechecker.CurrentTable = symbols, symbols
}

// exit the current scope of the resolver and typechecker
func (p *parser) exitScope() {
	p.resolver.CurrentTable, p.typechecker.CurrentTable = p.resolver.CurrentTable.Enclosing, p.typechecker.CurrentTable.Enclosing
}

// if the current tokenType is contained in types, advance
// returns wether we advanced or not
func (p *parser) match(types ...token.TokenType) bool {
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
func (p *parser) matchN(types ...token.TokenType) bool {
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
func (p *parser) consume(t token.TokenType) bool {
	if p.check(t) {
		p.advance()
		return true
	}

	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, t), p.peek().File)
	return false
}

// consume a series of tokens
func (p *parser) consumeN(t ...token.TokenType) bool {
	for _, v := range t {
		if !p.consume(v) {
			return false
		}
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

	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, toAnySlice(tokenTypes)...), p.peek().File)
	return false
}

// helper to report errors and enter panic mode
func (p *parser) err(code ddperror.Code, Range token.Range, msg string, file string) {
	if !p.panicMode {
		p.panicMode = true
		p.lastError = ddperror.New(code, Range, msg, file)
		p.errorHandler(p.lastError)
	}
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
func (p *parser) advance() token.Token {
	if !p.atEnd() {
		p.cur++
		return p.previous()
	}
	return p.peek() // return EOF
}

// returns the current token without advancing
func (p *parser) peek() token.Token {
	return p.tokens[p.cur]
}

// returns the n'th token starting from current without advancing
// p.peekN(0) is equal to p.peek()
func (p *parser) peekN(n int) token.Token {
	if p.cur+n >= len(p.tokens) || p.cur+n < 0 {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[p.cur+n]
}

// returns the token before peek()
func (p *parser) previous() token.Token {
	if p.cur < 1 {
		return token.Token{Type: token.ILLEGAL}
	}
	return p.tokens[p.cur-1]
}

// opposite of advance
func (p *parser) decrease() {
	if p.cur > 0 {
		p.cur--
	}
}

// retrives the last comment which comes before pos
// if their are no comments before pos nil is returned
func (p *parser) commentBeforePos(pos token.Position, file string) (result *token.Token) {
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
func (p *parser) commentAfterPos(pos token.Position, file string) (result *token.Token) {
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
func (p *parser) getLeadingOrTrailingComment() (result *token.Token) {
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

// counts all elements in the slice which fulfill the provided predicate function
func countElements[T any](elements []T, pred func(T) bool) (count int) {
	for _, v := range elements {
		if pred(v) {
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

// applies fun to every element in slice
func apply[T any](fun func(T), slice []T) {
	for i := range slice {
		fun(slice[i])
	}
}

func toAnySlice[T any](slice []T) []any {
	result := make([]any, len(slice))
	for i := range slice {
		result[i] = slice[i]
	}
	return result
}
