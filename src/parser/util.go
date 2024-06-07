/*
This file contains general utility functions used in the parser
*/
package parser

import (
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
		return *t1.AliasInfo == *t2.AliasInfo
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal == t2.Literal
	}

	return true
}

// check if two tokens are equal for alias overwriting (e.g. the ALIAS_PARAMETER type is not checked)
func genericAliasTokenEqual(t1, t2 *token.Token) bool {
	if t1.Type == token.ALIAS_PARAMETER && t2.Type == token.ALIAS_PARAMETER {
		return true
	}
	return tokenEqual(t1, t2)
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

		return t1.AliasInfo.Type.String() < t2.AliasInfo.Type.String()
	case token.IDENTIFIER, token.SYMBOL, token.INT, token.FLOAT, token.CHAR, token.STRING:
		return t1.Literal < t2.Literal
	}

	return false
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
