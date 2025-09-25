package util

func Tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	} else {
		return b
	}
}
