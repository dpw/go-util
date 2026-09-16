// Package set provides a generic set type, [Set], backed by a
// map[T]struct{}.  This has the same characteristics as direct use of
// a map[T]struct{} to represent a set, but Set provides convenience
// methods and functions to allow clearer, more concise code.
//
// Because a Set is simply a map, it can be treated as one when
// convenient.  For example, len(s) gives the number of elements, and
// the elements can be iterated over with a simple for-range statement:
//
//	for x := range s {
//		// ...
//	}
//
// A nil Set is a valid, empty, read-only set, just like any nil map.
package set

import (
	"cmp"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"

	"github.com/dpw/go-util/iterx"
)

type Set[T comparable] map[T]struct{}

// New constructs a new Set[T] containing the given elements.  With
// no arguments, the result is nil.
func New[T comparable](vals ...T) Set[T] {
	return FromSlice(vals)
}

// FromSlice constructs a new Set[T] containing the elements of the
// given slice.  If the slice is empty, the result is nil.
func FromSlice[T comparable](s []T) Set[T] {
	return Collect(slices.Values(s))
}

// Collect constructs a new Set[T] containing the elements from the
// given sequence.  If the sequence is empty, the result is nil.
func Collect[T comparable](vals iter.Seq[T]) (res Set[T]) {
	for val := range vals {
		res.AddNE(val)
	}
	return res
}

// Clone returns a copy of the set.  The result will be nil if and
// only if the target set is nil.
func (s Set[T]) Clone() Set[T] {
	return maps.Clone(s)
}

// Contains determines if the set contains the given element.
func (s Set[T]) Contains(val T) bool {
	_, present := s[val]
	return present
}

// ContainsAll determines whether the set contains every element of the
// given sequence.  It is true for an empty sequence.
func (s Set[T]) ContainsAll(vals iter.Seq[T]) bool {
	for val := range vals {
		if !s.Contains(val) {
			return false
		}
	}
	return true
}

// ContainsAny determines whether the set contains any elements from
// the given sequence.  It is false for an empty sequence.
func (s Set[T]) ContainsAny(vals iter.Seq[T]) bool {
	for val := range vals {
		if s.Contains(val) {
			return true
		}
	}
	return false
}

// Elements returns an iterator over the elements of the set.
func (s Set[T]) Elements() iter.Seq[T] {
	return maps.Keys(s)
}

// Add adds an element to the set.  As with maps, if the set is nil, a
// panic will occur.
func (s Set[T]) Add(val T) {
	s[val] = struct{}{}
}

// AddNE adds an element to an addressable set.  If the set is nil, it
// will be changed to an empty set before adding the element.
func (s *Set[T]) AddNE(val T) {
	if *s == nil {
		*s = make(Set[T])
	}
	(*s)[val] = struct{}{}
}

// AddAll adds a collection of elements to the set.  If the set is nil and the
// sequence is non-empty, a panic will occur.
func (s Set[T]) AddAll(vals iter.Seq[T]) {
	for val := range vals {
		s.Add(val)
	}
}

// AddAllNE adds a collection of elements to an addressable set.  If
// the collection is non-empty and the set is nil, it will be changed
// to an empty set before adding the elements.
func (s *Set[T]) AddAllNE(vals iter.Seq[T]) {
	for val := range vals {
		s.AddNE(val)
	}
}

// Remove removes an element from the set, if present.  Removing from
// a nil set is a no-op.
func (s Set[T]) Remove(val T) {
	delete(s, val)
}

// RemoveAll removes a collection of elements from the set.  Removing
// from a nil set is a no-op.
func (s Set[T]) RemoveAll(vals iter.Seq[T]) {
	for val := range vals {
		s.Remove(val)
	}
}

// Equal determines whether the two sets contain the same elements.  A
// nil set is equal to any other empty set.
func (s1 Set[T]) Equal(s2 Set[T]) bool {
	return len(s1) == len(s2) && s1.IsSubsetOf(s2)
}

// IsSubsetOf determines whether every element of the set is also an
// element of s2.
func (s1 Set[T]) IsSubsetOf(s2 Set[T]) bool {
	return s2.ContainsAll(s1.Elements())
}

// Intersect returns a new set containing the elements present in both
// sets.  If the intersection is empty, the result is nil.  The inputs
// are not modified.
func (s1 Set[T]) Intersect(s2 Set[T]) (res Set[T]) {
	// Always iterate over the smaller set.
	if len(s2) < len(s1) {
		s1, s2 = s2, s1
	}
	for val := range s1 {
		if s2.Contains(val) {
			res.AddNE(val)
		}
	}
	return res
}

// Subtract returns a new set containing the elements of the set that
// are not present in s2.  If the result is empty, it is nil.  The
// inputs are not modified.
func (s1 Set[T]) Subtract(s2 Set[T]) (res Set[T]) {
	for val := range s1 {
		if !s2.Contains(val) {
			res.AddNE(val)
		}
	}
	return res
}

// Union returns a new set containing the elements present in either
// set.  If both sets are empty, the result is nil.  The inputs are not
// modified.
func (s1 Set[T]) Union(s2 Set[T]) (res Set[T]) {
	res.AddAllNE(s1.Elements())
	res.AddAllNE(s2.Elements())
	return res
}

// NilToEmpty returns the set unchanged if it is non-nil.  Otherwise
// it returns a fresh empty set.
func (s Set[T]) NilToEmpty() Set[T] {
	if s == nil {
		return make(Set[T])
	}
	return s
}

// Reachable generates the set of elements reachable from the elements
// of the input set under a given relation.  The relation is
// represented as a function which takes a source element and yields a
// sequence of destination elements.  The function will be called once
// for each element reached.
func (s Set[T]) Reachable(rel func(T) iter.Seq[T]) (res Set[T]) {
	var todo []T
	for val := range s {
		res.AddNE(val)
		todo = append(todo, val)
	}

	for len(todo) != 0 {
		src := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		for dst := range rel(src) {
			if !res.Contains(dst) {
				res.AddNE(dst)
				todo = append(todo, dst)
			}
		}
	}

	return res
}

func (s Set[T]) Format(f fmt.State, verb rune) {
	var close []byte
	if verb == 'v' && f.Flag('#') {
		// "%#v" should yield a Go-syntax representation.
		t := reflect.TypeOf(s).Key()
		if len(s) == 0 {
			if s == nil {
				// Similarly to e.g. []int(nil)
				fmt.Fprintf(f, "set.Set[%s](nil)", t)
			} else {
				// "set.New[...]()" yields nil, so empty sets
				// should format as "set.Set[...]{}.
				fmt.Fprintf(f, "set.Set[%s]{}", t)
			}
			return
		}

		// Produce a "set.New[...](...)" expression.
		fmt.Fprintf(f, "set.New[%s](", t)
		close = closeParen
	} else {
		// For other formatting cases, format as "{...}".  All
		// empty sets are formatted as "{}", even nil.  This
		// is consistent with slices.
		f.Write(openBrace)
		close = closeBrace
	}

	formatStr := fmt.FormatString(f, verb)
	first := true
	for val := range s.formatElements() {
		if !first {
			f.Write(comma)
		}
		fmt.Fprintf(f, formatStr, val)
		first = false
	}

	f.Write(close)
}

var openBrace = []byte("{")
var closeBrace = []byte("}")
var closeParen = []byte(")")
var comma = []byte(", ")

func (s Set[T]) formatElements() iter.Seq[T] {
	// If there is more than a single element in the set, and the
	// type of the elements is a ordered, sort them for printing.
	if len(s) > 1 {
		if f, ok := reflectCmpFuncs[reflect.TypeOf(s).Key().Kind()]; ok {
			// The elements need to be converted to
			// reflect.Values for sortnig, and then
			// converted back in the result.
			vals := slices.Collect(iterx.Transform(s.Elements(), func(val T) reflect.Value {
				return reflect.ValueOf(val)
			}))
			slices.SortFunc(vals, f)
			return iterx.Transform(slices.Values(vals), func(val reflect.Value) T {
				return val.Interface().(T)
			})
		}
	}

	return s.Elements()
}

var reflectCmpInt = reflectCmpFunc((reflect.Value).Int)
var reflectCmpUint = reflectCmpFunc((reflect.Value).Uint)
var reflectCmpFloat = reflectCmpFunc((reflect.Value).Float)

var reflectCmpFuncs = map[reflect.Kind]func(a, b reflect.Value) int{
	reflect.Int:   reflectCmpInt,
	reflect.Int8:  reflectCmpInt,
	reflect.Int16: reflectCmpInt,
	reflect.Int32: reflectCmpInt,
	reflect.Int64: reflectCmpInt,

	reflect.Uint:   reflectCmpUint,
	reflect.Uint8:  reflectCmpUint,
	reflect.Uint16: reflectCmpUint,
	reflect.Uint32: reflectCmpUint,
	reflect.Uint64: reflectCmpUint,

	reflect.Float32: reflectCmpFloat,
	reflect.Float64: reflectCmpFloat,

	reflect.String: reflectCmpFunc((reflect.Value).String),
}

func reflectCmpFunc[T cmp.Ordered](f func(reflect.Value) T) func(a, b reflect.Value) int {
	return func(a, b reflect.Value) int {
		return cmp.Compare(f(a), f(b))
	}
}
