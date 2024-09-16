package parser

import (
	"fmt"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

func (pp *preprocessor) parseDeclaration() (ast.Declaration, int) {
	// might indicate a function, variable or struct
	location := pp.cur
	if pp.matchAny(token.DER, token.DIE, token.DAS, token.WIR) {
		if pp.previous().Type == token.WIR {
			if pp.matchSeq(token.NENNEN, token.DIE) {
				return pp.structDeclaration(), location
			} else if pp.matchAny(token.DEFINIEREN) {
				return pp.typeDefDecl(), location
			}
			return pp.typeAliasDecl(), location
		}

		n := -1
		if pp.matchAny(token.OEFFENTLICHE) {
			n = -2
		}

		switch t := pp.peek().Type; t {
		case token.ALIAS:
			pp.advance()
			return pp.aliasDecl(), location
		case token.FUNKTION:
			pp.advance()
			return pp.funcDeclaration(n - 1), location
		default:
			return pp.varDeclaration(n, false), location
		}
	}

	return nil, -1
}

func (pp *preprocessor) parseDeclComment(beginRange token.Range) *token.Token {
	comment := pp.commentBeforePos(beginRange.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < beginRange.Start.Line-1 {
		comment = nil
	}
	return comment
}

func (pp *preprocessor) structDeclaration() ast.Declaration {
	begin := pp.peekN(-3) // token.WIR
	comment := pp.parseDeclComment(begin.Range)

	isPublic := pp.matchAny(token.OEFFENTLICHE)
	pp.consume(token.KOMBINATION, token.AUS)

	// parse the fields
	var fields []ast.Declaration
	indent := begin.Indent + 1
	for pp.peek().Indent >= indent && !pp.atEnd() {
		pp.consumeAny(token.DER, token.DEM)
		n := -1
		if pp.matchAny(token.OEFFENTLICHEN) {
			n = -2
		}
		fields = append(fields, pp.varDeclaration(n, true))
		if !pp.consume(token.COMMA) {
			pp.advance()
		}
	}

	// deterime the grammatical gender
	gender := pp.parseGender()

	if !pp.consume(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Code:  ddperror.SEM_NAME_UNDEFINED,
				Range: token.NewRange(pp.peekN(-2), pp.peek()),
				File:  pp.module.FileName,
				Msg:   "Es wurde ein Kombinations Name erwartet",
			},
			Tok: *pp.peek(),
			Mod: pp.module,
		}
	}
	name := pp.previous()

	if _, exists := pp.scope().LookupType(name.Literal); exists {
		pp.err(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, fmt.Sprintf("Ein Typ mit dem Namen '%s' existiert bereits", name.Literal))
	}

	pp.consume(token.COMMA, token.UND, token.ERSTELLEN, token.SIE, token.SO, token.COLON, token.STRING)
	var rawAliases []*token.Token
	if pp.previous().Type == token.STRING {
		rawAliases = append(rawAliases, pp.previous())
	}
	for pp.matchAny(token.COMMA) || pp.matchAny(token.ODER) && pp.peek().Indent > 0 && !pp.atEnd() {
		if pp.consume(token.STRING) {
			rawAliases = append(rawAliases, pp.previous())
		}
	}

	var structAliases []*ast.StructAlias
	var structAliasTokens [][]*token.Token
	fieldsForValidation := toInterfaceSlice[ast.Declaration, *ast.VarDecl](
		filterSlice(fields, func(decl ast.Declaration) bool { _, ok := decl.(*ast.VarDecl); return ok }),
	)
	for _, rawAlias := range rawAliases {
		didError := false
		errHandleWrapper := func(err ddperror.Error) { didError = true; pp.errorHandler(err) }
		if aliasTokens, err := scanner.ScanAlias(*rawAlias, errHandleWrapper); err == nil && !didError {
			if len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
				pp.err(ddperror.SEM_MALFORMED_ALIAS, rawAlias.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if err, args := pp.validateStructAlias(aliasTokens, fieldsForValidation); err == nil {
				if ok, isFunc, existingAlias, pTokens := pp.aliasExists(aliasTokens); ok {
					pp.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, rawAlias.Range, ddperror.MsgAliasAlreadyExists(rawAlias.Literal, existingAlias.Decl().Name(), isFunc))
				} else {
					structAliases = append(structAliases, &ast.StructAlias{Tokens: aliasTokens, Original: *rawAlias, Struct: nil, Args: args})
					structAliasTokens = append(structAliasTokens, pTokens)
				}
			} else {
				pp.errVal(*err)
			}
		}
	}

	structType := &ddptypes.StructType{
		Name:       name.Literal,
		GramGender: gender,
		Fields:     varDeclsToFields(fieldsForValidation),
	}

	decl := &ast.StructDecl{
		Range:      token.NewRange(begin, pp.previous()),
		CommentTok: comment,
		Tok:        *begin,
		NameTok:    *name,
		IsPublic:   isPublic,
		Mod:        pp.module,
		Fields:     fields,
		Type:       structType,
		Aliases:    structAliases,
	}

	for i := range structAliases {
		structAliases[i].Struct = decl
		pp.aliases.Insert(structAliasTokens[i], structAliases[i])
	}

	return decl
}
