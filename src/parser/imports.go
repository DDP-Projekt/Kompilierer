package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddppath"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

func (pp *preprocessor) processImports() (imports []*ast.ImportStmt) {
	for pp.matchAny(token.BINDE) {
		import_location := pp.cur - 1
		stmt := pp.parseImport()
		if stmt != nil {
			pp.resolveImport(stmt)
			imports = append(imports, stmt)
			pp.parsedLocations[import_location] = stmt
		}

		if pp.panicMode {
			pp.panicMode = false
			pp.synchronize()
		}
	}
	return imports
}

func (pp *preprocessor) parseImport() *ast.ImportStmt {
	binde := pp.previous()

	var stmt *ast.ImportStmt
	if pp.matchAny(token.STRING) {
		stmt = &ast.ImportStmt{
			FileName:        *pp.previous(),
			ImportedSymbols: nil,
		}
	} else if pp.matchAny(token.IDENTIFIER) {
		stmt = pp.parseSymbolImport()
	} else {
		pp.err(
			ddperror.SYN_UNEXPECTED_TOKEN,
			pp.peek().Range,
			ddperror.MsgGotExpected(pp.peek(), "ein Text Literal oder ein Name"),
		)
	}

	pp.consume(token.EIN, token.DOT)
	if stmt != nil {
		stmt.Range = token.NewRange(binde, pp.previous())
	}
	return stmt
}

func (pp *preprocessor) parseSymbolImport() *ast.ImportStmt {
	importedSymbols := []token.Token{*pp.previous()}
	if pp.peek().Type != token.AUS {
		if pp.matchAny(token.UND) {
			pp.consume(token.IDENTIFIER)
			importedSymbols = append(importedSymbols, *pp.previous())
		} else {
			for pp.matchAny(token.COMMA) {
				if pp.consume(token.IDENTIFIER) {
					importedSymbols = append(importedSymbols, *pp.previous())
				}
			}
			if pp.consume(token.UND) && pp.consume(token.IDENTIFIER) {
				importedSymbols = append(importedSymbols, *pp.previous())
			}
		}
	}
	pp.consume(token.AUS)
	if pp.consume(token.STRING) {
		return &ast.ImportStmt{
			FileName:        *pp.previous(),
			ImportedSymbols: importedSymbols,
		}
	}
	return nil
}

func (pp *preprocessor) resolveImport(importStmt *ast.ImportStmt) {
	var (
		rawPath  = ast.TrimStringLit(&importStmt.FileName)
		inclPath string
		err      error
	)

	if rawPath == "" {
		pp.err(ddperror.MISC_INCLUDE_ERROR, importStmt.FileName.Range, "Bei Einbindungen muss ein Dateipfad angegeben werden")
		return
	}

	// resolve the actual file path
	if strings.HasPrefix(rawPath, "Duden") {
		inclPath = filepath.Join(ddppath.InstallDir, rawPath) + ".ddp"
	} else if inclPath, err = filepath.Abs(filepath.Join(filepath.Dir(pp.fileName), rawPath+".ddp")); err != nil {
		pp.err(ddperror.SYN_MALFORMED_INCLUDE_PATH,
			importStmt.FileName.Range,
			fmt.Sprintf("Fehlerhafter Dateipfad '%s': \"%s\"", rawPath+".ddp", err.Error()),
		)
		return
	}

	// the module is new
	if module, ok := pp.predefinedModules[inclPath]; !ok {
		pp.predefinedModules[inclPath] = nil // already add the name to the map to break recursion
		// parse the new module
		importStmt.Module, err = Parse(Options{
			FileName:     inclPath,
			Source:       nil,
			Tokens:       nil,
			Modules:      pp.predefinedModules,
			ErrorHandler: pp.errorHandler,
		})
		if err != nil {
			pp.err(ddperror.MISC_INCLUDE_ERROR, importStmt.Range, fmt.Sprintf("Fehler beim einbinden von '%s': %s", rawPath+".ddp", err.Error()))
			return // return early on error
		}

		// add the module to the list and to the importStmt
		pp.predefinedModules[inclPath] = importStmt.Module
	} else { // we already included the module
		// circular import error
		if module == nil {
			pp.err(ddperror.MISC_INCLUDE_ERROR,
				importStmt.Range,
				fmt.Sprintf("Zwei Module dürfen sich nicht gegenseitig einbinden! Das Modul '%s' versuchte das Modul '%s' einzubinden, während es von diesem Module eingebunden wurde", pp.fileName, rawPath+".ddp"),
			)
			return // return early on error
		}

		importStmt.Module = module
	}

	// add aliases and operator overloads to the symbol table
	ast.IterateImportedDecls(importStmt, func(_ string, decl ast.Declaration, tok token.Token) bool {
		if decl == nil {
			return true // continue
		}

		// skip decls that are already defined
		// the resolver will error here
		if _, exists, _ := pp.symbols.LookupDecl(decl.Name()); exists {
			return true // continue
		}

		switch decl := decl.(type) {
		case *ast.FuncDecl:
			pp.addAliases(toInterfaceSlice[*ast.FuncAlias, ast.Alias](decl.Aliases), tok.Range)
			if ast.IsOperatorOverload(decl) {
				pp.insertOperatorOverload(decl)
			}
		case *ast.StructDecl:
			pp.addAliases(toInterfaceSlice[*ast.StructAlias, ast.Alias](decl.Aliases), tok.Range)
		}
		return true
	})
}
