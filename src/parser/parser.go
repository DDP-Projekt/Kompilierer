package parser

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ast/resolver"
	"github.com/DDP-Projekt/Kompilierer/src/ast/typechecker"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state when parsing a .ddp file into an AST
type parser struct {
	tokens       []token.Token    // the tokens to parse (without comments)
	comments     []token.Token    // all the comments from the original tokens slice
	cur          int              // index of the current token
	errorHandler ddperror.Handler // a function to which errors are passed
	lastError    ddperror.Error   // latest reported error

	module                *ast.Module
	predefinedModules     map[string]*ast.Module            // modules that were passed as environment, might not all be used
	aliases               *at.Trie[*token.Token, ast.Alias] // all found aliases (+ inbuild aliases)
	typeNames             map[string]ddptypes.Type          // map of struct names to struct types
	currentFunction       string                            // function which is currently being parsed
	isCurrentFunctionBool bool                              // wether the current function returns a boolean
	panicMode             bool                              // flag to not report following errors
	errored               bool                              // wether the parser found an error
	resolver              *resolver.Resolver                // used to resolve every node directly after it has been parsed
	typechecker           *typechecker.Typechecker          // used to typecheck every node directly after it has been parsed
}

// returns a new parser, ready to parse the provided tokens
func newParser(name string, tokens []token.Token, modules map[string]*ast.Module, errorHandler ddperror.Handler) *parser {
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

	comments := make([]token.Token, 0)
	// filter the comments out
	i := 0
	for _, tok := range tokens {
		if tok.Type == token.COMMENT {
			comments = append(comments, tok)
		} else {
			tokens[i] = tok
			i++
		}
	}
	tokens = tokens[:i]

	parser := &parser{
		tokens:       tokens,
		comments:     comments,
		cur:          0,
		errorHandler: nil,
		module: &ast.Module{
			FileName:             name,
			Imports:              make([]*ast.ImportStmt, 0),
			ExternalDependencies: make(map[string]struct{}),
			Ast: &ast.Ast{
				Statements: make([]ast.Statement, 0),
				Comments:   comments,
				Symbols:    ast.NewSymbolTable(nil),
				Faulty:     false,
			},
			PublicDecls: make(map[string]ast.Declaration),
		},
		predefinedModules: modules,
		aliases:           at.New[*token.Token, ast.Alias](tokenEqual, tokenLess),
		typeNames:         make(map[string]ddptypes.Type),
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
	var err error
	if parser.resolver, err = resolver.New(parser.module, parser.errorHandler, name, &parser.panicMode); err != nil {
		panic(err)
	}
	if parser.typechecker, err = typechecker.New(parser.module, parser.errorHandler, name, &parser.panicMode); err != nil {
		panic(err)
	}

	return parser
}

// parse the provided tokens into an Ast
func (p *parser) parse() *ast.Module {
	defer parser_panic_wrapper(p)

	// main parsing loop
	for !p.atEnd() {
		if stmt := p.checkedDeclaration(); stmt != nil {
			p.module.Ast.Statements = append(p.module.Ast.Statements, stmt)
		}
	}

	p.module.Ast.Faulty = p.errored
	return p.module
}

// if an error was encountered we synchronize to a point where correct parsing is possible again
func (p *parser) synchronize() {
	p.panicMode = false

	// p.advance() // maybe this needs to stay?
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
				token.BUCHSTABEN, token.TEXT, token.WAHRHEITSWERT, token.FUNKTION:
				{
					return
				}
			}
		case token.DER:
			switch p.peekN(1).Type {
			case token.WAHRHEITSWERT, token.TEXT, token.BUCHSTABE, token.ALIAS:
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

	rawPath := ast.TrimStringLit(&importStmt.FileName)
	inclPath := ""

	if rawPath == "" {
		p.err(ddperror.MISC_INCLUDE_ERROR, importStmt.FileName.Range, "Bei Einbindungen muss ein Dateipfad angegeben werden")
		return
	}

	// resolve the actual file path
	var err error
	if strings.HasPrefix(rawPath, "Duden") {
		inclPath = filepath.Join(ddppath.InstallDir, rawPath) + ".ddp"
	} else {
		inclPath, err = filepath.Abs(filepath.Join(filepath.Dir(p.module.FileName), rawPath+".ddp"))
	}

	if err != nil {
		p.err(ddperror.SYN_MALFORMED_INCLUDE_PATH, importStmt.FileName.Range, fmt.Sprintf("Fehlerhafter Dateipfad '%s': \"%s\"", rawPath+".ddp", err.Error()))
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
			p.err(ddperror.MISC_INCLUDE_ERROR, importStmt.Range, fmt.Sprintf("Fehler beim einbinden von '%s': %s", rawPath+".ddp", err.Error()))
			return // return early on error
		} else {
			importStmt.Module.FileNameToken = &importStmt.FileName
			p.predefinedModules[inclPath] = importStmt.Module
		}
	} else { // we already included the module
		// circular import error
		if module == nil {
			p.err(ddperror.MISC_INCLUDE_ERROR, importStmt.Range, fmt.Sprintf("Zwei Module dürfen sich nicht gegenseitig einbinden! Das Modul '%s' versuchte das Modul '%s' einzubinden, während es von diesem Module eingebunden wurde", p.module.GetIncludeFilename(), rawPath+".ddp"))
			return // return early on error
		}

		importStmt.Module = module
	}

	ast.IterateImportedDecls(importStmt, func(_ string, decl ast.Declaration, tok token.Token) bool {
		if decl == nil {
			return true
		}

		// skip decls that are already defined
		// the resolver will error here
		_, exists, _ := p.scope().LookupDecl(decl.Name())

		if exists {
			return true // continue
		}

		var aliases []ast.Alias
		needAddAliases := true
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			aliases = append(aliases, toInterfaceSlice[*ast.FuncAlias, ast.Alias](decl.Aliases)...)
		case *ast.StructDecl:
			aliases = append(aliases, toInterfaceSlice[*ast.StructAlias, ast.Alias](decl.Aliases)...)
			p.typeNames[decl.Name()] = decl.Type
		default: // for VarDecls or BadDecls we don't need to add any aliases
			needAddAliases = false
		}

		// VarDecls don't have aliases
		if !needAddAliases {
			return true // continue
		}
		p.addAliases(aliases, tok.Range)
		return true
	})
}

// calls p.declaration and resolves and typechecks it
func (p *parser) checkStatement(stmt ast.Statement) {
	if stmt == nil {
		p.panic("nil statement passed to checkStatement")
	}
	if importStmt, ok := stmt.(*ast.ImportStmt); ok {
		p.resolveModuleImport(importStmt)
	}
	p.resolver.ResolveNode(stmt)      // resolve symbols in it (variables, functions, ...)
	p.typechecker.TypecheckNode(stmt) // typecheck the node
}

func (p *parser) checkedDeclaration() ast.Statement {
	stmt := p.declaration() // parse the node
	if stmt != nil {        // nil check, for alias declarations that aren't Ast Nodes
		p.checkStatement(stmt)
	}
	if p.panicMode { // synchronize the parsing if we are in panic mode
		p.synchronize()
	}
	return stmt
}

// entry point for the recursive descent parsing
func (p *parser) declaration() ast.Statement {
	if p.match(token.DER, token.DIE, token.DAS, token.WIR) { // might indicate a function, variable or struct
		if p.previous().Type == token.WIR {
			return &ast.DeclStmt{Decl: p.structDeclaration()}
		}

		n := -1
		if p.match(token.OEFFENTLICHE) {
			n = -2
		}

		switch t := p.peek().Type; t {
		case token.ALIAS:
			p.advance()
			return p.aliasDecl()
		case token.FUNKTION:
			p.advance()
			return &ast.DeclStmt{Decl: p.funcDeclaration(n - 1)}
		default:
			if p.isTypeName(p.peek()) {
				p.advance()
				return &ast.DeclStmt{Decl: p.varDeclaration(n-1, false)}
			}
		}

		p.decrease() // decrease, so expressionStatement() can recognize it as expression
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
		tok := p.previous() // wahr or falsch token
		// parse possible wahr/falsch wenn syntax
		if p.match(token.COMMA) {
			p.consume(token.WENN)
			// if it is false, we add a unary bool-negate into the ast
			if tok.Type == token.FALSE {
				rhs := p.expression() // the actual boolean expression after falsch wenn, which is negated
				expr = &ast.UnaryExpr{
					Range: token.Range{
						Start: token.NewStartPos(tok),
						End:   rhs.GetRange().End,
					},
					Tok:      *tok,
					Operator: ast.UN_NOT,
					Rhs:      rhs,
				}
			} else {
				expr = p.expression() // wahr wenn simply becomes a normal expression
			}
		} else { // no wahr/falsch wenn, only a boolean literal
			p.decrease() // decrease, so expression() can recognize the literal
			expr = p.expression()
		}
	} else {
		expr = p.expression() // no wahr/falsch, so a normal expression
	}

	return expr
}

// parses a variable declaration
// startDepth is the int passed to p.peekN(n) to get to the DER/DIE token of the declaration
// isField indicates that this declaration should be parsed as a struct field
func (p *parser) varDeclaration(startDepth int, isField bool) ast.Declaration {
	begin := p.peekN(startDepth) // Der/Die/Das
	comment := p.commentBeforePos(begin.Range.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE || p.peekN(startDepth+1).Type == token.OEFFENTLICHEN
	p.decrease()
	type_start := p.previous()
	typ := p.parseType()
	if typ == nil {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, token.NewRange(type_start, p.previous()), fmt.Sprintf("Invalider Typname %s", p.previous()))
	} else {
		getArticle := func(gender ddptypes.GrammaticalGender) token.TokenType {
			switch gender {
			case ddptypes.MASKULIN:
				if isField {
					return token.DEM
				}
				return token.DER
			case ddptypes.FEMININ:
				if isField {
					return token.DER
				}
				return token.DIE
			case ddptypes.NEUTRUM:
				if isField {
					return token.DEM
				}
				return token.DAS
			}
			return token.ILLEGAL // unreachable
		}

		if article := getArticle(typ.Gender()); begin.Type != article {
			p.err(ddperror.SYN_GENDER_MISMATCH, begin.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", article))
		}
	}

	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.module.FileName,
				Msg:   "Es wurde ein Variablen Name erwartet",
			},
			Tok: *p.peek(),
			Mod: p.module,
		}
	}

	name := p.previous()
	if isField {
		p.consume(token.MIT, token.STANDARDWERT)
	} else {
		p.consume(token.IST)
	}
	var expr ast.Expression

	if typ != ddptypes.WAHRHEITSWERT && ddptypes.IsList(typ) { // TODO: fix this with function calls and groupings
		expr = p.expression()
		if p.match(token.COUNT_MAL) {
			value := p.expression()
			expr_tok := expr.Token()
			expr = &ast.ListLit{
				Tok:    expr.Token(),
				Range:  token.NewRange(&expr_tok, p.previous()),
				Type:   typ.(ddptypes.ListType),
				Values: nil,
				Count:  expr,
				Value:  value,
			}
		}
	} else {
		expr = p.assignRhs()
	}

	if !isField {
		p.consume(token.DOT)
	}
	// prefer trailing comments as long as they are on the same line
	if trailingComment := p.commentAfterPos(p.previous().Range.End); trailingComment != nil && trailingComment.Range.Start.Line == p.previous().Range.End.Line {
		comment = trailingComment
	}

	return &ast.VarDecl{
		Range:      token.NewRange(begin, p.previous()),
		CommentTok: comment,
		Type:       typ,
		NameTok:    *name,
		IsPublic:   isPublic,
		Mod:        p.module,
		InitVal:    expr,
	}
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

	perr := func(code ddperror.Code, Range token.Range, msg string) {
		p.err(code, Range, msg)
		valid = false
	}

	begin := p.peekN(startDepth)
	comment := p.commentBeforePos(begin.Range.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE

	Funktion := p.previous() // save the token
	// we need a name, so bailout if none is provided
	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.New(ddperror.SYN_EXPECTED_IDENTIFIER, token.NewRange(begin, p.peek()), "Es wurde ein Funktions Name erwartet", p.module.FileName),
			Tok: *p.peek(),
			Mod: p.module,
		}
	}
	name := p.previous()

	// early error report if the name is already used
	if _, existed, _ := p.scope().LookupDecl(name.Literal); existed { // insert the name of the current function
		p.err(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, ddperror.MsgNameAlreadyExists(name.Literal))
	}

	// parse the parameter declaration
	// parameter names and types are declared seperately
	var (
		paramNames    []token.Token
		paramTypes    []ddptypes.ParameterType
		paramComments []*token.Token
	)
	if p.match(token.MIT) { // the function takes at least 1 parameter
		singleParameter := true
		if p.matchN(token.DEN, token.PARAMETERN) {
			singleParameter = false
		} else if !p.matchN(token.DEM, token.PARAMETER) {
			perr(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "'de[n/m] Parameter[n]'"))
		}
		validate(p.consume(token.IDENTIFIER))
		paramNames = append(paramNames, *p.previous()) // append the first parameter name
		paramComments = append(paramComments, p.getLeadingOrTrailingComment())
		if !singleParameter {
			// helper function to avoid too much repitition
			addParamName := func(name *token.Token) {
				if containsLiteral(paramNames, name.Literal) { // check that each parameter name is unique
					perr(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, fmt.Sprintf("Ein Parameter mit dem Namen '%s' ist bereits vorhanden", name.Literal))
				}
				paramNames = append(paramNames, *name)                                 // append the parameter name
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
				if !p.consume(token.UND, token.IDENTIFIER) {
					perr(ddperror.SYN_EXPECTED_IDENTIFIER, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "der letzte Parameter (und <Name>)")+"\nMeintest du vorher vielleicht 'dem Parameter' anstatt 'den Parametern'?")
				}
				addParamName(p.previous())
			}
		}
		// parse the types of the parameters
		validate(p.consume(token.VOM, token.TYP))
		firstType, ref := p.parseReferenceType()
		validate(firstType != nil)
		paramTypes = append(paramTypes, ddptypes.ParameterType{Type: firstType, IsReference: ref}) // append the first parameter type
		if !singleParameter {
			// helper function to avoid too much repitition
			addType := func() {
				// validate the parameter type and append it
				typ, ref := p.parseReferenceType()
				validate(typ != nil)
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
		perr(ddperror.SEM_PARAM_NAME_TYPE_COUNT_MISMATCH, token.NewRange(&paramNames[0], p.previous()), fmt.Sprintf("Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)", len(paramNames), len(paramTypes)))
	}

	// parse the return type declaration
	validate(p.consume(token.GIBT))
	Typ := p.parseReturnType()
	if Typ == nil {
		valid = false
	}
	if Typ == ddptypes.WAHRHEITSWERT {
		p.isCurrentFunctionBool = true
	}

	validate(p.consume(token.ZURÜCK, token.COMMA))
	bodyStart := -1
	definedIn := &token.Token{Type: token.ILLEGAL}
	if p.matchN(token.MACHT, token.COLON) {
		bodyStart = p.cur                             // save the body start-position for later, we first need to parse aliases to enable recursion
		indent := p.previous().Indent + 1             // indentation level of the function body
		for p.peek().Indent >= indent && !p.atEnd() { // advance to the alias definitions by checking the indentation
			p.advance()
		}
	} else {
		validate(p.consume(token.IST, token.IN, token.STRING, token.DEFINIERT))
		definedIn = p.peekN(-2)
		switch filepath.Ext(ast.TrimStringLit(definedIn)) {
		case ".c", ".lib", ".a", ".o":
		default:
			perr(ddperror.SEM_EXPECTED_LINKABLE_FILEPATH, definedIn.Range, fmt.Sprintf("Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden", definedIn.Literal))
		}
	}

	// parse the alias definitions before the body to enable recursion
	validate(p.consume(token.UND, token.KANN, token.SO, token.BENUTZT, token.WERDEN, token.COLON, token.STRING)) // at least 1 alias is required
	rawAliases := make([]*token.Token, 0)
	if p.previous().Type == token.STRING {
		rawAliases = append(rawAliases, p.previous())
	}
	// append the raw aliases
	for (p.match(token.COMMA) || p.match(token.ODER)) && p.peek().Indent > 0 && !p.atEnd() {
		if p.consume(token.STRING) {
			rawAliases = append(rawAliases, p.previous())
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
	funcAliases := make([]*ast.FuncAlias, 0)
	funcAliasTokens := make([][]*token.Token, 0)
	for _, v := range rawAliases {
		// scan the raw alias withouth the ""
		didError := false
		errHandleWrapper := func(err ddperror.Error) { didError = true; p.errorHandler(err) }
		if alias, err := scanner.ScanAlias(*v, errHandleWrapper); err == nil && !didError {
			if len(alias) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
				p.err(ddperror.SEM_MALFORMED_ALIAS, v.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if err := p.validateFunctionAlias(alias, paramNames, paramTypes); err == nil { // check that the alias fits the function
				if ok, isFun, existingAlias, pTokens := p.aliasExists(alias); ok {
					p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, v.Range, ddperror.MsgAliasAlreadyExists(v.Literal, existingAlias.Decl().Name(), isFun))
				} else {
					funcAliases = append(funcAliases, &ast.FuncAlias{Tokens: alias, Original: *v, Func: nil, Args: paramTypesMap})
					funcAliasTokens = append(funcAliasTokens, pTokens)
				}
			} else {
				p.errVal(*err)
			}
		}
	}

	aliasEnd := p.cur // save the end of the function declaration for later

	if !ast.IsGlobalScope(p.scope()) {
		perr(ddperror.SEM_NON_GLOBAL_FUNCTION, begin.Range, "Es können nur globale Funktionen deklariert werden")
	}

	if !valid {
		p.cur = aliasEnd
		return &ast.BadDecl{
			Err: p.lastError,
			Tok: *Funktion,
			Mod: p.module,
		}
	}

	decl := &ast.FuncDecl{
		Range:         token.NewRange(begin, p.previous()),
		CommentTok:    comment,
		Tok:           *begin,
		NameTok:       *name,
		IsPublic:      isPublic,
		Mod:           p.module,
		ParamNames:    paramNames,
		ParamTypes:    paramTypes,
		ParamComments: paramComments,
		Type:          Typ,
		Body:          nil,
		ExternFile:    *definedIn,
		Aliases:       funcAliases,
	}

	for i := range funcAliases {
		funcAliases[i].Func = decl
		p.aliases.Insert(funcAliasTokens[i], funcAliases[i])
	}

	// parse the body after the aliases to enable recursion
	var body *ast.BlockStmt = nil
	if bodyStart != -1 {
		p.cur = bodyStart // go back to the body
		p.currentFunction = name.Literal

		bodyTable := p.newScope() // temporary symbolTable for the function parameters
		globalScope := bodyTable.Enclosing
		// insert the name of the current function
		if existed := globalScope.InsertDecl(p.currentFunction, decl); !existed && decl.IsPublic {
			p.module.PublicDecls[decl.Name()] = decl
		}
		// add the parameters to the table
		for i, l := 0, len(paramNames); i < l; i++ {
			bodyTable.InsertDecl(paramNames[i].Literal,
				&ast.VarDecl{
					NameTok:    paramNames[i],
					IsPublic:   false,
					Mod:        p.module,
					Type:       paramTypes[i].Type,
					Range:      token.NewRange(&paramNames[i], &paramNames[i]),
					CommentTok: paramComments[i],
				},
			)
		}
		body = p.blockStatement(bodyTable).(*ast.BlockStmt) // parse the body with the parameters in the current table
		decl.Body = body

		// check that the function has a return statement if it needs one
		if !ddptypes.IsVoid(Typ) { // only if the function does not return void
			if len(body.Statements) < 1 { // at least the return statement is needed
				perr(ddperror.SEM_MISSING_RETURN, body.Range, ddperror.MSG_MISSING_RETURN)
			} else {
				// the last statement must be a return statement
				lastStmt := body.Statements[len(body.Statements)-1]
				if _, ok := lastStmt.(*ast.ReturnStmt); !ok {
					perr(ddperror.SEM_MISSING_RETURN, token.NewRange(p.previous(), p.previous()), ddperror.MSG_MISSING_RETURN)
				}
			}
		}
	} else { // the function is defined in an extern file
		// insert the name of the current function
		if existed := p.scope().InsertDecl(name.Literal, decl); !existed && decl.IsPublic {
			p.module.PublicDecls[decl.Name()] = decl
		}
		p.module.ExternalDependencies[ast.TrimStringLit(&decl.ExternFile)] = struct{}{} // add the extern declaration
	}

	p.currentFunction = ""
	p.cur = aliasEnd // go back to the end of the function to continue parsing

	return decl
}

func isAliasExpr(t token.Token) bool    { return t.Type == token.ALIAS_PARAMETER } // helper to check for parameters
func isIllegalToken(t token.Token) bool { return t.Type == token.ILLEGAL }         // helper to check for illegal tokens

// helper for funcDeclaration to check that every parameter is provided exactly once
// and that no ILLEGAL tokens are present
func (p *parser) validateFunctionAlias(aliasTokens []token.Token, paramNames []token.Token, paramTypes []ddptypes.ParameterType) *ddperror.Error {
	if count := countElements(aliasTokens, isAliasExpr); count != len(paramNames) { // validate that the alias contains as many parameters as the function
		err := ddperror.New(ddperror.SEM_ALIAS_BAD_NUM_ARGS,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			fmt.Sprintf("Der Alias braucht %d Parameter aber hat %d", len(paramNames), count),
			p.module.FileName,
		)
		return &err
	}
	if countElements(aliasTokens, isIllegalToken) > 0 { // validate that the alias does not contain illegal tokens
		err := ddperror.New(
			ddperror.SEM_MALFORMED_ALIAS,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			"Der Alias enthält ungültige Symbole",
			p.module.FileName,
		)
		return &err
	}
	nameSet := map[string]ddptypes.ParameterType{} // set that holds the parameter names contained in the alias and their corresponding type
	for i, v := range paramNames {
		if i < len(paramTypes) {
			nameSet[v.Literal] = paramTypes[i]
		}
	}
	// validate that each parameter is contained in the alias exactly once
	// and fill in the AliasInfo
	for i, v := range aliasTokens {
		if isAliasExpr(v) {
			k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
			if argTyp, ok := nameSet[k]; ok {
				aliasTokens[i].AliasInfo = &argTyp
				delete(nameSet, k)
			} else {
				err := ddperror.New(ddperror.SEM_ALIAS_BAD_NUM_ARGS,
					token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
					fmt.Sprintf("Der Alias enthält den Parameter %s mehrmals", k),
					p.module.FileName,
				)
				return &err
			}
		}
	}
	return nil
}

// helper for structDeclaration to check that every field is provided once at max
// and that no ILLEGAL tokens are present
// fields should not contain bad decls
// returns wether the alias is valid and its arguments
func (p *parser) validateStructAlias(aliasTokens []token.Token, fields []*ast.VarDecl) (*ddperror.Error, map[string]ddptypes.Type) {
	if count := countElements(aliasTokens, isAliasExpr); count > len(fields) { // validate that the alias contains as many parameters as the struct
		err := ddperror.New(ddperror.SEM_ALIAS_BAD_NUM_ARGS,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			fmt.Sprintf("Der Alias erwartet Maximal %d Parameter aber hat %d", len(fields), count),
			p.module.FileName,
		)
		return &err, nil
	}
	if countElements(aliasTokens, isIllegalToken) > 0 { // validate that the alias does not contain illegal tokens
		err := ddperror.New(
			ddperror.SEM_MALFORMED_ALIAS,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			"Der Alias enthält ungültige Symbole",
			p.module.FileName,
		)
		return &err, nil
	}
	nameSet := map[string]ddptypes.ParameterType{} // set that holds the parameter names contained in the alias and their corresponding type
	args := map[string]ddptypes.Type{}             // the arguments of the alias
	for _, v := range fields {
		nameSet[v.Name()] = ddptypes.ParameterType{
			Type:        v.Type,
			IsReference: false, // fields are never references
		}
		args[v.Name()] = v.Type
	}
	// validate that each parameter is contained in the alias once at max
	// and fill in the AliasInfo
	for i, v := range aliasTokens {
		if isAliasExpr(v) {
			k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
			if argTyp, ok := nameSet[k]; ok {
				aliasTokens[i].AliasInfo = &argTyp
				delete(nameSet, k)
			} else {
				err := ddperror.New(ddperror.SEM_ALIAS_BAD_NUM_ARGS,
					token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
					fmt.Sprintf("Der Alias enthält den Parameter %s mehrmals", k),
					p.module.FileName,
				)
				return &err, nil
			}
		}
	}
	return nil, args
}

// helper for structDeclaration
func varDeclsToFields(decls []*ast.VarDecl) []ddptypes.StructField {
	result := make([]ddptypes.StructField, 0, len(decls))
	for _, v := range decls {
		result = append(result, ddptypes.StructField{
			Name: v.Name(),
			Type: v.Type,
		})
	}
	return result
}

func (p *parser) structDeclaration() ast.Declaration {
	begin := p.previous() // Wir
	comment := p.commentBeforePos(begin.Range.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}

	p.consume(token.NENNEN, token.DIE)
	isPublic := p.match(token.OEFFENTLICHE)
	p.consume(token.KOMBINATION, token.AUS)

	var fields []ast.Declaration
	indent := begin.Indent + 1
	for p.peek().Indent >= indent && !p.atEnd() {
		p.consumeAny(token.DER, token.DEM)
		n := -1
		if p.match(token.OEFFENTLICHEN) {
			n = -2
		}
		p.advance()
		fields = append(fields, p.varDeclaration(n-1, true))
		if !p.consume(token.COMMA) {
			p.advance()
		}
	}

	p.consumeAny(token.EINEN, token.EINE, token.EIN)
	gender := ddptypes.INVALID
	switch p.previous().Type {
	case token.EINEN:
		gender = ddptypes.MASKULIN
	case token.EINE:
		gender = ddptypes.FEMININ
	case token.EIN:
		gender = ddptypes.NEUTRUM
	}

	if !p.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.module.FileName,
				Msg:   "Es wurde ein Strukturen Name erwartet",
			},
			Tok: *p.peek(),
			Mod: p.module,
		}
	}
	name := p.previous()

	p.consume(token.COMMA, token.UND, token.ERSTELLEN, token.SIE, token.SO, token.COLON, token.STRING)
	var rawAliases []*token.Token
	if p.previous().Type == token.STRING {
		rawAliases = append(rawAliases, p.previous())
	}
	for p.match(token.COMMA) || p.match(token.ODER) && p.peek().Indent > 0 && !p.atEnd() {
		if p.consume(token.STRING) {
			rawAliases = append(rawAliases, p.previous())
		}
	}

	var structAliases []*ast.StructAlias
	var structAliasTokens [][]*token.Token
	fieldsForValidation := toInterfaceSlice[ast.Declaration, *ast.VarDecl](
		filterSlice(fields, func(decl ast.Declaration) bool { _, ok := decl.(*ast.VarDecl); return ok }),
	)
	for _, rawAlias := range rawAliases {
		didError := false
		errHandleWrapper := func(err ddperror.Error) { didError = true; p.errorHandler(err) }
		if aliasTokens, err := scanner.ScanAlias(*rawAlias, errHandleWrapper); err == nil && !didError {
			if len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
				p.err(ddperror.SEM_MALFORMED_ALIAS, rawAlias.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if err, args := p.validateStructAlias(aliasTokens, fieldsForValidation); err == nil {
				if ok, isFunc, existingAlias, pTokens := p.aliasExists(aliasTokens); ok {
					p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, rawAlias.Range, ddperror.MsgAliasAlreadyExists(rawAlias.Literal, existingAlias.Decl().Name(), isFunc))
				} else {
					structAliases = append(structAliases, &ast.StructAlias{Tokens: aliasTokens, Original: *rawAlias, Struct: nil, Args: args})
					structAliasTokens = append(structAliasTokens, pTokens)
				}
			} else {
				p.errVal(*err)
			}
		}
	}

	structType := &ddptypes.StructType{
		Name:       name.Literal,
		GramGender: gender,
		Fields:     varDeclsToFields(fieldsForValidation),
	}

	decl := &ast.StructDecl{
		Range:      token.NewRange(begin, p.previous()),
		CommentTok: comment,
		Tok:        *begin,
		NameTok:    *name,
		IsPublic:   isPublic,
		Mod:        p.module,
		Fields:     fields,
		Type:       structType,
		Aliases:    structAliases,
	}

	for i := range structAliases {
		structAliases[i].Struct = decl
		p.aliases.Insert(structAliasTokens[i], structAliases[i])
	}

	if _, exists := p.typeNames[decl.Name()]; !exists {
		p.typeNames[decl.Name()] = decl.Type
	}

	return decl
}

// TODO: add support for struct aliases
func (p *parser) aliasDecl() ast.Statement {
	begin := p.peekN(-2)
	if begin.Type != token.DER {
		p.err(ddperror.SYN_GENDER_MISMATCH, begin.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", token.DER))
	}
	p.consume(token.STRING)
	aliasTok := p.previous()
	p.consume(token.STEHT, token.FÜR, token.DIE, token.FUNKTION, token.IDENTIFIER)
	fun := p.previous()

	decl, ok, isVar := p.scope().LookupDecl(fun.Literal)
	if !ok {
		p.err(ddperror.SEM_NAME_UNDEFINED, fun.Range, fmt.Sprintf("Der Name %s wurde noch nicht deklariert", fun.Literal))
		return nil
	} else if isVar {
		p.err(ddperror.SEM_BAD_NAME_CONTEXT, fun.Range, fmt.Sprintf("Der Name %s steht für eine Variable und nicht für eine Funktion", fun.Literal))
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
	var pTokens []*token.Token
	if aliasTokens, err := scanner.ScanAlias(*aliasTok, func(err ddperror.Error) { p.err(err.Code, err.Range, err.Msg) }); err == nil && len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
		p.err(ddperror.SEM_MALFORMED_ALIAS, aliasTok.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
	} else if err := p.validateFunctionAlias(aliasTokens, funDecl.ParamNames, funDecl.ParamTypes); err == nil { // check that the alias fits the function
		if ok, isFun, existingAlias, toks := p.aliasExists(aliasTokens); ok {
			p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, aliasTok.Range, ddperror.MsgAliasAlreadyExists(aliasTok.Literal, existingAlias.Decl().Name(), isFun))
		} else {
			alias = &ast.FuncAlias{Tokens: aliasTokens, Original: *aliasTok, Func: funDecl, Args: paramTypes}
			pTokens = toks
		}
	} else {
		p.errVal(*err)
	}

	p.consume(token.DOT)

	if begin.Indent > 0 {
		p.err(ddperror.SEM_ALIAS_MUST_BE_GLOBAL, token.NewRange(begin, p.previous()), "Ein Alias darf nur im globalen Bereich deklariert werden!")
		return &ast.BadStmt{
			Err: p.lastError,
			Tok: *begin,
		}
	} else if alias != nil {
		p.aliases.Insert(pTokens, alias)
		funDecl.Aliases = append(funDecl.Aliases, alias)
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
		return p.voidReturnOrBreak()
	case token.FAHRE:
		p.consume(token.FAHRE)
		return p.continueStatement()
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
			FileName:        *p.previous(),
			ImportedSymbols: nil,
		}
	} else if p.match(token.IDENTIFIER) {
		importedSymbols := []token.Token{*p.previous()}
		if p.peek().Type != token.AUS {
			if p.match(token.UND) {
				p.consume(token.IDENTIFIER)
				importedSymbols = append(importedSymbols, *p.previous())
			} else {
				for p.match(token.COMMA) {
					if p.consume(token.IDENTIFIER) {
						importedSymbols = append(importedSymbols, *p.previous())
					}
				}
				if p.consume(token.UND) && p.consume(token.IDENTIFIER) {
					importedSymbols = append(importedSymbols, *p.previous())
				}
			}
		}
		p.consume(token.AUS)
		if p.consume(token.STRING) {
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
	} else {
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "ein Text Literal oder ein Name"))
		return &ast.BadStmt{
			Tok: *p.peek(),
			Err: p.lastError,
		}
	}
	p.consume(token.EIN, token.DOT)
	stmt.Range = token.NewRange(binde, p.previous())
	return stmt
}

// either consumes the neccesery . or adds a postfix do-while or repeat
func (p *parser) finishStatement(stmt ast.Statement) ast.Statement {
	if p.match(token.DOT) || p.panicMode {
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
		p.consume(token.DOT)
		return stmt
	}

	if !p.match(token.COUNT_MAL) {
		count_tok := count.Token()
		p.err(ddperror.SYN_UNEXPECTED_TOKEN, count.GetRange(),
			fmt.Sprintf("%s\nWolltest du vor %s vielleicht einen Punkt setzten?",
				ddperror.MsgGotExpected(p.previous(), token.COUNT_MAL), &count_tok,
			),
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
		p.consume(token.DOT)
		typ := p.typechecker.EvaluateSilent(varName)
		operator := ast.UN_NEGATE
		if typ == ddptypes.WAHRHEITSWERT {
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
		p.consume(token.DURCH)
	} else {
		p.consume(token.UM)
	}

	operand := p.expression()

	if tok.Type == token.VERSCHIEBE {
		p.consume(token.BIT, token.NACH)
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
		p.consume(token.DOT)
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
	p.consume(token.IST)
	expr := p.assignRhs() // parse the expression
	// validate that the expression is a literal
	switch expr := expr.(type) {
	case *ast.IntLit, *ast.FloatLit, *ast.BoolLit, *ast.StringLit, *ast.CharLit, *ast.ListLit:
	default:
		if typ := p.typechecker.Evaluate(ident); typ != ddptypes.WAHRHEITSWERT {
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
	p.consume(token.IN)
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
	p.consume(token.COMMA)      // must be boolean, so an ist is required for grammar
	var Then ast.Statement
	thenScope := p.newScope()
	if p.match(token.DANN) { // with dann: the body is a block statement
		p.consume(token.COLON)
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
					Colon:      *_else,
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
		If:        *If,
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
	p.resolver.LoopDepth++
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
	p.consume(token.COLON)
	p.resolver.LoopDepth++
	body := p.blockStatement(nil)
	p.resolver.LoopDepth--
	p.consume(token.SOLANGE)
	condition := p.expression()
	p.consume(token.DOT)
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
	p.consume(token.COLON)
	p.resolver.LoopDepth++
	body := p.blockStatement(nil)
	p.resolver.LoopDepth--
	count := p.expression()
	p.consume(token.COUNT_MAL, token.DOT)
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
	getPronoun := func(gender ddptypes.GrammaticalGender) token.TokenType {
		switch gender {
		case ddptypes.MASKULIN:
			return token.JEDEN
		case ddptypes.FEMININ:
			return token.JEDE
		case ddptypes.NEUTRUM:
			return token.JEDES
		}
		return token.ILLEGAL // unreachable
	}

	For := p.previous()
	p.consumeAny(token.JEDE, token.JEDEN, token.JEDES)
	pronoun_tok := p.previous()
	TypeTok := p.peek()
	Typ := p.parseType()
	if Typ != nil {
		if pronoun := getPronoun(Typ.Gender()); pronoun != pronoun_tok.Type {
			p.err(ddperror.SYN_GENDER_MISMATCH, pronoun_tok.Range, fmt.Sprintf("Falsches Pronomen, meintest du %s?", pronoun))
		}
	}

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
			CommentTok: iteratorComment,
			Type:       Typ,
			NameTok:    *Ident,
			IsPublic:   false,
			Mod:        p.module,
			InitVal:    from,
		}
		p.consume(token.BIS)
		to := p.expression()                            // end of the counter
		var step ast.Expression = &ast.IntLit{Value: 1} // step-size (default = 1)
		if Typ == ddptypes.KOMMAZAHL {
			step = &ast.FloatLit{Value: 1.0}
		}
		if p.match(token.MIT) {
			p.consume(token.SCHRITTGRÖßE)
			step = p.expression() // custom specified step-size
		}
		p.consume(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		p.resolver.LoopDepth++
		if p.match(token.MACHE) { // body is a block statement
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
	} else if p.match(token.IN) {
		In := p.expression()
		initializer := &ast.VarDecl{
			Range: token.Range{
				Start: token.NewStartPos(TypeTok),
				End:   In.GetRange().End,
			},
			Type:     Typ,
			NameTok:  *Ident,
			IsPublic: false,
			Mod:      p.module,
			InitVal:  In,
		}
		p.consume(token.COMMA)
		var Body *ast.BlockStmt
		bodyTable := p.newScope()                        // temporary symbolTable for the loop variable
		bodyTable.InsertDecl(Ident.Literal, initializer) // add the loop variable to the table
		p.resolver.LoopDepth++
		if p.match(token.MACHE) { // body is a block statement
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
		expr = p.assignRhs()
	} else {
		expr = p.expression()
	}

	p.consume(token.ZURÜCK, token.DOT)
	rnge := token.NewRange(Return, p.previous())
	if p.currentFunction == "" {
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
	p.consume(token.DIE)
	if p.match(token.SCHLEIFE) {
		p.consume(token.DOT)
		return &ast.BreakContinueStmt{
			Range: token.NewRange(Leave, p.previous()),
			Tok:   *Leave,
		}
	}

	p.consume(token.FUNKTION, token.DOT)
	rnge := token.NewRange(Leave, p.previous())
	if p.currentFunction == "" {
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
	p.consume(token.MIT, token.DER, token.SCHLEIFE, token.FORT, token.DOT)
	return &ast.BreakContinueStmt{
		Range: token.NewRange(Continue, p.previous()),
		Tok:   *Continue,
	}
}

func (p *parser) blockStatement(symbols *ast.SymbolTable) ast.Statement {
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

func (p *parser) expressionStatement() ast.Statement {
	return p.finishStatement(&ast.ExprStmt{Expr: p.expression()})
}

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

func (p *parser) alias() ast.Expression {
	start := p.cur // save start position to restore the state if no alias was recognized

	// used as a map[int]int abusing the fact that node_index is incremental
	// keys are the indices where start_indices[i] < start_indices[i+1]
	// example order: 0, 1, 2, 5, 7, 9
	start_indices := make([]int, 0, 30)
	matchedAliases := p.aliases.Search(func(node_index int, tok *token.Token) (*token.Token, bool) {
		// the if statement below is a more efficient map[int]int implementation
		// abusing the fact that node_index is incremental
		if node_index < len(start_indices) { // key is already in the map
			// -1 is a placeholder for an unused keys
			if i := start_indices[node_index]; i == -1 {
				start_indices[node_index] = p.cur // assign the value
			} else {
				p.cur = i // the value is valid so use it
			}
		} else { // key is not in the map
			// we need to insert n more keys
			n := node_index - len(start_indices) + 1
			// placeholder -1 in every key
			for i := 0; i < n; i++ {
				start_indices = append(start_indices, -1)
			}
			// assign the value to the new key
			start_indices[node_index] = p.cur
		}

		if tok.Type == token.ALIAS_PARAMETER {
			switch t := p.peek(); t.Type {
			case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER, token.SYMBOL:
				p.advance()
				return tok, true
			case token.NEGATE:
				p.advance()
				if !p.match(token.INT, token.FLOAT, token.IDENTIFIER) {
					return nil, false
				}
				return tok, true
			case token.LPAREN:
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
				if p.atEnd() {
					return nil, false
				}
				return tok, true
			}
		}
		return p.advance(), true
	})

	if len(matchedAliases) == 0 { // check if any alias was matched
		p.cur = start
		return nil // no alias -> no function call
	}

	// sort the aliases in descending order
	// Stable so equal aliases stay in the order they were defined
	sort.SliceStable(matchedAliases, func(i, j int) bool {
		return len(matchedAliases[i].GetTokens()) > len(matchedAliases[j].GetTokens())
	})

	// a argument that was already parsed
	type cachedArg struct {
		Arg     ast.Expression   // expression (might be an assignable)
		Errors  []ddperror.Error // the errors that occured while parsing the argument
		exprEnd int              // where the expression was over (p.cur for the token after)
	}

	// a key for a cached argument
	type cachedArgKey struct {
		cur         int  // the start pos of that argument
		isReference bool // wether the argument was parsed with p.assignable() or p.expression()
	}

	// used for the algorithm below to parse each argument only once
	cached_args := map[cachedArgKey]*cachedArg{}
	// attempts to evaluate the arguments for the passed alias and checks if types match
	// returns nil if argument and parameter types don't match
	// similar to the alogrithm above
	// it also returns all errors that might have occured while doing so
	checkAlias := func(mAlias ast.Alias, typeSensitive bool) (map[string]ast.Expression, []ddperror.Error) {
		p.cur = start
		args := map[string]ast.Expression{}
		reported_errors := make([]ddperror.Error, 0)
		mAliasTokens := mAlias.GetTokens()
		mAliasArgs := mAlias.GetArgs()

		for i, l := 0, len(mAliasTokens); i < l && mAliasTokens[i].Type != token.EOF; i++ {
			tok := &mAliasTokens[i]

			if tok.Type == token.ALIAS_PARAMETER {
				argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
				paramType := mAliasArgs[argName]           // type of the current parameter

				pType := p.peek().Type
				// early return if a non-identifier expression is passed as reference
				if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
					return nil, reported_errors
				}

				// create the key for the argument
				cached_arg_key := cachedArgKey{cur: p.cur, isReference: paramType.IsReference}
				cached_arg, ok := cached_args[cached_arg_key]

				if !ok { // if the argument was not already parsed
					cached_arg = &cachedArg{}
					exprStart := p.cur
					isGrouping := false
					switch pType {
					case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER:
						p.advance() // single-token argument
					case token.NEGATE:
						p.advance()
						p.match(token.INT, token.FLOAT, token.IDENTIFIER)
					case token.LPAREN: // multiple-token arguments must be wrapped in parentheses
						isGrouping = true
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
					cached_arg.exprEnd = p.cur

					tokens := make([]token.Token, p.cur-exprStart, p.cur-exprStart+1)
					copy(tokens, p.tokens[exprStart:p.cur]) // copy all the tokens of the expression to be able to append the EOF
					// append the EOF needed for the parser
					eof := token.Token{Type: token.EOF, Literal: "", Indent: 0, Range: tok.Range, AliasInfo: nil}
					tokens = append(tokens, eof)
					argParser := &parser{
						tokens: tokens,
						errorHandler: func(err ddperror.Error) {
							reported_errors = append(reported_errors, err)
							cached_arg.Errors = append(cached_arg.Errors, err)
						},
						module: &ast.Module{
							FileName: p.module.FileName,
						},
						aliases:     p.aliases,
						typeNames:   p.typeNames,
						resolver:    p.resolver,
						typechecker: p.typechecker,
					}

					if paramType.IsReference {
						argParser.advance() // consume the identifier or LPAREN for assigneable() to work
						cached_arg.Arg = argParser.assigneable()
					} else if isGrouping {
						argParser.advance() // consume the LPAREN for grouping() to work
						cached_arg.Arg = argParser.grouping()
					} else {
						cached_arg.Arg = argParser.expression() // parse the argument
					}
					cached_args[cached_arg_key] = cached_arg
				} else {
					p.cur = cached_arg.exprEnd // skip the already parsed argument
					reported_errors = append(reported_errors, cached_arg.Errors...)
				}

				// check if the argument type matches the prameter type

				// we are in the for loop below, so the types must match
				// otherwise it doesn't matter
				if typeSensitive {
					typ := p.typechecker.EvaluateSilent(cached_arg.Arg) // evaluate the argument

					didMatch := true
					if typ != paramType.Type {
						didMatch = false
					} else if ass, ok := cached_arg.Arg.(*ast.Indexing);             // string-indexings may not be passed as char-reference
					paramType.IsReference && paramType.Type == ddptypes.BUCHSTABE && // if the parameter is a char-reference
						ok { // and the argument is a indexing
						lhs := p.typechecker.EvaluateSilent(ass.Lhs)
						if lhs == ddptypes.TEXT { // check if the lhs is a string
							didMatch = false
						}
					}

					if !didMatch {
						return nil, reported_errors
					}
				}

				args[argName] = cached_arg.Arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}
		return args, reported_errors
	}

	callOrLiteralFromAlias := func(alias ast.Alias, args map[string]ast.Expression) ast.Expression {
		if fnalias, isFuncAlias := alias.(*ast.FuncAlias); isFuncAlias {
			return &ast.FuncCall{
				Range: token.NewRange(&p.tokens[start], p.previous()),
				Tok:   p.tokens[start],
				Name:  fnalias.Func.Name(),
				Func:  fnalias.Func,
				Args:  args,
			}
		}

		stralias := alias.(*ast.StructAlias)
		return &ast.StructLiteral{
			Range:  token.NewRange(&p.tokens[start], p.previous()),
			Tok:    p.tokens[start],
			Struct: stralias.Struct,
			Args:   args,
		}
	}

	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		if args, errs := checkAlias(matchedAliases[i], true); args != nil {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
			return callOrLiteralFromAlias(matchedAliases[i], args)
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments
	mostFitting := matchedAliases[0]
	args, errs := checkAlias(mostFitting, false)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return callOrLiteralFromAlias(mostFitting, args)
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

// wether the next token indicates a typename
func (p *parser) isTypeName(t *token.Token) bool {
	switch t.Type {
	case token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE, token.TEXT,
		token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN:
		return true
	case token.IDENTIFIER:
		_, exists := p.typeNames[t.Literal]
		return exists
	}
	return false
}

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
	}
	p.panic("invalid TokenType (%d)", t)
	return ddptypes.VoidType{} // unreachable
}

// parses tokens into a DDPType
// expects the next token to be the start of the type
// returns nil and errors if no typename was found
func (p *parser) parseType() ddptypes.Type {
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return p.tokenTypeToType(p.previous().Type)
	case token.WAHRHEITSWERT, token.TEXT:
		if !p.match(token.LISTE) {
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
	case token.IDENTIFIER:
		if Type, exists := p.typeNames[p.previous().Literal]; exists {
			if p.match(token.LISTE) {
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
	if !p.match(token.WAHRHEITSWERT, token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER) {
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
	case token.IDENTIFIER:
		if Type, exists := p.typeNames[p.previous().Literal]; exists {
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
	if !p.match(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil, false // void indicates error
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE:
		return p.tokenTypeToType(p.previous().Type), false
	case token.WAHRHEITSWERT, token.TEXT:
		if p.match(token.LISTE) {
			return ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-2).Type)}, false
		} else if p.match(token.LISTEN) {
			if !p.consume(token.REFERENZ) {
				// report the error on the REFERENZ token, but still advance
				// because there is a valid token afterwards
				p.advance()
			}
			return ddptypes.ListType{Underlying: p.tokenTypeToType(p.peekN(-3).Type)}, true
		} else if p.match(token.REFERENZ) {
			return p.tokenTypeToType(p.peekN(-2).Type), true
		}
		return p.tokenTypeToType(p.previous().Type), false
	case token.ZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.ZAHL}, false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.ZAHL}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.ZAHL, true
	case token.KOMMAZAHLEN:
		if p.match(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.KOMMAZAHL}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.KOMMAZAHL, true
	case token.BUCHSTABEN:
		if p.match(token.LISTE) {
			return ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, false
		} else if p.match(token.LISTEN) {
			p.consume(token.REFERENZ)
			return ddptypes.ListType{Underlying: ddptypes.BUCHSTABE}, true
		}
		p.consume(token.REFERENZ)
		return ddptypes.BUCHSTABE, true
	case token.IDENTIFIER:
		if Type, exists := p.typeNames[p.previous().Literal]; exists {
			if p.match(token.LISTE) {
				return ddptypes.ListType{Underlying: Type}, false
			} else if p.match(token.LISTEN) {
				if !p.consume(token.REFERENZ) {
					// report the error on the REFERENZ token, but still advance
					// because there is a valid token afterwards
					p.advance()
				}
				return ddptypes.ListType{Underlying: Type}, true
			} else if p.match(token.REFERENZ) {
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

	if p.match(token.NICHTS) {
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

// returns the current scope of the parser, resolver and typechecker
func (p *parser) scope() *ast.SymbolTable {
	// same pointer as the one of the typechecker
	return p.resolver.CurrentTable
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
func (p *parser) consume1(t token.TokenType) bool {
	if p.check(t) {
		p.advance()
		return true
	}

	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, t))
	return false
}

// consume a series of tokens
func (p *parser) consume(t ...token.TokenType) bool {
	for _, v := range t {
		if !p.consume1(v) {
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

	p.err(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, toInterfaceSlice[token.TokenType, any](tokenTypes)...))
	return false
}

func (p *parser) errVal(err ddperror.Error) {
	if !p.panicMode {
		p.panicMode = true
		p.lastError = err
		p.errorHandler(p.lastError)
	}
}

// helper to report errors and enter panic mode
func (p *parser) err(code ddperror.Code, Range token.Range, msg string) {
	p.errVal(ddperror.New(code, Range, msg, p.module.FileName))
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

// check if a slice of tokens contains a literal
func containsLiteral(tokens []token.Token, literal string) bool {
	for _, v := range tokens {
		if v.Literal == literal {
			return true
		}
	}
	return false
}

// check if two tokens are equal for alias matching
func tokenEqual(t1, t2 *token.Token) bool {
	if t1 == t2 {
		return true
	}

	if t1.Type != t2.Type {
		return false
	}

	switch t1.Type {
	case token.ALIAS_PARAMETER:
		return *t1.AliasInfo == *t2.AliasInfo
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal == t2.Literal
	}

	return true
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// reports wether t1 < t2
// tokens are first sorted by their type
// if the types are equal they are sorted by their AliasInfo or Literal
// for AliasInfo: name < list < reference
// for Identifiers: t1.Literal < t2.Literal
func tokenLess(t1, t2 *token.Token) bool {
	if t1.Type != t2.Type {
		return t1.Type < t2.Type
	}

	switch t1.Type {
	case token.ALIAS_PARAMETER:
		if t1.AliasInfo.IsReference != t2.AliasInfo.IsReference {
			return boolToInt(t1.AliasInfo.IsReference) < boolToInt(t2.AliasInfo.IsReference)
		}

		isList1, isList2 := ddptypes.IsList(t1.AliasInfo.Type), ddptypes.IsList(t2.AliasInfo.Type)
		if isList1 != isList2 {
			return boolToInt(isList1) < boolToInt(isList2)
		}

		return t1.AliasInfo.Type.String() < t2.AliasInfo.Type.String()
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal < t2.Literal
	}

	return false
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

// applies fun to every element in slice
func apply[T any](fun func(T), slice []T) {
	for i := range slice {
		fun(slice[i])
	}
}

// converts a slice of a subtype to it's basetype
// T must be convertible to U
func toInterfaceSlice[T any, U any](slice []T) []U {
	result := make([]U, len(slice))
	for i := range slice {
		result[i] = any(slice[i]).(U)
	}
	return result
}

// keeps only those elements for which the filter function returns true
func filterSlice[T any](slice []T, filter func(t T) bool) []T {
	result := make([]T, 0, len(slice))
	for i := range slice {
		if filter(slice[i]) {
			result = append(result, slice[i])
		}
	}
	return result
}

func toPointerSlice[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

// returns (aliasExists, isFuncAlias, alias, pTokens)
func (p *parser) aliasExists(alias []token.Token) (bool, bool, ast.Alias, []*token.Token) {
	pTokens := toPointerSlice(alias[:len(alias)-1])
	if ok, alias := p.aliases.Contains(pTokens); ok {
		_, isFun := alias.(*ast.FuncAlias)
		return alias != nil, isFun, alias, pTokens
	}
	return false, false, nil, pTokens
}

// helper to add an alias slice to the parser
// only adds aliases that are not already defined
// and errors otherwise
func (p *parser) addAliases(aliases []ast.Alias, errRange token.Range) {
	// add all the aliases
	for _, alias := range aliases {
		// I fucking hate this thing
		// thank god they finally fixed it in 1.22
		// still leave it here just to be sure
		alias := alias

		if ok, isFun, existingAlias, pTokens := p.aliasExists(alias.GetTokens()); ok {
			p.err(ddperror.SEM_ALIAS_ALREADY_DEFINED, errRange, ddperror.MsgAliasAlreadyExists(existingAlias.GetOriginal().Literal, existingAlias.Decl().Name(), isFun))
		} else {
			p.aliases.Insert(pTokens, alias)
		}
	}
}
