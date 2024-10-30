package parser

import (
	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// an unknownType that was found during preprocessing
type unknownType struct {
	// name of the type
	name     string
	isPublic bool
	// ddptypes.INVALID_GENDER if not known
	gender ddptypes.GrammaticalGender
	// nil if this type is not yet known to be a struct
	// filled with the field if it stems from a structDecl
	structFields []ddptypes.StructField
}

var _ = ddptypes.Type(unknownType{})

func (t unknownType) String() string {
	return t.name
}

func (t unknownType) Gender() ddptypes.GrammaticalGender {
	return t.gender
}

// part of the result after preprocessing
// fields already present in the parser (like aliases) are omitted
type preprocessResult struct {
	// maps starting positions (p.cur) to preprocessed declarations
	declarations map[int]ast.Declaration
	// maps type names to their not-yet resolved type
	types map[string]unknownType
	// list of all aliases that were found
	aliases []ast.Alias
}

func (p *parser) preprocess() {
	errHndl := p.errorHandler
	defer func() {
		p.errorHandler = errHndl
	}()
	p.errorHandler = ddperror.EmptyHandler

	for !p.atEnd() {
		p.preprocessDeclaration()
	}
}

func (p *parser) preprocessDeclaration() ast.Declaration {
	// might indicate a function, variable or struct
	if p.matchAny(token.DER, token.DIE, token.DAS, token.WIR) {
		if p.previous().Type == token.WIR {
			if p.matchSeq(token.NENNEN, token.DIE) {
				return p.preprocessStructDecl()
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
		// TODO
		// case token.ALIAS:
		// 	p.advance()
		// 	return p.aliasDecl()
		case token.FUNKTION:
			p.advance()
			return &ast.DeclStmt{Decl: p.funcDeclaration(n - 1)}
		default:
			return p.preprocessVarDeclaration(n, false)
		}
	}

	return p.statement() // no declaration, so it must be a statement
}

func (p *parser) preprocessStructDecl() ast.Declaration {
	begin := p.peekN(-3) // token.WIR
	comment := p.parseDeclComment(begin.Range)

	isPublic := p.matchAny(token.OEFFENTLICHE)
	if !p.matchSeq(token.KOMBINATION, token.AUS) {
		return nil
	}

	var fields []ddptypes.StructField
	indent := begin.Indent + 1
	for p.peek().Indent >= indent && !p.atEnd() {
		if !p.matchSeq(token.DER, token.DEM) {
			return nil
		}
		n := -1
		if p.matchAny(token.OEFFENTLICHEN) {
			n = -2
		}
		if field := p.preprocessFieldDecl(n); field != nil {
			fields = append(fields, *field)
		}
		if !p.matchAny(token.COMMA) {
			p.advance()
		}
	}

	gender := p.parseGender()
	if !p.matchAny(token.IDENTIFIER) {
		return nil
	}
	name := p.previous()

	if !p.matchSeq(token.COMMA, token.UND, token.ERSTELLEN, token.SIE, token.SO, token.COLON, token.STRING) {
		return nil
	}

	var rawAliases []*token.Token
	if p.previous().Type == token.STRING {
		rawAliases = append(rawAliases, p.previous())
	}
	for p.matchAny(token.COMMA) || p.matchAny(token.ODER) && p.peek().Indent > 0 && !p.atEnd() {
		if p.matchAny(token.STRING) {
			rawAliases = append(rawAliases, p.previous())
		}
	}

	var structAliases []*ast.StructAlias
	var structAliasTokens [][]*token.Token
	fieldsForValidation := mapSlice(fields, func(field ddptypes.StructField) *ast.VarDecl {
		return &ast.VarDecl{
			NameTok: token.Token{Literal: field.Name},
			Type:    field.Type,
		}
	})
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

	if _, ok := p.ppr.types[name.Literal]; ok {
		p.err(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, ddperror.MsgNameAlreadyExists(name.Literal))
	}

	p.ppr.types[name.Literal] = unknownType{
		name:         name.Literal,
		isPublic:     isPublic,
		gender:       gender,
		structFields: fields,
	}

	decl := &ast.StructDecl{
		Range:      token.NewRange(begin, p.previous()),
		CommentTok: comment,
		Tok:        *begin,
		NameTok:    *name,
		IsPublic:   isPublic,
		Mod:        p.module,
		Fields:     nil,
		Type:       nil,
		Aliases:    structAliases,
	}

	for i := range structAliases {
		structAliases[i].Struct = decl
		p.ppr.aliases = append(p.ppr.aliases, structAliases[i])
	}

	return decl
}

func (p *parser) preprocessFieldDecl(startDepth int) *ddptypes.StructField {
	begin := p.peekN(startDepth) // Der/Die/Das
	p.parseDeclComment(begin.Range)

	typ := p.preprocessType()

	if !p.matchAny(token.IDENTIFIER) {
		return nil
	}

	name := p.previous()
	for !p.atEnd() && p.peek().Line() == begin.Line() {
		p.advance()
	}
	if p.previous().Type == token.COMMA {
		p.decrease()
	}
	return &ddptypes.StructField{
		Name: name.Literal,
		Type: typ,
	}
}

func (p *parser) preprocessVarDeclaration(startDepth int) ast.Declaration {
	begin := p.peekN(startDepth) // Der/Die/Das
	comment := p.parseDeclComment(begin.Range)

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE || p.peekN(startDepth+1).Type == token.OEFFENTLICHEN

	isExternVisible := false
	if isPublic && p.matchAny(token.COMMA) || p.matchAny(token.EXTERN) {
		if p.previous().Type == token.COMMA {
			p.consume(token.EXTERN)
		}
		p.consume(token.SICHTBARE)
		isExternVisible = true
	}

	type_start := p.peek()
	typ := p.preprocessType()
	type_end := p.previous()

	// we need a name, so bailout if none is provided
	if !p.matchAny(token.IDENTIFIER) {
		return nil
	}

	name := p.previous()
	if !p.matchAny(token.IST) {
		return nil
	}

	for !p.atEnd() && p.peek().Line() == begin.Line() {
		p.advance()
	}

	return &ast.VarDecl{
		Range:           token.NewRange(begin, p.previous()),
		CommentTok:      comment,
		Type:            typ,
		NameTok:         *name,
		TypeRange:       token.NewRange(type_start, type_end),
		IsPublic:        isPublic,
		IsExternVisible: isExternVisible,
		Mod:             p.module,
		InitVal:         nil,
	}
}

func (p *parser) preprocessType() ddptypes.Type {
	if !p.matchAny(token.ZAHL, token.KOMMAZAHL, token.WAHRHEITSWERT, token.BUCHSTABE,
		token.TEXT, token.ZAHLEN, token.KOMMAZAHLEN, token.BUCHSTABEN, token.IDENTIFIER, token.VARIABLE, token.VARIABLEN) {
		p.err(ddperror.SYN_EXPECTED_TYPENAME, p.peek().Range, ddperror.MsgGotExpected(p.peek().Literal, "ein Typname"))
		return nil
	}

	switch p.previous().Type {
	case token.ZAHL, token.KOMMAZAHL, token.BUCHSTABE, token.VARIABLE:
		return p.tokenTypeToType(p.previous().Type)
	case token.WAHRHEITSWERT, token.TEXT:
		if !p.matchAny(token.LISTE) {
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
	case token.VARIABLEN:
		p.consume(token.LISTE)
		return ddptypes.ListType{Underlying: ddptypes.VARIABLE}
	case token.IDENTIFIER:
		Type := unknownType{
			name:         p.previous().Literal,
			isPublic:     false,
			gender:       ddptypes.INVALID_GENDER,
			structFields: nil,
		}

		if p.matchAny(token.LISTE) {
			return ddptypes.ListType{Underlying: Type}
		}
		return Type
	}

	return nil // unreachable
}

func mapSlice[T, U any](s []T, mapper func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}
