package slashcommands

import "cmp"

func CmpPtrDefault[T cmp.Ordered](v1, v2 *T, defaultVal T) int {
	av, bv := defaultVal, defaultVal
	if v1 != nil {
		av = *v1
	}
	if v2 != nil {
		bv = *v2
	}

	return cmp.Compare(av, bv)
}
