/*
This File defines the entry point and some general functions of the parser
*/
package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/parser/resolver"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
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
			ExternalDependencies: make(map[string]struct{}, 5),
			Ast: &ast.Ast{
				Statements: make([]ast.Statement, 0),
				Comments:   comments,
				Symbols:    ast.NewSymbolTable(nil),
				Faulty:     false,
			},
			PublicDecls: make(map[string]ast.Declaration, 8),
		},
		predefinedModules:     modules,
		aliases:               at.New[*token.Token, ast.Alias](tokenEqual, tokenLess),
		typeNames:             make(map[string]ddptypes.Type, 8),
		currentFunction:       "",
		isCurrentFunctionBool: false,
		panicMode:             false,
		errored:               false,
		resolver:              &resolver.Resolver{},
		typechecker:           &typechecker.Typechecker{},
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

		p.advance()
		switch t := p.previous().Type; t {
		case token.ALIAS:
			return p.aliasDecl()
		case token.FUNKTION:
			return &ast.DeclStmt{Decl: p.funcDeclaration(n - 1)}
		default:
			return &ast.DeclStmt{Decl: p.varDeclaration(n-1, false)}
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
	p.errVal(ddperror.New(code, Range, msg, p.module.FileName))
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
