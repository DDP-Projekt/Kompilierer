/*
This file defines the functions used to parse Aliases
*/
package parser

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	at "github.com/DDP-Projekt/Kompilierer/src/parser/alias_trie"
	"github.com/DDP-Projekt/Kompilierer/src/parser/resolver"
	"github.com/DDP-Projekt/Kompilierer/src/parser/typechecker"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

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
			for range n {
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
				if !p.matchAny(token.INT, token.FLOAT, token.IDENTIFIER, token.SYMBOL) {
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
	sortAliases(matchedAliases)

	callOrLiteralFromAlias := func(alias ast.Alias, args map[string]ast.Expression, funcInstantiation *ast.FuncDecl, structTypeInstantiation *ddptypes.StructType) ast.Expression {
		if fnalias, isFuncAlias := alias.(*ast.FuncAlias); isFuncAlias {
			fun := fnalias.Func
			if funcInstantiation != nil {
				fun = funcInstantiation
			}

			fnCall := &ast.FuncCall{
				Range: token.NewRange(&p.tokens[start], p.previous()),
				Tok:   p.tokens[start],
				Name:  fnalias.Func.Name(),
				Func:  fun,
				Args:  args,
			}

			if fnalias.Negated {
				return &ast.UnaryExpr{
					Range:    fnCall.Range,
					Tok:      p.tokens[start],
					Operator: ast.UN_NOT,
					Rhs:      fnCall,
				}
			}

			return fnCall
		}

		stralias := alias.(*ast.StructAlias)

		structType := stralias.Struct.Type
		if structTypeInstantiation != nil {
			structType = structTypeInstantiation
		}

		return &ast.StructLiteral{
			Range:  token.NewRange(&p.tokens[start], p.previous()),
			Tok:    p.tokens[start],
			Struct: stralias.Struct,
			Type:   structType.(*ddptypes.StructType),
			Args:   args,
		}
	}

	type checkAliasResult struct {
		alias                   ast.Alias
		errs                    []ddperror.Error
		funcInstantiation       *ast.FuncDecl
		structTypeInstantiation *ddptypes.StructType
		args                    map[string]ast.Expression
	}

	var mostFitting *checkAliasResult
	cached_args := make(map[cachedArgKey]*cachedArg, 4)

	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		args, funcInstantiation, structTypeInstantiation, errs := p.checkAlias(matchedAliases[i], true, start, cached_args)
		if mostFitting == nil {
			mostFitting = &checkAliasResult{matchedAliases[i], errs, funcInstantiation, structTypeInstantiation, args}
		}

		if args != nil && len(errs) == 0 {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
			return callOrLiteralFromAlias(matchedAliases[i], args, funcInstantiation, structTypeInstantiation)
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments

	// generic aliases may not be called with typeSensitive = false
	if funcAlias, ok := mostFitting.alias.(*ast.FuncAlias); ok && ast.IsGeneric(funcAlias.Func) {
		p.errVal(ddperror.Error{
			Code:                 ddperror.SEM_ERROR_INSTANTIATING_GENERIC_FUNCTION,
			Level:                ddperror.LEVEL_ERROR,
			Range:                token.NewRange(&p.tokens[start], p.previous()),
			Msg:                  fmt.Sprintf("Es gab Fehler beim Instanziieren der generischen Funktion '%s'", funcAlias.Func.Name()),
			File:                 p.module.FileName,
			WrappedGenericErrors: mostFitting.errs,
		})
		p.cur = start
		return nil
	}

	args, funcInstantiation, structTypeInstantiation, errs := p.checkAlias(mostFitting.alias, false, start, cached_args)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return callOrLiteralFromAlias(mostFitting.alias, args, funcInstantiation, structTypeInstantiation)
}

// sorts aliases by
//   - their length
//   - how many reference parameters they take
//   - how many generic parameters they take
func sortAliases(matchedAliases []ast.Alias) {
	sort.Slice(matchedAliases, func(i, j int) bool {
		toksi, toksj := matchedAliases[i].GetTokens(), matchedAliases[j].GetTokens()
		if len(toksi) != len(toksj) {
			return len(toksi) > len(toksj)
		}

		// Sort by functions that take references up and generics down

		countRefAndGenericArgs := func(params map[string]ddptypes.ParameterType) (refs, gen int) {
			for _, paramType := range params {
				if paramType.IsReference {
					refs++
				}
				if ddptypes.IsGeneric(paramType.Type) {
					gen++
				}
			}
			return refs, gen
		}

		refNi, genNi := countRefAndGenericArgs(matchedAliases[i].GetArgs())
		refNj, genNj := countRefAndGenericArgs(matchedAliases[j].GetArgs())
		if genNi != genNj {
			return genNi < genNj // generic functions are sorted down, not up
		}

		return refNi > refNj
	})
}

// used for caching by checkAlias
// represents an argument that was already parsed
type cachedArg struct {
	Arg     ast.Expression   // expression (might be an assignable)
	Errors  []ddperror.Error // the errors that occured while parsing the argument
	exprEnd int              // where the expression was over (p.cur for the token after)
}

// used for caching by checkAlias
// represents a key for a cached argument
type cachedArgKey struct {
	cur         int  // the start pos of that argument
	isReference bool // wether the argument was parsed with p.assignable() or p.expression()
}

// used by p.alias
// attempts to evaluate the arguments for the passed alias and checks if types match
// also unifies generic types and attempts to instantiate generic functions
// returns nil if argument and parameter types don't match
// it also returns all errors that might have occured while doing so
func (p *parser) checkAlias(mAlias ast.Alias, typeSensitive bool, start int, cached_args map[cachedArgKey]*cachedArg) (map[string]ast.Expression, *ast.FuncDecl, *ddptypes.StructType, []ddperror.Error) {
	p.cur = start
	args := make(map[string]ast.Expression, 4)
	reported_errors := make([]ddperror.Error, 0)
	mAliasTokens := mAlias.GetTokens()
	mAliasArgs := mAlias.GetArgs()

	genericTypes := make(map[string]ddptypes.Type, 4)

	for i, l := 0, len(mAliasTokens); i < l && mAliasTokens[i].Type != token.EOF; i++ {
		tok := &mAliasTokens[i]

		if tok.Type == token.ALIAS_PARAMETER {
			argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
			paramType := mAliasArgs[argName]           // type of the current parameter

			pType := p.peek().Type
			// early return if a non-identifier expression is passed as reference
			if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
				return nil, nil, nil, reported_errors
			}

			// create the key for the argument
			cached_arg_key := cachedArgKey{cur: p.cur, isReference: paramType.IsReference}
			cached_arg, ok := cached_args[cached_arg_key]

			if !ok { // if the argument was not already parsed
				cached_arg = &cachedArg{}
				exprStart := p.cur
				isGrouping := false
				switch pType {
				case token.INT, token.FLOAT, token.TRUE, token.FALSE, token.CHAR, token.STRING, token.IDENTIFIER, token.SYMBOL:
					p.advance() // single-token argument
				case token.NEGATE:
					p.advance()
					p.matchAny(token.INT, token.FLOAT, token.IDENTIFIER, token.SYMBOL)
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
					module:      p.module,
					aliases:     p.aliases,
					resolver:    p.resolver,
					typechecker: p.typechecker,
					Operators:   p.Operators,
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

				underlyingParamType := ddptypes.UnifyGenericType(typ, paramType, genericTypes)

				if !ddptypes.Equal(typ, underlyingParamType) {
					didMatch = false
				} else if ass, ok := cached_arg.Arg.(*ast.Indexing);                                // string-indexings may not be passed as char-reference
				paramType.IsReference && ddptypes.Equal(underlyingParamType, ddptypes.BUCHSTABE) && // if the parameter is a char-reference
					ok { // and the argument is a indexing
					lhs := p.typechecker.EvaluateSilent(ass.Lhs)
					if ddptypes.Equal(lhs, ddptypes.TEXT) { // check if the lhs is a string
						didMatch = false
					}
				}

				if !didMatch {
					return nil, nil, nil, reported_errors
				}
			}

			args[argName] = cached_arg.Arg
			p.decrease() // to not skip a token
		}
		p.advance() // ignore non-argument tokens
	}

	funcDecl, isFuncDecl := mAlias.Decl().(*ast.FuncDecl)
	if isFuncDecl && ast.IsGeneric(funcDecl) {
		instantiation, errs := p.InstantiateGenericFunction(funcDecl, genericTypes)
		reported_errors = append(reported_errors, errs...)
		return args, instantiation, nil, reported_errors
	}

	// TODO: validate that all field types can be instantiated and are initialized
	structDecl, isStructDecl := mAlias.Decl().(*ast.StructDecl)
	if isStructDecl && ast.IsGeneric(structDecl) {
		typeParams, err := p.fillAndVerifyGenericStructInstantiationParams(structDecl, genericTypes)
		if err != nil {
			p.errVal(*err)
			return args, nil, nil, append(reported_errors, *err)
		}

		instantiation := ddptypes.GetInstantiatedStructType(structDecl.Type.(*ddptypes.GenericStructType), typeParams)
		if instantiation == nil {
			reported_errors = append(reported_errors, ddperror.New(
				ddperror.TYP_COULD_NOT_INSTANTIATE_GENERIC,
				ddperror.LEVEL_ERROR,
				token.NewRange(&p.tokens[start],
					p.previous()),
				fmt.Sprintf("Der generische Typ %s konnte nicht mit den Typparametern %s instanziiert werden", structDecl.Type.String(), typeParams),
				p.module.GetIncludeFilename(),
			))
		}
		return args, nil, instantiation, reported_errors
	}

	return args, nil, nil, reported_errors
}

// instantiates a generic function with the given types
// genericTypes maps GenericTypeName -> Type
// returns the new instantiation and any errors that occured during instatiation
func (p *parser) InstantiateGenericFunction(genericFunc *ast.FuncDecl, genericTypes map[string]ddptypes.Type) (*ast.FuncDecl, []ddperror.Error) {
	if !ast.IsGeneric(genericFunc) {
		panic("tried to instantiate non-generic function")
	}

	parameters := make([]ast.ParameterInfo, len(genericFunc.Parameters))
	// assign the types to the parameters
	// unification of generic types must have taken place beforehand
	// meaning types must contain the correct type for each parameter
	for i, param := range genericFunc.Parameters {
		parameters[i] = param
		_, isGeneric := ddptypes.CastDeeplyNestedGenerics(param.Type.Type)
		if !isGeneric {
			continue
		}

		parameters[i].Type.Type = ddptypes.GetInstantiatedType(parameters[i].Type.Type, genericTypes)
	}

	genericModule := p.genericModule
	if genericModule == nil {
		genericModule = p.module
	}

	if ast.IsExternFunc(genericFunc) {
		genericModule = genericFunc.Module()
	}

	instantiations := genericFunc.Generic.Instantiations[genericModule]

	for _, instantiation := range instantiations {
		if slices.EqualFunc(instantiation.Parameters, parameters, func(a, b ast.ParameterInfo) bool {
			return ddptypes.ParamTypesEqual(a.Type, b.Type)
		}) {
			return instantiation, nil
		}
	}

	decl := *genericFunc
	decl.Parameters = parameters
	decl.ReturnType = ddptypes.GetInstantiatedType(genericFunc.ReturnType, genericTypes)
	decl.Generic = nil
	decl.GenericInstantiation = &ast.GenericInstantiationInfo{
		GenericDecl: genericFunc,
		Types:       genericTypes,
	}
	decl.Mod = genericModule

	if ast.IsExternFunc(genericFunc) {
		// add the instantiation to prevent recursion
		genericFunc.Generic.Instantiations[genericModule] = append(genericFunc.Generic.Instantiations[p.module], &decl)
		return &decl, nil
	}

	context := p.generateGenericContext(genericFunc.Generic.Context, parameters, genericTypes)

	errorCollector := ddperror.Collector{}
	declParser := &parser{
		tokens:                genericFunc.Generic.Tokens,
		errorHandler:          errorCollector.GetHandler(),
		module:                genericFunc.Mod,
		genericModule:         genericModule,
		aliases:               context.Aliases,
		currentFunction:       &decl,
		isCurrentFunctionBool: ddptypes.Equal(decl.ReturnType, ddptypes.WAHRHEITSWERT),
		Operators:             context.Operators,
	}
	// prepare the resolver and typechecker with the inbuild symbols and types
	declParser.resolver = resolver.New(declParser.module, declParser.Operators, declParser.errorHandler, &declParser.panicMode)
	declParser.typechecker = typechecker.New(declParser.module, declParser.Operators, declParser.errorHandler, declParser, &declParser.panicMode)

	declParser.setScope(context.Symbols)

	// add the instantiation to prevent recursion
	genericFunc.Generic.Instantiations[genericModule] = append(genericFunc.Generic.Instantiations[genericModule], &decl)

	declParser.advance() // skip the colon for blockStatement()
	decl.Body = declParser.blockStatement(declParser.scope()).(*ast.BlockStmt)
	declParser.ensureReturnStatementPresent(&decl, decl.Body)

	if errorCollector.DidError() {
		// remove the instantiation as we errored
		genericFunc.Generic.Instantiations[genericModule] = slices.DeleteFunc(genericFunc.Generic.Instantiations[genericModule], func(f *ast.FuncDecl) bool { return f == &decl })
	}

	return &decl, errorCollector.Errors
}

func (p *parser) generateGenericContext(fun ast.GenericContext, params []ast.ParameterInfo, genericTypes map[string]ddptypes.Type) ast.GenericContext {
	aliases := at.Copy(p.aliases)

	matched := fun.Aliases.Search(func(i int, t *token.Token) (*token.Token, bool) {
		return t, true
	})
	for _, alias := range matched {
		aliases.Insert(alias.GetKey(), alias)
	}

	symbols := ast.NewSymbolTable(newGenericSymbolTable(p.scope(), fun.Symbols, genericTypes))

	// add the parameters to the table
	for i := range params {
		name := params[i].Name.Literal
		if !p.paramNameAllowed(&params[i].Name) { // check that the parameter name is not already used
			name = "$" + name
		}

		symbols.InsertDecl(name,
			&ast.VarDecl{
				NameTok:    params[i].Name,
				IsPublic:   false,
				Mod:        p.module,
				Type:       ddptypes.GetInstantiatedType(params[i].Type.Type, genericTypes),
				Range:      token.NewRange(&params[i].Name, &params[i].Name),
				CommentTok: params[i].Comment,
			},
		)
	}

	operators := maps.Clone(p.Operators)
	maps.Copy(operators, fun.Operators)

	return ast.GenericContext{
		Symbols:   symbols,
		Aliases:   aliases,
		Operators: operators,
	}
}

// TODO: unit test this
func (p *parser) fillAndVerifyGenericStructInstantiationParams(structDecl *ast.StructDecl, genericTypes map[string]ddptypes.Type) ([]ddptypes.Type, *ddperror.Error) {
	genericStructType := structDecl.Type.(*ddptypes.GenericStructType)

	for _, field := range structDecl.Fields {
		field, ok := field.(*ast.VarDecl)
		if !ok || !ddptypes.IsGeneric(field.Type) {
			continue
		}

		if _, wasUnified := genericTypes[field.Type.String()]; !wasUnified {
			genericTypes[field.Type.String()] = field.InitType
		}
	}

	for _, field := range genericStructType.StructType.Fields {
		if _, wasUnified := genericTypes[field.Type.String()]; ddptypes.IsGeneric(field.Type) && !wasUnified {
			// TODO: better error range
			err := ddperror.New(ddperror.TYP_GENERIC_TYPE_NOT_UNIFIED, ddperror.LEVEL_ERROR, p.previous().Range,
				fmt.Sprintf("Der generische Typ %s des Feldes %s konnte nicht unifiziert werden", field.Type, field.Name),
				p.module.FileName)
			return nil, &err
		}
	}

	typeParams := make([]ddptypes.Type, 0, len(genericStructType.GenericTypes))
	for _, param := range genericStructType.GenericTypes {
		typeParams = append(typeParams, genericTypes[param.Name])
	}

	return typeParams, nil
}
