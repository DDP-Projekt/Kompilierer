/*
This file contains functions to parse function/struct/... declarations
*/
package parser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
	"golang.org/x/exp/maps"
)

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
				if ok, existingAlias, pTokens := p.aliasExists(alias); ok {
					p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, v.Range, ast.MsgAliasAlreadyExists(existingAlias))
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
				if ok, existingAlias, pTokens := p.aliasExists(aliasTokens); ok {
					p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, rawAlias.Range, ast.MsgAliasAlreadyExists(existingAlias))
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

func (p *parser) validateExpressionAlias(aliasTokens []token.Token) ([]string, *ddperror.Error) {
	if countElements(aliasTokens, isIllegalToken) > 0 { // validate that the alias does not contain illegal tokens
		err := ddperror.New(
			ddperror.SEM_MALFORMED_ALIAS,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			"Der Alias enthält ungültige Symbole",
			p.module.FileName,
		)
		return nil, &err
	}

	parameters := map[string]struct{}{} // set that holds the parameter names contained in the alias

	// validate that each parameter is contained in the alias exactly once
	// and fill in the AliasInfo
	for i, v := range aliasTokens {
		if isAliasExpr(v) {
			k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
			if _, exists := parameters[k]; exists {
				err := ddperror.New(ddperror.SEM_ALIAS_BAD_NUM_ARGS,
					token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
					fmt.Sprintf("Der Alias enthält den Parameter %s mehrmals", k),
					p.module.FileName,
				)
				return nil, &err
			} else {
				parameters[k] = struct{}{}
				aliasTokens[i].AliasInfo = &ddptypes.ParameterType{
					Type:        ddptypes.VoidType{},
					IsReference: false,
				}
			}
		}
	}
	return maps.Keys(parameters), nil
}

// used for generating the internal names
// NOTE: maybe make this atomic if concurrency is used in the future
var expressionDeclCount = 0

// parses and expression Declaration
// startDepth is the int passed to p.peekN(n) to get to the DIE token of the declaration
// TODO: scoped aliases
func (p *parser) expressionDecl(startDepth int) ast.Declaration {
	begin := p.peekN(startDepth)
	comment := p.commentBeforePos(begin.Range.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < begin.Range.Start.Line-1 {
		comment = nil
	}

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE

	p.consume(token.STRING)
	aliasTok := p.previous()

	var alias *ast.ExpressionAlias
	didError := false
	errHandleWrapper := func(err ddperror.Error) { didError = true; p.errorHandler(err) }
	if alias, err := scanner.ScanAlias(*aliasTok, errHandleWrapper); err == nil && !didError {
		if len(alias) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
			p.err(ddperror.SEM_MALFORMED_ALIAS, aliasTok.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
		} else if parameters, err := p.validateExpressionAlias(alias); err == nil { // check that the alias fits the function
			if ok, existingAlias, pTokens := p.aliasExists(alias); ok {
				p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, aliasTok.Range, ast.MsgAliasAlreadyExists(existingAlias))
			} else {
				alias := &ast.ExpressionAlias{Tokens: alias, Original: *aliasTok, ExprDecl: nil, Args: parameters}
				p.aliases.Insert(pTokens, alias)
			}
		} else {
			p.errVal(*err)
		}
	}

	p.consume(token.STEHT, token.FÜR, token.DEN, token.AUSDRUCK)

	symbols := p.newScope()
	for _, argName := range alias.Args {
		symbols.InsertDecl(argName, &ast.VarDecl{
			Range:      aliasTok.Range,
			CommentTok: nil,
			Type:       ddptypes.VoidType{},
			NameTok:    *aliasTok,
			IsPublic:   false,
			Mod:        p.module,
			InitVal:    nil,
		})
	}

	p.setScope(symbols)
	expr := p.expression()
	p.exitScope()

	name := fmt.Sprintf("$expr_decl_%d", expressionDeclCount)
	var NameTok *token.Token
	if p.match(token.MIT) {
		p.consume(token.NAMEN, token.IDENTIFIER)
		NameTok = p.previous()
		name = NameTok.Literal
	}
	p.consume(token.DOT)

	alias.ExprDecl = &ast.ExpressionDecl{
		Range:        token.NewRange(begin, p.previous()),
		CommentTok:   comment,
		Tok:          *begin,
		Alias:        alias,
		Expr:         expr,
		NameTok:      NameTok,
		AssignedName: name,
		IsPublic:     isPublic,
		Mod:          p.module,
		Symbols:      symbols,
	}
	expressionDeclCount++

	return alias.ExprDecl
}
