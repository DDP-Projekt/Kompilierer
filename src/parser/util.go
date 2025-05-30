/*
This file contains general utility functions used in the parser
*/
package parser

import (
	"slices"

	"github.com/DDP-Projekt/Kompilierer/src/ast"
	"github.com/DDP-Projekt/Kompilierer/src/ddptypes"
	"github.com/DDP-Projekt/Kompilierer/src/token"
)

// check if two tokens are equal for alias matching
func tokenEqual(t1, t2 *token.Token) bool {
	if t1 == t2 {
		return true
	}

	if t1.Type != t2.Type {
		return false
	}

	switch t1.Type {
	case token.ALIAS_PARAMETER:
		return ddptypes.ParamTypesEqual(*t1.AliasInfo, *t2.AliasInfo)
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal == t2.Literal
	}

	return true
}

// converts b to 1 or 0
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// reports wether t1 < t2
// tokens are first sorted by their type
// if the types are equal they are sorted by their AliasInfo or Literal
// for AliasInfo: name < list < reference
// for Identifiers: t1.Literal < t2.Literal
func tokenLess(t1, t2 *token.Token) bool {
	if t1.Type != t2.Type {
		return t1.Type < t2.Type
	}

	switch t1.Type {
	case token.ALIAS_PARAMETER:
		if t1.AliasInfo.IsReference != t2.AliasInfo.IsReference {
			return boolToInt(t1.AliasInfo.IsReference) < boolToInt(t2.AliasInfo.IsReference)
		}

		isList1, isList2 := ddptypes.IsList(t1.AliasInfo.Type), ddptypes.IsList(t2.AliasInfo.Type)
		if isList1 != isList2 {
			return boolToInt(isList1) < boolToInt(isList2)
		}

		return ddptypes.GetUnderlying(t1.AliasInfo.Type).String() < ddptypes.GetUnderlying(t2.AliasInfo.Type).String()
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal < t2.Literal
	}

	return false
}

// returns gender for DER, DIE, DAS, DEM
func genderFromArticle(t token.TokenType, isField bool) ddptypes.GrammaticalGender {
	if isField {
		switch t {
		case token.DEM:
			return ddptypes.MASKULIN
		case token.DER:
			return ddptypes.FEMININ
		}
		return ddptypes.INVALID_GENDER
	}

	switch t {
	case token.DER:
		return ddptypes.MASKULIN
	case token.DIE:
		return ddptypes.FEMININ
	case token.DAS:
		return ddptypes.NEUTRUM
	}
	return ddptypes.INVALID_GENDER
}

// returns articles DEM, DER, DIE, DAS
func articleFromGender(g ddptypes.GrammaticalGender, isField bool) token.TokenType {
	switch g {
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
	return token.ILLEGAL
}

func genderFromForPronoun(t token.TokenType) ddptypes.GrammaticalGender {
	switch t {
	case token.JEDEN:
		return ddptypes.MASKULIN
	case token.JEDE:
		return ddptypes.FEMININ
	case token.JEDES:
		return ddptypes.NEUTRUM
	}
	return ddptypes.INVALID_GENDER
}

func forPronounFromGender(gender ddptypes.GrammaticalGender) token.TokenType {
	switch gender {
	case ddptypes.MASKULIN:
		return token.JEDEN
	case ddptypes.FEMININ:
		return token.JEDE
	case ddptypes.NEUTRUM:
		return token.JEDES
	}
	return token.ILLEGAL // unreachable
}

// returns articles einen, einem, ein
func articleFromGender2Akkusativ(gender ddptypes.GrammaticalGender) token.TokenType {
	switch gender {
	case ddptypes.MASKULIN:
		return token.EINEN
	case ddptypes.FEMININ:
		return token.EINE
	case ddptypes.NEUTRUM:
		return token.EIN
	}
	return token.ILLEGAL // unreachable
}

// returns gender for einen, eine, ein
func genderFromArticle2Akkusativ(tok token.TokenType) ddptypes.GrammaticalGender {
	switch tok {
	case token.EINEN:
		return ddptypes.MASKULIN
	case token.EINE:
		return ddptypes.FEMININ
	case token.EIN:
		return ddptypes.NEUTRUM
	}
	return ddptypes.INVALID_GENDER // unreachable
}

// returns articles einen, einem, ein
func articleFromGender2Dativ(gender ddptypes.GrammaticalGender) token.TokenType {
	switch gender {
	case ddptypes.MASKULIN, ddptypes.NEUTRUM:
		return token.EINEM
	case ddptypes.FEMININ:
		return token.EINER
	}
	return token.ILLEGAL // unreachable
}

// returns gender for einen, eine, ein
func genderFromArticle2Dativ(tok token.TokenType) []ddptypes.GrammaticalGender {
	switch tok {
	case token.EINEM:
		return []ddptypes.GrammaticalGender{ddptypes.MASKULIN, ddptypes.NEUTRUM}
	case token.EINER:
		return []ddptypes.GrammaticalGender{ddptypes.FEMININ}
	}
	return []ddptypes.GrammaticalGender{ddptypes.INVALID_GENDER} // unreachable
}

// counts all elements in the slice which fulfill the provided predicate function
func countElements[T any](elements []T, pred func(T) bool) (count int) {
	for _, v := range elements {
		if pred(v) {
			count++
		}
	}
	return count
}

// applies fun to every element in slice
func apply[T any](fun func(T), slice []T) {
	for i := range slice {
		fun(slice[i])
	}
}

// converts a slice of a subtype to it's basetype
// T must be convertible to U
func toInterfaceSlice[T any, U any](slice []T) []U {
	result := make([]U, len(slice))
	for i := range slice {
		result[i] = any(slice[i]).(U)
	}
	return result
}

// keeps only those elements for which the filter function returns true
func filterSlice[T any](slice []T, filter func(t T) bool) []T {
	result := make([]T, 0, len(slice))
	for i := range slice {
		if filter(slice[i]) {
			result = append(result, slice[i])
		}
	}
	return result
}

func toPointerSlice[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

func isDefaultValue[T comparable](v T) bool {
	var default_value T
	return v == default_value
}

func operatorReturnTypeEqual(a, b ddptypes.Type) bool {
	_, isGen1 := ddptypes.CastDeeplyNestedGenerics(a)
	_, isGen2 := ddptypes.CastDeeplyNestedGenerics(b)

	if isGen1 || isGen2 {
		return isGen1 && isGen2
	}

	return ddptypes.Equal(a, b)
}

func operatorParameterTypesEqual(pi1, pi2 []ast.ParameterInfo) bool {
	return slices.EqualFunc(pi1, pi2, func(pi1, pi2 ast.ParameterInfo) bool {
		_, isGen1 := ddptypes.CastDeeplyNestedGenerics(pi1.Type.Type)
		_, isGen2 := ddptypes.CastDeeplyNestedGenerics(pi2.Type.Type)

		if isGen1 || isGen2 {
			return isGen1 && isGen2 && pi1.Type.IsReference == pi2.Type.IsReference
		}

		return ddptypes.ParamTypesEqual(pi1.Type, pi2.Type)
	})
}

func mapSlice[T, U any](s []T, mapper func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}
