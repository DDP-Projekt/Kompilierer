/*
This file defines the functions used to parse Aliases
*/
package parser

import (
	"sort"
	"strings"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddperror"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

func (p *parser) alias() ast.Expression {
	if p.strictAliases {
		return p.strict_alias()
	}
	return p.new_alias()
}

func (p *parser) new_alias() ast.Expression {
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
	sort.Slice(matchedAliases, func(i, j int) bool {
		toksi, toksj := matchedAliases[i].GetTokens(), matchedAliases[j].GetTokens()
		if len(toksi) != len(toksj) {
			return len(toksi) > len(toksj)
		}

		// Sort by functions that take references up
		// TODO: improve this heuristic

		countRefArgs := func(params map[string]ddptypes.ParameterType) (n int) {
			for _, paramType := range params {
				if paramType.IsReference {
					n++
				}
			}
			return n
		}

		refNi, refNj := countRefArgs(matchedAliases[i].GetArgs()), countRefArgs(matchedAliases[j].GetArgs())
		return refNi > refNj
	})

	// a argument that was already parsed
	type cachedArg struct {
		Arg     ast.Expression   // expression (might be an assignable)
		Errors  []ddperror.Error // the errors that occured while parsing the argument
		exprEnd int              // where the expression was over (p.cur for the token after)
	}

	// a key for a cached argument
	type cachedArgKey struct {
		cur         int  // the start pos of that argument
		isReference bool // wether the argument was parsed with p.assignable() or p.expression()
	}

	// used for the algorithm below to parse each argument only once
	cached_args := make(map[cachedArgKey]*cachedArg, 4)
	// attempts to evaluate the arguments for the passed alias and checks if types match
	// returns nil if argument and parameter types don't match
	// similar to the alogrithm above
	// it also returns all errors that might have occured while doing so
	checkAlias := func(mAlias ast.Alias, typeSensitive bool) (map[string]ast.Expression, []ddperror.Error) {
		p.cur = start
		args := make(map[string]ast.Expression, 4)
		reported_errors := make([]ddperror.Error, 0)
		mAliasTokens := mAlias.GetTokens()
		mAliasArgs := mAlias.GetArgs()

		for i, l := 0, len(mAliasTokens); i < l && mAliasTokens[i].Type != token.EOF; i++ {
			tok := &mAliasTokens[i]

			if tok.Type == token.ALIAS_PARAMETER {
				argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
				paramType := mAliasArgs[argName]           // type of the current parameter

				pType := p.peek().Type
				// early return if a non-identifier expression is passed as reference
				if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
					return nil, reported_errors
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
						module: &ast.Module{
							FileName: p.module.FileName,
						},
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
					if !ddptypes.Equal(typ, paramType.Type) {
						didMatch = false
					} else if ass, ok := cached_arg.Arg.(*ast.Indexing);                           // string-indexings may not be passed as char-reference
					paramType.IsReference && ddptypes.Equal(paramType.Type, ddptypes.BUCHSTABE) && // if the parameter is a char-reference
						ok { // and the argument is a indexing
						lhs := p.typechecker.EvaluateSilent(ass.Lhs)
						if ddptypes.Equal(lhs, ddptypes.TEXT) { // check if the lhs is a string
							didMatch = false
						}
					}

					if !didMatch {
						return nil, reported_errors
					}
				}

				args[argName] = cached_arg.Arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}
		return args, reported_errors
	}

	callOrLiteralFromAlias := func(alias ast.Alias, args map[string]ast.Expression) ast.Expression {
		if fnalias, isFuncAlias := alias.(*ast.FuncAlias); isFuncAlias {
			fnCall := &ast.FuncCall{
				Range: token.NewRange(&p.tokens[start], p.previous()),
				Tok:   p.tokens[start],
				Name:  fnalias.Func.Name(),
				Func:  fnalias.Func,
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

	var mostFitting *ast.Alias
	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		args, errs := checkAlias(matchedAliases[i], true)
		if args != nil && mostFitting == nil {
			mostFitting = &matchedAliases[i]
		}

		if args != nil && len(errs) == 0 {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
			return callOrLiteralFromAlias(matchedAliases[i], args)
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments
	if mostFitting == nil {
		mostFitting = &matchedAliases[0]
	}
	args, errs := checkAlias(*mostFitting, false)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return callOrLiteralFromAlias(*mostFitting, args)
}

func (p *parser) strict_alias() ast.Expression {
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
	sort.Slice(matchedAliases, func(i, j int) bool {
		toksi, toksj := matchedAliases[i].GetTokens(), matchedAliases[j].GetTokens()
		if len(toksi) != len(toksj) {
			return len(toksi) > len(toksj)
		}

		// Sort by functions that take references up
		// TODO: improve this heuristic

		countRefArgs := func(params map[string]ddptypes.ParameterType) (n int) {
			for _, paramType := range params {
				if paramType.IsReference {
					n++
				}
			}
			return n
		}

		refNi, refNj := countRefArgs(matchedAliases[i].GetArgs()), countRefArgs(matchedAliases[j].GetArgs())
		return refNi > refNj
	})

	// a argument that was already parsed
	type cachedArg struct {
		Arg     ast.Expression   // expression (might be an assignable)
		Errors  []ddperror.Error // the errors that occured while parsing the argument
		exprEnd int              // where the expression was over (p.cur for the token after)
	}

	// a key for a cached argument
	type cachedArgKey struct {
		cur         int  // the start pos of that argument
		isReference bool // wether the argument was parsed with p.assignable() or p.expression()
	}

	// used for the algorithm below to parse each argument only once
	cached_args := make(map[cachedArgKey]*cachedArg, 4)
	// attempts to evaluate the arguments for the passed alias and checks if types match
	// returns nil if argument and parameter types don't match
	// similar to the alogrithm above
	// it also returns all errors that might have occured while doing so
	checkAlias := func(mAlias ast.Alias, typeSensitive bool) (map[string]ast.Expression, []ddperror.Error) {
		p.cur = start
		args := make(map[string]ast.Expression, 4)
		reported_errors := make([]ddperror.Error, 0)
		mAliasTokens := mAlias.GetTokens()
		mAliasArgs := mAlias.GetArgs()

		for i, l := 0, len(mAliasTokens); i < l && mAliasTokens[i].Type != token.EOF; i++ {
			tok := &mAliasTokens[i]

			if tok.Type == token.ALIAS_PARAMETER {
				argName := strings.Trim(tok.Literal, "<>") // remove the <> from the alias parameter
				paramType := mAliasArgs[argName]           // type of the current parameter

				pType := p.peek().Type
				// early return if a non-identifier expression is passed as reference
				if typeSensitive && paramType.IsReference && pType != token.IDENTIFIER && pType != token.LPAREN {
					return nil, reported_errors
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
						module: &ast.Module{
							FileName: p.module.FileName,
						},
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
					if !ddptypes.Equal(typ, paramType.Type) {
						didMatch = false
					} else if ass, ok := cached_arg.Arg.(*ast.Indexing);                           // string-indexings may not be passed as char-reference
					paramType.IsReference && ddptypes.Equal(paramType.Type, ddptypes.BUCHSTABE) && // if the parameter is a char-reference
						ok { // and the argument is a indexing
						lhs := p.typechecker.EvaluateSilent(ass.Lhs)
						if ddptypes.Equal(lhs, ddptypes.TEXT) { // check if the lhs is a string
							didMatch = false
						}
					}

					if !didMatch {
						return nil, reported_errors
					}
				}

				args[argName] = cached_arg.Arg
				p.decrease() // to not skip a token
			}
			p.advance() // ignore non-argument tokens
		}
		return args, reported_errors
	}

	callOrLiteralFromAlias := func(alias ast.Alias, args map[string]ast.Expression) ast.Expression {
		if fnalias, isFuncAlias := alias.(*ast.FuncAlias); isFuncAlias {
			fnCall := &ast.FuncCall{
				Range: token.NewRange(&p.tokens[start], p.previous()),
				Tok:   p.tokens[start],
				Name:  fnalias.Func.Name(),
				Func:  fnalias.Func,
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

	var mostFitting *ast.Alias
	// search for the longest possible alias whose parameter types match
	for i := range matchedAliases {
		args, errs := checkAlias(matchedAliases[i], true)
		if args != nil && mostFitting == nil {
			mostFitting = &matchedAliases[i]
		}

		if args != nil && len(errs) == 0 {
			// log the errors that occured while parsing
			apply(p.errorHandler, errs)
			return callOrLiteralFromAlias(matchedAliases[i], args)
		}
	}

	// no alias matched the type requirements
	// so we take the longest one (most likely to be wanted)
	// and "call" it so that the typechecker will report
	// errors for the arguments
	if mostFitting == nil {
		mostFitting = &matchedAliases[0]
	}
	args, errs := checkAlias(*mostFitting, false)

	// log the errors that occured while parsing
	apply(p.errorHandler, errs)

	return callOrLiteralFromAlias(*mostFitting, args)
}
