/*
This File defines the entry point and some general functions of the parser
*/
package parser

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/parser/resolver"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state when parsing a .ddp file into an AST
type parser struct {
	// the tokens to parse (without comments)
	tokens []token.Token
	// all the comments from the original tokens slice
	comments []token.Token
	// index of the current token
	cur int
	// a function to which errors are passed
	errorHandler ddperror.Handler
	// latest reported error
	lastError ddperror.Error

	module *ast.Module
	// modules that were passed as environment, might not all be used
	predefinedModules map[string]*ast.Module
	// all found aliases (+ inbuild aliases)
	aliases *at.Trie[*token.Token, ast.Alias]
	// function which is currently being parsed
	currentFunction *ast.FuncDecl
	// wether the current function returns a boolean
	isCurrentFunctionBool bool
	// flag to not report following errors
	panicMode bool
	// wether the parser found an error
	errored bool
	// used to resolve every node directly after it has been parsed
	resolver *resolver.Resolver
	// used to typecheck every node directly after it has been parsed
	typechecker *typechecker.Typechecker
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

	is_first_tok_comment := tokens[0].Type == token.COMMENT

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

	var module_comment *token.Token
	if is_first_tok_comment {
		module_comment = &comments[0]
	}

	parser := &parser{
		tokens:       tokens,
		comments:     comments,
		cur:          0,
		errorHandler: nil,
		module: &ast.Module{
			FileName:             name,
			Imports:              make([]*ast.ImportStmt, 0),
			Comment:              module_comment,
			ExternalDependencies: make(map[string]struct{}, 5),
			Ast: &ast.Ast{
				Statements: make([]ast.Statement, 0),
				Comments:   comments,
				Symbols:    ast.NewSymbolTable(nil),
				Faulty:     false,
			},
			PublicDecls: make(map[string]ast.Declaration, 8),
			Operators:   make(map[ast.Operator][]*ast.FuncDecl, 8),
		},
		predefinedModules:     modules,
		aliases:               at.New[*token.Token, ast.Alias](tokenEqual, tokenLess),
		currentFunction:       nil,
		isCurrentFunctionBool: false,
		panicMode:             false,
		errored:               false,
		resolver:              &resolver.Resolver{},
		typechecker:           &typechecker.Typechecker{},
	}

	// wrap the errorHandler to set the parsers Errored variable
	// if it is called
	parser.errorHandler = func(err ddperror.Error) {
		if err.Level == ddperror.LEVEL_ERROR {
			parser.errored = true
		}
		errorHandler(err)
	}

	// prepare the resolver and typechecker with the inbuild symbols and types
	parser.resolver = resolver.New(parser.module, parser.errorHandler, name, &parser.panicMode)
	parser.typechecker = typechecker.New(parser.module, parser.errorHandler, name, &parser.panicMode)

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

	p.validateForwardDecls()

	p.module.Ast.Faulty = p.errored
	return p.module
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
	if p.matchAny(token.DER, token.DIE, token.DAS, token.WIR) { // might indicate a function, variable or struct
		if p.previous().Type == token.WIR {
			if p.matchSeq(token.NENNEN, token.DIE) {
				return &ast.DeclStmt{Decl: p.structDeclaration()}
			} else if p.matchAny(token.DEFINIEREN) {
				return &ast.DeclStmt{Decl: p.typeDefDecl()}
			}
			return &ast.DeclStmt{Decl: p.typeAliasDecl()}
		}

		n := -1
		if p.matchAny(token.OEFFENTLICHE) {
			n = -2
		}

		switch t := p.peek().Type; t {
		case token.ALIAS:
			p.advance()
			return p.aliasDecl()
		case token.FUNKTION:
			p.advance()
			return p.funcDeclaration(n - 1)
		default:
			return &ast.DeclStmt{Decl: p.varDeclaration(n, false)}
		}
	}

	return p.statement() // no declaration, so it must be a statement
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
			if ast.IsOperatorOverload(decl) {
				p.insertOperatorOverload(decl)
			}
		case *ast.StructDecl:
			aliases = append(aliases, toInterfaceSlice[*ast.StructAlias, ast.Alias](decl.Aliases)...)
		case *ast.TypeAliasDecl:
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

func (p *parser) validateForwardDecls() {
	ast.VisitModule(p.module, ast.FuncDeclVisitorFunc(func(decl *ast.FuncDecl) ast.VisitResult {
		if decl.Body == nil && decl.ExternFile.Type == token.ILLEGAL && decl.Def == nil {
			p.err(ddperror.SEM_FORWARD_DECL_WITHOUT_DEF,
				decl.NameTok.Range,
				fmt.Sprintf("Die Funktion '%s' wurde nur deklariert aber nie definiert", decl.Name()))
			p.panicMode = false
		}
		return ast.VisitSkipChildren
	}))
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

func (p *parser) errVal(err ddperror.Error) {
	if !p.panicMode {
		p.panicMode = true
		p.lastError = err
		p.errorHandler(p.lastError)
	}
}

// helper to report errors and enter panic mode
func (p *parser) err(code ddperror.Code, Range token.Range, msg string) {
	p.errVal(ddperror.New(code, ddperror.LEVEL_ERROR, Range, msg, p.module.FileName))
}

// helper to report errors and enter panic mode
func (p *parser) warn(code ddperror.Code, Range token.Range, msg string) {
	p.errorHandler(ddperror.New(code, ddperror.LEVEL_WARN, Range, msg, p.module.FileName))
}

// checks wether the alias already exists AND has a value attached to it
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

func (p *parser) insertOperatorOverload(decl *ast.FuncDecl) {
	overloads := p.module.Operators[decl.Operator]

	valid := true
	for _, overload := range overloads {
		if operatorParameterTypesEqual(overload.Parameters, decl.Parameters) {
			p.err(ddperror.SEM_OVERLOAD_ALREADY_DEFINED, decl.NameTok.Range, fmt.Sprintf("Der Operator '%s' ist für diese Parametertypen bereits überladen", decl.Operator))
			valid = false
		}
	}

	if valid {
		// keep the slice sorted in descending order, so that references are prioritized
		i, _ := slices.BinarySearchFunc(overloads, decl, func(a, t *ast.FuncDecl) int {
			countRefArgs := func(params []ast.ParameterInfo) int {
				result := 0
				for i := range params {
					if params[i].Type.IsReference {
						result++
					}
				}
				return result
			}

			return countRefArgs(t.Parameters) - countRefArgs(a.Parameters)
		})

		overloads = slices.Insert(overloads, i, decl)
		p.module.Operators[decl.Operator] = overloads
	}
}
