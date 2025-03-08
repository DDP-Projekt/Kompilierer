package compiler

func stackTop[T any](s []T) T {
	return s[len(s)-1]
}

func stackPush[T any](s *[]T, e T) {
	*s = append(*s, e)
}

func stackPop[T any](s *[]T) T {
	e := (*s)[len(*s)-1]
	*s = (*s)[:len(*s)-1]
	return e
}
