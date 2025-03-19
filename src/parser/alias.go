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

	callOrLiteralFromAlias := func(alias ast.Alias, args map[string]ast.Expression, instantiation ast.Declaration) ast.Expression {
		if fnalias, isFuncAlias := alias.(*ast.FuncAlias); isFuncAlias {
			fun := fnalias.Func
			if genericFunInstantiation, isFun := instantiation.(*ast.FuncDecl); isFun {
				fun = genericFunInstantiation
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
		return &ast.StructLiteral{
			Range:  token.NewRange(&p.tokens[start], p.previous()),
			Tok:    p.tokens[start],
			Struct: stralias.Struct,
			Args:   args,
		}
	}

	createInstantiationError := func(instantiation *ast.FuncDecl, errs []ddperror.Error) ddperror.Error {
		msg := strings.Builder{}
		msg.WriteString(fmt.Sprintf("Es gab Fehler beim Instanziieren der generischen Funktion '%s':", instantiation.Name()))

		for _, err := range errs {
			msg.WriteString("\n\t")
			msg.WriteString(err.String())
		}

		return ddperror.New(ddperror.SEM_ERROR_INSTANTIATING_GENERIC_FUNCTION, ddperror.LEVEL_ERROR, p.previous().Range, msg.String(), p.module.FileName)
	}

	type checkAliasResult struct {
		alias         ast.Alias
		errs          []ddperror.Error
		instantiation ast.Declaration
		args          map[string]ast.Expression
	}

	var mostFitting *checkAliasResult
	cached_args := make(map[cachedArgKey]*cachedArg, 4)

	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		args, instantiation, errs := p.checkAlias(matchedAliases[i], true, start, cached_args)
		if mostFitting == nil {
			mostFitting = &checkAliasResult{matchedAliases[i], errs, instantiation, args}
		}

		if args != nil && len(errs) == 0 {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
			return callOrLiteralFromAlias(matchedAliases[i], args, instantiation)
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments

	// generic aliases may not be called with typeSensitive = false
	if funcAlias, ok := mostFitting.alias.(*ast.FuncAlias); ok && ast.IsGeneric(funcAlias.Func) {
		p.errVal(createInstantiationError(mostFitting.instantiation.(*ast.FuncDecl), mostFitting.errs))
		p.cur = start
		return nil
	}

	args, instantiation, errs := p.checkAlias(mostFitting.alias, false, start, cached_args)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return callOrLiteralFromAlias(mostFitting.alias, args, instantiation)
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
func (p *parser) checkAlias(mAlias ast.Alias, typeSensitive bool, start int, cached_args map[cachedArgKey]*cachedArg) (map[string]ast.Expression, ast.Declaration, []ddperror.Error) {
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
				return nil, nil, reported_errors
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
					return nil, nil, reported_errors
				}
			}

			args[argName] = cached_arg.Arg
			p.decrease() // to not skip a token
		}
		p.advance() // ignore non-argument tokens
	}

	funcDecl, isFuncDecl := mAlias.Decl().(*ast.FuncDecl)
	if isFuncDecl && ast.IsGeneric(funcDecl) {
		returnType := funcDecl.ReturnType
		if generic, isGeneric := ddptypes.CastGeneric(returnType); isGeneric {
			returnType = genericTypes[generic.Name]
		}

		instantiation, errs := p.InstantiateGenericFunction(funcDecl, genericTypes, returnType)
		reported_errors = append(reported_errors, errs...)
		return args, instantiation, reported_errors
	}

	return args, nil, reported_errors
}

// instantiates a generic function with the given types
// genericTypes maps GenericTypeName -> Type
// returns the new instantiation and any errors that occured during instatiation
func (p *parser) InstantiateGenericFunction(genericFunc *ast.FuncDecl, genericTypes map[string]ddptypes.Type, returnType ddptypes.Type) (*ast.FuncDecl, []ddperror.Error) {
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

	instantiations := genericFunc.Generic.Instantiations[p.module]
	for _, instantiation := range instantiations {
		if slices.EqualFunc(instantiation.Parameters, parameters, func(a, b ast.ParameterInfo) bool {
			return ddptypes.ParamTypesEqual(a.Type, b.Type)
		}) {
			return instantiation, nil
		}
	}

	decl := *genericFunc
	decl.Parameters = parameters
	decl.ReturnType = returnType
	decl.Generic = nil
	decl.GenericDecl = genericFunc
	decl.Mod = p.module

	context := p.generateGenericContext(genericFunc.Generic.Context, parameters, genericTypes)

	errorCollector := ddperror.Collector{}
	declParser := &parser{
		tokens:          genericFunc.Generic.Tokens,
		errorHandler:    errorCollector.GetHandler(),
		module:          genericFunc.Mod,
		aliases:         context.Aliases,
		currentFunction: &decl,
		Operators:       context.Operators,
	}
	// prepare the resolver and typechecker with the inbuild symbols and types
	declParser.resolver = resolver.New(declParser.module, declParser.Operators, declParser.errorHandler, &declParser.panicMode)
	declParser.typechecker = typechecker.New(declParser.module, declParser.Operators, declParser.errorHandler, declParser, &declParser.panicMode)

	declParser.setScope(context.Symbols)

	// add the instantiation to prevent recursion
	genericFunc.Generic.Instantiations[p.module] = append(genericFunc.Generic.Instantiations[p.module], &decl)

	declParser.advance() // skip the colon for blockStatement()
	decl.Body = declParser.blockStatement(declParser.scope()).(*ast.BlockStmt)
	declParser.ensureReturnStatementPresent(&decl, decl.Body)

	if errorCollector.DidError() {
		// remove the instantiation as we errored
		genericFunc.Generic.Instantiations[p.module] = slices.DeleteFunc(genericFunc.Generic.Instantiations[p.module], func(f *ast.FuncDecl) bool { return f == &decl })
		return &decl, errorCollector.Errors
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
