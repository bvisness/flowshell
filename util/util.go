package util

func Tern[T any](cond bool, a, b T) T {
	if cond {
		return a
	} else {
		return b
	}
}

func Map[T1, T2 any](s []T1, f func(v T1) T2) []T2 {
	if s == nil {
		return nil
	}

	res := make([]T2, len(s))
	for i := range s {
		res[i] = f(s[i])
	}
	return res
}
