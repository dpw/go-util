package cmpx

import (
	"cmp"
)

// By returns a comparison function that orders values of type V by the
// key extracted by f, comparing keys with [cmp.Compare]. It adapts a
// key function for use with APIs that take a comparison function, such
// as [slices.SortFunc] and [slices.SortedFunc].
func By[V any, K cmp.Ordered](f func(V) K) func(a, b V) int {
	return func(a, b V) int {
		return cmp.Compare(f(a), f(b))
	}
}
