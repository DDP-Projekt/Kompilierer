package ddptypes

func mapSlice[T, U any](s []T, mapper func(T) U) []U {
	result := make([]U, len(s))
	for i := range s {
		result[i] = mapper(s[i])
	}
	return result
}
