/*
This file contains functions to parse function/struct/... declarations
*/
package parser

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/scanner"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

func (p *parser) parseDeclComment(beginRange token.Range) *token.Token {
	comment := p.commentBeforePos(beginRange.Start)
	// ignore the comment if it is not next to or directly above the declaration
	if comment != nil && comment.Range.End.Line < beginRange.Start.Line-1 {
		comment = nil
	}
	// prefer to attach the comment to a declaration, rather than to the module
	if comment == p.module.Comment {
		p.module.Comment = nil
	}
	return comment
}

// helper for boolean assignments
func (p *parser) assignRhs(withComma bool) ast.Expression {
	var expr ast.Expression // the final expression

	if p.matchAny(token.TRUE, token.FALSE) {
		tok := p.previous() // wahr or falsch token
		// parse possible wahr/falsch wenn syntax
		if p.matchSeq(token.COMMA, token.WENN) {
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
			if withComma {
				p.consumeSeq(token.COMMA)
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

func (p *parser) matchExternSichtbar(isPublic bool) bool {
	if isPublic && p.matchAny(token.COMMA) || p.matchAny(token.EXTERN) {
		if p.previous().Type == token.COMMA {
			p.consumeSeq(token.EXTERN)
		}
		p.consumeSeq(token.SICHTBARE)
		return true
	}
	return false
}

func (p *parser) constDeclaration(startDepth int) ast.Declaration {
	begin := p.peekN(startDepth) // Die
	if begin.Type != token.DIE {
		p.err(ddperror.SYN_GENDER_MISMATCH, begin.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", "Die"))
	}

	comment := p.parseDeclComment(begin.Range)

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE

	p.consumeSeq(token.KONSTANTE)

	// we need a name, so bailout if none is provided
	if !p.consumeSeq(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.module.FileName,
				Msg:   "Es wurde ein Name für die Konstante erwartet",
			},
			Tok: *p.peek(),
			Mod: p.module,
		}
	}
	name := p.previous()

	p.consumeSeq(token.IST)

	expr := p.assignRhs(false) // TODO: add support for lists "N Mal x"
	p.consumeSeq(token.DOT)

	if _, isLiteral := expr.(ast.Literal); !isLiteral {
		p.err(ddperror.SYN_EXPECTED_LITERAL, expr.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
	}
	if expr, isList := expr.(*ast.ListLit); isList {
		for _, v := range expr.Values {
			if _, isLiteral := v.(ast.Literal); !isLiteral {
				p.err(ddperror.SYN_EXPECTED_LITERAL, v.GetRange(), "Es wurde ein Literal erwartet aber ein Ausdruck gefunden")
			}
		}
	}

	// prefer trailing comments as long as they are on the same line
	if trailingComment := p.commentAfterPos(p.previous().Range.End); trailingComment != nil && trailingComment.Range.Start.Line == p.previous().Range.End.Line {
		comment = trailingComment
	}

	return &ast.ConstDecl{
		Range:      token.NewRange(begin, p.previous()),
		Mod:        p.module,
		CommentTok: comment,
		IsPublic:   isPublic,
		NameTok:    *name,
		Val:        expr,
	}
}

// parses a variable declaration
// startDepth is the int passed to p.peekN(n) to get to the DER/DIE token of the declaration
// isField indicates that this declaration should be parsed as a struct field
func (p *parser) varDeclaration(startDepth int, isField bool) ast.Declaration {
	begin := p.peekN(startDepth) // Der/Die/Das
	comment := p.parseDeclComment(begin.Range)

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE || p.peekN(startDepth+1).Type == token.OEFFENTLICHEN
	isExternVisible := p.matchExternSichtbar(isPublic)

	type_start := p.peek()
	typ := p.parseType()
	type_end := p.previous()
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
	if !p.consumeSeq(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.module.FileName,
				Msg:   "Es wurde ein Name für die Variable erwartet",
			},
			Tok: *p.peek(),
			Mod: p.module,
		}
	}

	name := p.previous()
	if isField {
		p.consumeSeq(token.MIT, token.STANDARDWERT)
	} else {
		p.consumeSeq(token.IST)
	}
	var expr ast.Expression

	if !ddptypes.Equal(typ, ddptypes.WAHRHEITSWERT) && ddptypes.IsList(typ) { // TODO: fix this with function calls and groupings
		expr = p.expression()
		if p.matchAny(token.COUNT_MAL) {
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
		expr = p.assignRhs(false)
	}

	if !isField {
		p.consumeSeq(token.DOT)
	}

	// prefer trailing comments as long as they are on the same line
	if trailingComment := p.commentAfterPos(p.previous().Range.End); trailingComment != nil && trailingComment.Range.Start.Line == p.previous().Range.End.Line {
		comment = trailingComment
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
		InitVal:         expr,
	}
}

// helper for parsing function declarations
// a paramName is allowed if it does not override a struct or function declaration
// other variables are allowed to be overridden because of name-shadowing
func (p *parser) paramNameAllowed(name *token.Token) bool {
	_, exists, isVar := p.scope().LookupDecl(name.Literal)
	return !exists || (exists && isVar)
}

// helper for funcDeclaration
// parses the parameters of a function declaration
func (p *parser) parseFunctionParameters(perr func(ddperror.Code, token.Range, string), validate func(bool)) (params []ast.ParameterInfo) {
	if !p.matchAny(token.MIT) {
		return params
	}

	// parse if there will be one or multiple parameters
	singleParameter := true
	if p.matchSeq(token.DEN, token.PARAMETERN) {
		singleParameter = false
	} else if !p.matchSeq(token.DEM, token.PARAMETER) {
		perr(ddperror.SYN_UNEXPECTED_TOKEN, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "'de[n/m] Parameter[n]'"))
	}

	// parse the first param name
	validate(p.consumeSeq(token.IDENTIFIER))
	firstName := p.previous()
	if !p.paramNameAllowed(firstName) { // check that the parameter name is not already used
		perr(ddperror.SEM_NAME_ALREADY_DEFINED, firstName.Range, ddperror.MsgNameAlreadyExists(firstName.Literal))
	}

	params = append(params, ast.ParameterInfo{
		Name:    *firstName,
		Comment: p.getLeadingOrTrailingComment(),
	})

	if !singleParameter {
		// helper function to avoid too much repitition
		addParamName := func(name *token.Token) {
			if containsName(params, name.Literal) { // check that each parameter name is unique
				perr(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, fmt.Sprintf("Ein Parameter mit dem Namen '%s' ist bereits vorhanden", name.Literal))
				return
			}
			if !p.paramNameAllowed(name) { // check that the parameter name is not already used
				perr(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, ddperror.MsgNameAlreadyExists(name.Literal))
				return
			}
			params = append(params, ast.ParameterInfo{
				Name:    *name,
				Comment: p.getLeadingOrTrailingComment(),
			})
		}

		if p.matchAny(token.UND) {
			validate(p.consumeSeq(token.IDENTIFIER))
			addParamName(p.previous())
		} else {
			for p.matchAny(token.COMMA) { // the function takes multiple parameters
				if !p.consumeSeq(token.IDENTIFIER) {
					break
				}
				addParamName(p.previous())
			}
			if !p.consumeSeq(token.UND, token.IDENTIFIER) {
				perr(ddperror.SYN_EXPECTED_IDENTIFIER, p.peek().Range, ddperror.MsgGotExpected(p.peek(), "der letzte Parameter (und <Name>)")+"\nMeintest du vorher vielleicht 'dem Parameter' anstatt 'den Parametern'?")
			}
			addParamName(p.previous())
		}
	}

	// parse the types of the parameters
	validate(p.consumeSeq(token.VOM, token.TYP))
	firstTypeStart := p.previous()
	firstType, ref := p.parseReferenceType()
	firstTypeEnd := p.previous()
	validate(firstType != nil)
	params[0].Type = ddptypes.ParameterType{Type: firstType, IsReference: ref}
	params[0].TypeRange = token.NewRange(firstTypeStart, firstTypeEnd)

	if !singleParameter {
		i := 1
		// helper function to avoid too much repitition
		addType := func() {
			// validate the parameter type and append it
			typeStart := p.peek()
			typ, ref := p.parseReferenceType()
			typeEnd := p.previous()
			validate(typ != nil)
			if i < len(params) {
				params[i].Type = ddptypes.ParameterType{Type: typ, IsReference: ref}
				params[i].TypeRange = token.NewRange(typeStart, typeEnd)
				i++
			}
		}

		if p.matchAny(token.UND) {
			addType()
		} else {
			for p.matchAny(token.COMMA) { // parse the other parameter types
				if p.check(token.GIBT) { // , gibt indicates the end of the parameter list
					break
				}
				addType()
			}
			p.consumeSeq(token.UND)
			addType()
		}
	}

	p.consumeSeq(token.COMMA)

	invalidTypeIndex := slices.IndexFunc(params, isDefaultValue[ast.ParameterInfo])
	// we need as many parmeter names as types
	if invalidTypeIndex >= 0 {
		perr(
			ddperror.SEM_PARAM_NAME_TYPE_COUNT_MISMATCH,
			token.NewRange(&params[0].Name, p.previous()),
			fmt.Sprintf("Die Anzahl von Parametern stimmt nicht mit der Anzahl von Parameter-Typen überein (%d Parameter aber %d Typen)",
				len(params),
				invalidTypeIndex))
	}

	return params
}

// helper for funcDeclaration
func (p *parser) parseFunctionAliases(params []ast.ParameterInfo, validate func(bool)) ([]*ast.FuncAlias, [][]*token.Token) {
	// parse the alias definitions before the body to enable recursion
	validate(p.consumeSeq(token.UND, token.KANN, token.SO, token.BENUTZT, token.WERDEN, token.COLON, token.STRING)) // at least 1 alias is required
	rawAliases := make([]*token.Token, 0)
	if p.previous().Type == token.STRING {
		rawAliases = append(rawAliases, p.previous())
	}
	// append the raw aliases
	for (p.matchAny(token.COMMA) || p.matchAny(token.ODER)) && p.peek().Indent > 0 && !p.atEnd() {
		if p.consumeSeq(token.STRING) {
			rawAliases = append(rawAliases, p.previous())
		}
	}

	// map function parameters to their type (given to the alias if it is valid)
	paramTypesMap := make(map[string]ddptypes.ParameterType, len(params))
	for _, param := range params {
		if param.HasValidType() {
			paramTypesMap[param.Name.Literal] = param.Type
		}
	}

	// scan the raw aliases into tokens
	funcAliases := make([]*ast.FuncAlias, 0, len(rawAliases))
	funcAliasTokens := make([][]*token.Token, 0, len(rawAliases))
	for _, v := range rawAliases {
		// scan the raw alias withouth the ""
		didError := false
		errHandleWrapper := func(err ddperror.Error) { didError = true; p.errorHandler(err) }

		scanAndValidate := func(t token.Token, negated bool) {
			alias, err := scanner.ScanAlias(t, errHandleWrapper)
			if err != nil && didError {
				return
			}

			if len(alias) < 2 { // empty strings are not allowed (we need at least 1 token + EOF)
				p.err(ddperror.SEM_MALFORMED_ALIAS, v.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
			} else if err := p.validateFunctionAlias(alias, params); err == nil { // check that the alias fits the function
				if ok, isFun, existingAlias, pTokens := p.aliasExists(alias); ok {
					p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, v.Range, ddperror.MsgAliasAlreadyExists(v.Literal, existingAlias.Decl().Name(), isFun))
				} else {
					funcAliases = append(funcAliases, &ast.FuncAlias{Tokens: alias, Original: t, Func: nil, Args: paramTypesMap, Negated: negated})
					funcAliasTokens = append(funcAliasTokens, pTokens)
				}
			} else {
				p.errVal(*err)
			}
		}

		negMarkerStart := strings.Index(v.Literal, "<!")
		if negMarkerStart != -1 {
			if !p.isCurrentFunctionBool {
				p.err(ddperror.SEM_ALIAS_BAD_ARGS, v.Range, "Eine Funktion die kein Wahrheitswert zurück gibt, darf auch keine Negationsmarkierungen haben")
				continue
			}

			if strings.Contains(v.Literal[negMarkerStart+1:], "<!") {
				p.err(ddperror.SEM_MALFORMED_ALIAS, v.Range, "Der Alias enthält mehr als eine Aliasnegationsmarkierung")
			}

			original := v.Literal
			negMarkerEnd := (negMarkerStart + 2) + strings.IndexRune(v.Literal[negMarkerStart+2:], '>') + 1

			negatedV := *v
			negatedV.Literal = original[:negMarkerStart] + original[negMarkerStart+2:negMarkerEnd-1] + original[negMarkerEnd:]

			scanAndValidate(negatedV, true)

			v.Literal = original[:negMarkerStart] + original[negMarkerEnd:]
		}

		scanAndValidate(*v, false)
	}

	return funcAliases, funcAliasTokens
}

// helper for funcDeclaration
func (p *parser) parseOperatorOverloading(params []ast.ParameterInfo, returnType ddptypes.Type, validate func(bool)) ast.Operator {
	validate(p.consumeSeq(token.DEN, token.STRING, token.OPERATOR, token.DOT))
	operator_token := p.peekN(-3)
	operator_name := ast.TrimStringLit(operator_token)

	operator, is_operator := ast.GetOperator(operator_name)
	if !is_operator {
		p.err(ddperror.SYN_INVALID_OPERATOR, operator_token.Range, fmt.Sprintf("'%s' steht nicht für einen Operator", operator_name))
	}

	switch op := operator.(type) {
	case ast.UnaryOperator:
		if len(params) != 1 {
			p.err(ddperror.SEM_BAD_OPERATOR_PARAMS, operator_token.Range, fmt.Sprintf("Der '%s' Operator erwartet nur einen Parameter, aber hat %d bekommen", op, len(params)))
		}
	case ast.BinaryOperator:
		if len(params) != 2 {
			p.err(ddperror.SEM_BAD_OPERATOR_PARAMS, operator_token.Range, fmt.Sprintf("Der '%s' Operator erwartet zwei Parameter, aber hat %d bekommen", op, len(params)))
		}
	case ast.TernaryOperator:
		if len(params) != 3 {
			p.err(ddperror.SEM_BAD_OPERATOR_PARAMS, operator_token.Range, fmt.Sprintf("Der '%s' Operator erwartet drei Parameter, aber hat %d bekommen", op, len(params)))
		}
	case ast.CastOperator:
		if len(params) != 1 {
			p.err(ddperror.SEM_BAD_OPERATOR_PARAMS, operator_token.Range, fmt.Sprintf("Der '%s' Operator erwartet nur einen Parameter, aber hat %d bekommen", op, len(params)))
		}
	}

	if ddptypes.IsVoid(returnType) {
		p.err(ddperror.TYP_BAD_OPERATOR_RETURN_TYPE, operator_token.Range, "Ein Operator muss einen Wert zurückgeben")
	}

	return operator
}

// parses a function declaration
// startDepth is the int passed to p.peekN(n) to get to the DIE token of the declaration
func (p *parser) funcDeclaration(startDepth int) ast.Statement {
	// used later to check if the functions aliases may be added to the parsers state
	valid := true
	// helper for setting the valid flag (to simplify some big boolean expressions)
	validate := func(b bool) {
		if !b {
			valid = false
		}
	}

	// local version of p.err that also sets valid = false
	perr := func(code ddperror.Code, Range token.Range, msg string) {
		p.err(code, Range, msg)
		valid = false
	}

	begin := p.peekN(startDepth) // token.DIE
	comment := p.parseDeclComment(begin.Range)

	isPublic := p.peekN(startDepth+1).Type == token.OEFFENTLICHE

	// we need a name, so bailout if none is provided
	if !p.consumeSeq(token.IDENTIFIER) {
		return &ast.DeclStmt{
			Decl: &ast.BadDecl{
				Err: ddperror.New(ddperror.SYN_EXPECTED_IDENTIFIER, ddperror.LEVEL_ERROR, token.NewRange(begin, p.peek()), "Es wurde ein Funktions Name erwartet", p.module.FileName),
				Tok: *p.peek(),
				Mod: p.module,
			},
		}
	}
	funcName := p.previous()

	// definition of forward decl
	if p.matchAny(token.MACHT) {
		return p.funcDefinition(begin, funcName)
	}

	// early error report if the name is already used
	if _, existed, _ := p.scope().LookupDecl(funcName.Literal); existed {
		p.err(ddperror.SEM_NAME_ALREADY_DEFINED, funcName.Range, ddperror.MsgNameAlreadyExists(funcName.Literal))
	}

	// parse the parameter declaration
	params := p.parseFunctionParameters(perr, validate)

	// parse the return type declaration
	validate(p.consumeSeq(token.GIBT))
	returnTypeStart := p.previous()
	returnType := p.parseReturnType()
	returnTypeEnd := p.previous()
	if returnType == nil {
		valid = false
	}
	p.isCurrentFunctionBool = ddptypes.Equal(returnType, ddptypes.WAHRHEITSWERT)

	validate(p.consumeSeq(token.ZURÜCK, token.COMMA))

	isExternVisible := false
	externVisibleRange := token.Range{} // for the possible error message below
	if p.matchSeq(token.IST, token.EXTERN, token.SICHTBAR, token.COMMA) {
		isExternVisible = true
		externVisibleRange = token.NewRange(p.peekN(-4), p.previous())
	}

	isForwardDecl := false
	bodyStart := -1
	definedIn := &token.Token{Type: token.ILLEGAL}
	if p.matchAny(token.MACHT) {
		validate(p.consumeSeq(token.COLON))
		bodyStart = p.cur                             // save the body start-position for later, we first need to parse aliases to enable recursion
		indent := p.previous().Indent + 1             // indentation level of the function body
		for p.peek().Indent >= indent && !p.atEnd() { // advance to the alias definitions by checking the indentation
			p.advance()
		}
	} else if p.matchAny(token.WIRD) {
		validate(p.consumeSeq(token.SPÄTER, token.DEFINIERT))
		isForwardDecl = true
	} else {
		validate(p.consumeSeq(token.IST, token.IN, token.STRING, token.DEFINIERT))
		definedIn = p.peekN(-2)
		switch filepath.Ext(ast.TrimStringLit(definedIn)) {
		case ".c", ".lib", ".a", ".o":
		default:
			perr(ddperror.SEM_EXPECTED_LINKABLE_FILEPATH, definedIn.Range, fmt.Sprintf("Es wurde ein Pfad zu einer .c, .lib, .a oder .o Datei erwartet aber '%s' gefunden", definedIn.Literal))
		}
		if isExternVisible {
			perr(ddperror.SEM_UNNECESSARY_EXTERN_VISIBLE, externVisibleRange, "Es ist unnötig eine externe Funktion auch als extern sichtbar zu deklarieren")
		}
	}

	var (
		funcAliases     []*ast.FuncAlias
		funcAliasTokens [][]*token.Token
		operator        ast.Operator
	)
	if p.matchSeq(token.UND, token.ÜBERLÄDT) {
		operator = p.parseOperatorOverloading(params, returnType, validate)
	} else {
		funcAliases, funcAliasTokens = p.parseFunctionAliases(params, validate)
	}

	aliasEnd := p.cur // save the end of the function declaration for later

	if !ast.IsGlobalScope(p.scope()) {
		perr(ddperror.SEM_NON_GLOBAL_FUNCTION, begin.Range, "Es können nur globale Funktionen deklariert werden")
	}

	if !valid {
		p.cur = aliasEnd
		return &ast.DeclStmt{
			Decl: &ast.BadDecl{
				Err: p.lastError,
				Tok: *begin,
				Mod: p.module,
			},
		}
	}

	decl := &ast.FuncDecl{
		Range:           token.NewRange(begin, p.previous()),
		CommentTok:      comment,
		Tok:             *begin,
		NameTok:         *funcName,
		IsPublic:        isPublic,
		IsExternVisible: isExternVisible,
		Mod:             p.module,
		Parameters:      params,
		ReturnType:      returnType,
		ReturnTypeRange: token.NewRange(returnTypeStart, returnTypeEnd),
		Body:            nil,
		Def:             nil,
		ExternFile:      *definedIn,
		Operator:        operator,
		Aliases:         funcAliases,
	}

	for i := range funcAliases {
		funcAliases[i].Func = decl
		p.aliases.Insert(funcAliasTokens[i], funcAliases[i])
	}

	// parse the body after the aliases to enable recursion
	if bodyStart != -1 {
		p.cur = bodyStart // go back to the body
		decl.Body = p.parseFunctionBody(decl)
	} else { // the function is defined in an extern file
		// insert the name of the current function
		if existed := p.scope().InsertDecl(funcName.Literal, decl); !existed && decl.IsPublic {
			p.module.PublicDecls[decl.Name()] = decl
		}

		if !isForwardDecl {
			// add the extern declaration
			p.module.ExternalDependencies[ast.TrimStringLit(&decl.ExternFile)] = struct{}{}
		}
	}
	// operator overloads are not recursive, so insert this
	// after the body has been parsed
	if operator != nil {
		p.insertOperatorOverload(decl)
	}

	p.cur = aliasEnd // go back to the end of the function to continue parsing

	return &ast.DeclStmt{Decl: decl}
}

func (p *parser) parseFunctionBody(decl *ast.FuncDecl) *ast.BlockStmt {
	p.currentFunction = decl
	bodyTable := p.newScope() // temporary symbolTable for the function parameters
	globalScope := bodyTable.Enclosing
	// insert the name of the current function
	if existed := globalScope.InsertDecl(decl.Name(), decl); !existed && decl.IsPublic {
		p.module.PublicDecls[decl.Name()] = decl
	}
	// add the parameters to the table
	for i := range decl.Parameters {
		name := decl.Parameters[i].Name.Literal
		if !p.paramNameAllowed(&decl.Parameters[i].Name) { // check that the parameter name is not already used
			name = "$" + name
		}

		bodyTable.InsertDecl(name,
			&ast.VarDecl{
				NameTok:    decl.Parameters[i].Name,
				IsPublic:   false,
				Mod:        p.module,
				Type:       decl.Parameters[i].Type.Type,
				Range:      token.NewRange(&decl.Parameters[i].Name, &decl.Parameters[i].Name),
				CommentTok: decl.Parameters[i].Comment,
			},
		)
	}
	body := p.blockStatement(bodyTable).(*ast.BlockStmt) // parse the body with the parameters in the current table

	// check that the function has a return statement if it needs one
	if !ddptypes.IsVoid(decl.ReturnType) { // only if the function does not return void
		if len(body.Statements) < 1 { // at least the return statement is needed
			p.err(ddperror.SEM_MISSING_RETURN, body.Range, ddperror.MSG_MISSING_RETURN)
		} else {
			// the last statement must be a return statement or a todo statement
			lastStmt := body.Statements[len(body.Statements)-1]
			switch lastStmt.(type) {
			case *ast.ReturnStmt, *ast.TodoStmt:
			default:
				p.err(ddperror.SEM_MISSING_RETURN, token.NewRange(p.previous(), p.previous()), ddperror.MSG_MISSING_RETURN)
			}
		}
	}

	p.currentFunction = nil
	return body
}

func (p *parser) funcDefinition(begin, nameTok *token.Token) ast.Statement {
	decl := p.getDeclForDefinition(nameTok)
	if decl == nil {
		return nil
	}

	if !ast.IsGlobalScope(p.scope()) {
		p.err(ddperror.SEM_NON_GLOBAL_FUNCTION, p.previous().Range, "Funktionen müssen global definiert werden")
		return nil
	}

	p.consumeSeq(token.COLON)

	body := p.parseFunctionBody(decl)

	return &ast.FuncDef{
		Range: token.NewRange(begin, p.previous()),
		Tok:   *begin,
		Func:  decl,
		Body:  body,
	}
}

// gets the function decl for a definition and reports all errors in the process
// returns nil in case of error
func (p *parser) getDeclForDefinition(nameTok *token.Token) *ast.FuncDecl {
	if decl, exists, _ := p.scope().LookupDecl(nameTok.Literal); !exists {
		p.err(ddperror.SEM_NAME_UNDEFINED, nameTok.Range, fmt.Sprintf("Es wurde noch keine Funktion mit dem Namen '%s' deklariert", nameTok.Literal))
	} else if funcDecl, ok := decl.(*ast.FuncDecl); !ok {
		p.err(ddperror.SEM_BAD_NAME_CONTEXT, nameTok.Range, fmt.Sprintf("Der Name '%s' steht für eine Variable oder Struktur und nicht für eine Funktion", nameTok.Literal))
	} else if funcDecl.Mod != p.module {
		p.err(ddperror.SEM_WRONG_DECL_MODULE, nameTok.Range, "Es können nur Funktionen aus demselben Modul definiert werden")
	} else if funcDecl.Body != nil || funcDecl.ExternFile.Type != token.ILLEGAL || funcDecl.Def != nil {
		p.err(ddperror.SEM_DEFINITION_ALREADY_DEFINED, nameTok.Range, fmt.Sprintf("Die Funktion '%s' wurde bereits definiert", nameTok.Literal))
	} else {
		return funcDecl
	}

	return nil
}

func isAliasParam(t token.Token) bool   { return t.Type == token.ALIAS_PARAMETER } // helper to check for parameters
func isIllegalToken(t token.Token) bool { return t.Type == token.ILLEGAL }         // helper to check for illegal tokens

// helper for funcDeclaration to check that every parameter is provided exactly once
// and that no ILLEGAL tokens are present
func (p *parser) validateFunctionAlias(aliasTokens []token.Token, params []ast.ParameterInfo) *ddperror.Error {
	// validate that the alias contains as many parameters as the function
	if count := countElements(aliasTokens, isAliasParam); count != len(params) {
		err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			fmt.Sprintf("Der Alias braucht %d Parameter aber hat %d", len(params), count),
			p.module.FileName,
		)
		return &err
	}

	// validate that the alias does not contain illegal tokens
	if countElements(aliasTokens, isIllegalToken) > 0 {
		err := ddperror.New(
			ddperror.SEM_MALFORMED_ALIAS,
			ddperror.LEVEL_ERROR,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			"Der Alias enthält ungültige Symbole",
			p.module.FileName,
		)
		return &err
	}

	nameTypeMap := make(map[string]ddptypes.ParameterType, len(params)) // map that holds the parameter names contained in the alias and their corresponding type
	nameSet := make(map[string]struct{}, len(params))                   // set that holds the parameter names contained in the alias
	for _, param := range params {
		if param.HasValidType() {
			nameTypeMap[param.Name.Literal] = param.Type
			nameSet[param.Name.Literal] = struct{}{}
		}
	}
	// validate that each parameter is contained in the alias exactly once
	// and fill in the AliasInfo
	for i, v := range aliasTokens {
		if !isAliasParam(v) {
			continue
		}

		k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
		if _, ok := nameSet[k]; !ok {
			err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
				token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
				fmt.Sprintf("Die Funktion hat keinen Parameter mit Namen %s", k),
				p.module.FileName,
			)
			return &err
		}

		if argTyp, ok := nameTypeMap[k]; ok {
			aliasTokens[i].AliasInfo = &argTyp
			delete(nameTypeMap, k)
		} else {
			err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
				token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
				fmt.Sprintf("Der Alias enthält den Parameter %s mehrmals", k),
				p.module.FileName,
			)
			return &err
		}
	}
	return nil
}

// helper for structDeclaration to check that every field is provided once at max
// and that no ILLEGAL tokens are present
// fields should not contain bad decls
// returns wether the alias is valid and its arguments
func (p *parser) validateStructAlias(aliasTokens []token.Token, fields []*ast.VarDecl) (*ddperror.Error, map[string]ddptypes.Type) {
	// validate that the alias contains as many parameters as the struct
	if count := countElements(aliasTokens, isAliasParam); count > len(fields) {
		err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			fmt.Sprintf("Der Alias erwartet Maximal %d Parameter aber hat %d", len(fields), count),
			p.module.FileName,
		)
		return &err, nil
	}

	// validate that the alias does not contain illegal tokens
	if countElements(aliasTokens, isIllegalToken) > 0 {
		err := ddperror.New(
			ddperror.SEM_MALFORMED_ALIAS,
			ddperror.LEVEL_ERROR,
			token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
			"Der Alias enthält ungültige Symbole",
			p.module.FileName,
		)
		return &err, nil
	}

	nameTypeMap := make(map[string]ddptypes.ParameterType, len(fields)) // map that holds the parameter names contained in the alias and their corresponding type
	nameSet := make(map[string]struct{}, len(fields))                   // set that holds the parameter names contained in the alias
	args := make(map[string]ddptypes.Type, len(fields))                 // the arguments of the alias
	for _, v := range fields {
		nameTypeMap[v.Name()] = ddptypes.ParameterType{
			Type:        v.Type,
			IsReference: false, // fields are never references
		}
		args[v.Name()] = v.Type
		nameSet[v.Name()] = struct{}{}
	}
	// validate that each parameter is contained in the alias once at max
	// and fill in the AliasInfo
	for i, v := range aliasTokens {
		if !isAliasParam(v) {
			continue
		}

		k := strings.Trim(v.Literal, "<>") // remove the <> from <argname>
		if _, ok := nameSet[k]; !ok {
			err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
				token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
				fmt.Sprintf("Die Struktur hat kein Feld mit Namen %s", k),
				p.module.FileName,
			)
			return &err, nil
		}

		if argTyp, ok := nameTypeMap[k]; ok {
			aliasTokens[i].AliasInfo = &argTyp
			delete(nameTypeMap, k)
		} else {
			err := ddperror.New(ddperror.SEM_ALIAS_BAD_ARGS, ddperror.LEVEL_ERROR,
				token.NewRange(&aliasTokens[len(aliasTokens)-1], &aliasTokens[len(aliasTokens)-1]),
				fmt.Sprintf("Der Alias enthält den Parameter %s mehrmals", k),
				p.module.FileName,
			)
			return &err, nil
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

func (p *parser) parseGender() ddptypes.GrammaticalGender {
	p.consumeAny(token.EINEN, token.EINE, token.EIN)
	switch p.previous().Type {
	case token.EINEN:
		return ddptypes.MASKULIN
	case token.EINE:
		return ddptypes.FEMININ
	case token.EIN:
		return ddptypes.NEUTRUM
	default:
		return ddptypes.INVALID_GENDER
	}
}

func (p *parser) structDeclaration() ast.Declaration {
	begin := p.peekN(-3) // token.WIR
	comment := p.parseDeclComment(begin.Range)

	isPublic := p.matchAny(token.OEFFENTLICHE)
	p.consumeSeq(token.KOMBINATION, token.AUS)

	// parse the fields
	var fields []ast.Declaration
	indent := begin.Indent + 1
	for p.peek().Indent >= indent && !p.atEnd() {
		p.consumeAny(token.DER, token.DEM)
		n := -1
		if p.matchAny(token.OEFFENTLICHEN) {
			n = -2
		}
		fields = append(fields, p.varDeclaration(n, true))
		if !p.consumeSeq(token.COMMA) {
			p.advance()
		}
	}

	// deterime the grammatical gender
	gender := p.parseGender()

	if !p.consumeSeq(token.IDENTIFIER) {
		return &ast.BadDecl{
			Err: ddperror.Error{
				Code:  ddperror.SEM_NAME_UNDEFINED,
				Range: token.NewRange(p.peekN(-2), p.peek()),
				File:  p.module.FileName,
				Msg:   "Es wurde ein Kombinations Name erwartet",
			},
			Tok: *p.peek(),
			Mod: p.module,
		}
	}
	name := p.previous()

	if _, exists := p.scope().LookupType(name.Literal); exists {
		p.err(ddperror.SEM_NAME_ALREADY_DEFINED, name.Range, fmt.Sprintf("Ein Typ mit dem Namen '%s' existiert bereits", name.Literal))
	}

	p.consumeSeq(token.COMMA, token.UND, token.ERSTELLEN, token.SIE, token.SO, token.COLON, token.STRING)
	var rawAliases []*token.Token
	if p.previous().Type == token.STRING {
		rawAliases = append(rawAliases, p.previous())
	}
	for p.matchAny(token.COMMA) || p.matchAny(token.ODER) && p.peek().Indent > 0 && !p.atEnd() {
		if p.consumeSeq(token.STRING) {
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

	return decl
}

func (p *parser) typeAliasDecl() ast.Declaration {
	begin := p.previous() // Wir
	comment := p.parseDeclComment(begin.Range)

	p.consumeSeq(token.NENNEN)
	p.consumeAny(token.EIN, token.EINE, token.EINEN)
	underlyingStart := p.peek()
	underlying := p.parseType()
	underlyingEnd := p.previous()

	isPublic := p.matchAny(token.OEFFENTLICH)
	p.consumeSeq(token.AUCH)

	gender := p.parseGender()
	p.consumeSeq(token.IDENTIFIER)
	typeName := p.previous()

	p.consumeSeq(token.DOT)

	decl := &ast.TypeAliasDecl{
		Range:           token.NewRange(begin, p.previous()),
		Tok:             *begin,
		CommentTok:      comment,
		NameTok:         *typeName,
		IsPublic:        isPublic,
		Mod:             p.module,
		Underlying:      underlying,
		UnderlyingRange: token.NewRange(underlyingStart, underlyingEnd),
		Type: &ddptypes.TypeAlias{
			Name:       typeName.Literal,
			Underlying: underlying,
			GramGender: gender,
		},
	}

	return decl
}

func (p *parser) typeDefDecl() ast.Declaration {
	begin := p.peekN(-2) // Wir
	comment := p.parseDeclComment(begin.Range)

	gender := p.parseGender()
	p.consumeSeq(token.IDENTIFIER)
	typeName := p.previous()

	isPublic := p.matchAny(token.OEFFENTLICH)
	p.consumeSeq(token.ALS)

	p.consumeAny(token.EIN, token.EINE, token.EINEN)
	underlyingStart := p.peek()
	underlying := p.parseType()
	underlyingEnd := p.previous()
	underlyingRange := token.NewRange(underlyingStart, underlyingEnd)

	if ddptypes.Equal(underlying, ddptypes.VARIABLE) {
		p.err(ddperror.SEM_BAD_TYPEDEF, underlyingRange, fmt.Sprintf("Es kann kein neuer Typ als '%s' definiert werden", ddptypes.VARIABLE))
	}

	p.consumeSeq(token.DOT)

	decl := &ast.TypeDefDecl{
		Range:           token.NewRange(begin, p.previous()),
		Tok:             *begin,
		CommentTok:      comment,
		NameTok:         *typeName,
		IsPublic:        isPublic,
		Mod:             p.module,
		Underlying:      underlying,
		UnderlyingRange: underlyingRange,
		Type: &ddptypes.TypeDef{
			Name:       typeName.Literal,
			Underlying: underlying,
			GramGender: gender,
		},
	}

	return decl
}

// TODO: add support for struct aliases
func (p *parser) aliasDecl() ast.Statement {
	begin := p.peekN(-1)
	if begin.Type != token.DER {
		p.err(ddperror.SYN_GENDER_MISMATCH, begin.Range, fmt.Sprintf("Falscher Artikel, meintest du %s?", token.DER))
	}
	p.consumeSeq(token.STRING)
	aliasTok := p.previous()
	p.consumeSeq(token.STEHT, token.FÜR, token.DIE, token.FUNKTION, token.IDENTIFIER)
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
	paramTypes := make(map[string]ddptypes.ParameterType, 4)
	for _, param := range funDecl.Parameters {
		if param.HasValidType() {
			paramTypes[param.Name.Literal] = param.Type
		}
	}

	// scan the raw alias withouth the ""
	var alias *ast.FuncAlias
	var pTokens []*token.Token
	if aliasTokens, err := scanner.ScanAlias(*aliasTok, func(err ddperror.Error) { p.err(err.Code, err.Range, err.Msg) }); err == nil && len(aliasTokens) < 2 { // empty strings are not allowed (we need at leas 1 token + EOF)
		p.err(ddperror.SEM_MALFORMED_ALIAS, aliasTok.Range, "Ein Alias muss mindestens 1 Symbol enthalten")
	} else if err := p.validateFunctionAlias(aliasTokens, funDecl.Parameters); err == nil { // check that the alias fits the function
		if ok, isFun, existingAlias, toks := p.aliasExists(aliasTokens); ok {
			p.err(ddperror.SEM_ALIAS_ALREADY_TAKEN, aliasTok.Range, ddperror.MsgAliasAlreadyExists(aliasTok.Literal, existingAlias.Decl().Name(), isFun))
		} else {
			alias = &ast.FuncAlias{Tokens: aliasTokens, Original: *aliasTok, Func: funDecl, Args: paramTypes}
			pTokens = toks
		}
	} else {
		p.errVal(*err)
	}

	p.consumeSeq(token.DOT)

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

func containsName(params []ast.ParameterInfo, name string) bool {
	for i := range params {
		if params[i].Name.Literal == name {
			return true
		}
	}
	return false
}
