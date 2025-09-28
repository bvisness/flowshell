package util

import "golang.org/x/exp/constraints"

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

func Must1[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Must1B[T any](v T, ok bool) T {
	if !ok {
		panic("expected ok")
	}
	return v
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	} else {
		return b
	}
}

func Max[T constraints.Ordered](a, b T) T {
	if a > b {
		return a
	} else {
		return b
	}
}
