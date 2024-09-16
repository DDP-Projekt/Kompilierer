package parser

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// holds state used during pre-processing
//
// pre-processing scans the tokens before parsing them
// and extracts all function/struct aliases and type declarations
type preprocessor struct {
	tokenWalker
	errorReporter
	// modules that were passed as environment, might not all be used
	predefinedModules map[string]*ast.Module
	// all found aliases
	aliases *at.Trie[*token.Token, ast.Alias]
	// global symbol table
	symbols *ast.SymbolTable
	// map of token index to pre-processed statement/declarations
	parsedLocations map[int]ast.Node
}

type preprocessingResult struct {
	// pre-parsed alias trie
	aliases *at.Trie[*token.Token, ast.Alias]
	// global symbol table with pre-processed type and function declarations
	// (no variable declarations)
	symbols *ast.SymbolTable
	// parsed import statements
	imports []*ast.ImportStmt
	// all pre-processed declarations
	declarations []ast.Declaration
	// map of token index to pre-processed statement/declarations
	parsedLocations map[int]ast.Node
}

func newPreprocessor(name string, tokens, comments []token.Token,
	errorHandler ddperror.Handler, predefinedModules map[string]*ast.Module,
) *preprocessor {
	pp := &preprocessor{
		tokenWalker: tokenWalker{
			tokens:   tokens,
			comments: comments,
			cur:      0,
		},
		errorReporter: errorReporter{
			errorHandler: errorHandler,
			lastError:    ddperror.Error{},
			panicMode:    false,
			errored:      false,
			fileName:     name,
		},
		predefinedModules: predefinedModules,
		aliases:           &at.Trie[*token.Token, ast.Alias]{},
		symbols:           ast.NewSymbolTable(nil),
		parsedLocations:   make(map[int]ast.Node, 8),
	}
	pp.errFunc = func(c ddperror.Code, r token.Range, s string) {
		pp.err(c, r, s)
	}
	pp.initializeReporter()

	return pp
}

func (pp *preprocessor) preprocess() preprocessingResult {
	// imports may only occur at the top and are all pre-processed first
	imports := pp.processImports()

	// pre-process the rest
	declarations := make([]ast.Declaration, 0, 8)
	for !pp.atEnd() {
		decl, location := pp.parseDeclaration()
		if decl != nil {
			declarations = append(declarations, decl)
			pp.parsedLocations[location] = decl
		}

		if pp.panicMode {
			pp.panicMode = false
			pp.synchronize()
		}
	}

	return preprocessingResult{
		aliases:         pp.aliases,
		symbols:         pp.symbols,
		imports:         imports,
		declarations:    declarations,
		parsedLocations: pp.parsedLocations,
	}
}

// checks wether the alias already exists AND has a value attached to it
// returns (aliasExists, isFuncAlias, alias, pTokens)
func (pp *preprocessor) aliasExists(alias []token.Token) (bool, bool, ast.Alias, []*token.Token) {
	pTokens := toPointerSlice(alias[:len(alias)-1])
	if ok, alias := pp.aliases.Contains(pTokens); ok {
		_, isFun := alias.(*ast.FuncAlias)
		return alias != nil, isFun, alias, pTokens
	}
	return false, false, nil, pTokens
}

// helper to add an alias slice to the parser
// only adds aliases that are not already defined
// and errors otherwise
func (pp *preprocessor) addAliases(aliases []ast.Alias, errRange token.Range) {
	// add all the aliases
	for _, alias := range aliases {
		// I fucking hate this thing
		// thank god they finally fixed it in 1.22
		// still leave it here just to be sure
		alias := alias

		if ok, isFun, existingAlias, pTokens := pp.aliasExists(alias.GetTokens()); ok {
			pp.err(ddperror.SEM_ALIAS_ALREADY_DEFINED, errRange, ddperror.MsgAliasAlreadyExists(existingAlias.GetOriginal().Literal, existingAlias.Decl().Name(), isFun))
		} else {
			pp.aliases.Insert(pTokens, alias)
		}
	}
}

func (pp *preprocessor) insertOperatorOverload(decl *ast.FuncDecl) {
	overloads := pp.symbols.LookupOperator(decl.Operator)

	// check that the overload does not exist already
	for _, overload := range overloads {
		if operatorParameterTypesEqual(overload.Parameters, decl.Parameters) {
			pp.err(ddperror.SEM_OVERLOAD_ALREADY_DEFINED, decl.NameTok.Range, fmt.Sprintf("Der Operator '%s' ist für diese Parametertypen bereits überladen", decl.Operator))
			return
		}
	}

	pp.symbols.AddOverload(decl.Operator, decl)
}
