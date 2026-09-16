package set

import (
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/alecthomas/assert/v2"

	"github.com/dpw/go-util/iterx"
)

func TestNew(t *testing.T) {
	assert.True(t, New[int]() == nil)
	assert.Equal(t, Set[int]{42: struct{}{}}, New(42))
	assert.Equal(t, Set[int]{7: struct{}{}, 42: struct{}{}}, New(7, 42, 7))
}

func TestFromSlice(t *testing.T) {
	assert.True(t, FromSlice[string](nil) == nil)
	assert.True(t, FromSlice([]string{}) == nil)
	assert.Equal(t, Set[string]{"a": struct{}{}}, FromSlice([]string{"a"}))
	assert.Equal(t, Set[string]{"a": struct{}{}, "b": struct{}{}},
		FromSlice([]string{"a", "b", "a"}))
}

func TestCollect(t *testing.T) {
	assert.True(t, Collect(iterx.Empty[string]()) == nil)

	assert.Equal(t, Set[int]{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		Collect(slices.Values([]int{1, 2, 3, 1})))

	m := map[int]string{1: "a", 2: "b", 3: "a"}
	assert.Equal(t, Set[int]{1: struct{}{}, 2: struct{}{}, 3: struct{}{}},
		Collect(maps.Keys(m)))
	assert.Equal(t, Set[string]{"a": struct{}{}, "b": struct{}{}},
		Collect(maps.Values(m)))
}

func TestClone(t *testing.T) {
	// Check the nilness requirements.
	assert.True(t, Set[int](nil).Clone() == nil)
	assert.False(t, make(Set[int]).Clone() == nil)

	// The clone has the same contents but is a distinct map: mutating
	// one doesn't affect the other.
	s := New(1, 2, 3)
	c := s.Clone()
	assert.Equal(t, s, c)
	assert.False(t, reflect.ValueOf(s).Pointer() == reflect.ValueOf(c).Pointer())

	c.Add(4)
	assert.Equal(t, New(1, 2, 3), s)
	assert.Equal(t, New(1, 2, 3, 4), c)
}

func TestContains(t *testing.T) {
	// A nil set contains nothing.
	assert.False(t, Set[int](nil).Contains(1))

	s := New(1, 2, 3)
	assert.True(t, s.Contains(1))
	assert.True(t, s.Contains(3))
	assert.False(t, s.Contains(4))
}

func TestContainsAll(t *testing.T) {
	// The empty sequence is always contained, even by a nil set.
	assert.True(t, Set[int](nil).ContainsAll(iterx.Empty[int]()))
	assert.True(t, New(1, 2, 3).ContainsAll(iterx.Empty[int]()))

	s := New(1, 2, 3)
	assert.True(t, s.ContainsAll(slices.Values([]int{1, 3})))
	assert.True(t, s.ContainsAll(slices.Values([]int{1, 2, 3})))
	assert.False(t, s.ContainsAll(slices.Values([]int{1, 4})))
	assert.False(t, Set[int](nil).ContainsAll(slices.Values([]int{1})))
}

func TestContainsAny(t *testing.T) {
	// The empty sequence is never contained.
	assert.False(t, Set[int](nil).ContainsAny(iterx.Empty[int]()))
	assert.False(t, New(1, 2, 3).ContainsAny(iterx.Empty[int]()))

	s := New(1, 2, 3)
	assert.True(t, s.ContainsAny(slices.Values([]int{3, 4})))
	assert.True(t, s.ContainsAny(slices.Values([]int{1, 2, 3})))
	assert.False(t, s.ContainsAny(slices.Values([]int{4, 5})))
	assert.False(t, Set[int](nil).ContainsAny(slices.Values([]int{1})))
}

func TestElements(t *testing.T) {
	// A nil set yields no elements.
	assert.Equal(t, []int(nil), slices.Sorted(Set[int](nil).Elements()))

	s := New(1, 2, 3)
	assert.Equal(t, []int{1, 2, 3}, slices.Sorted(s.Elements()))

	// Consumption can stop early without exhausting the sequence.
	var first int
	for v := range s.Elements() {
		first = v
		break
	}
	assert.True(t, s.Contains(first))
}

func TestAdd(t *testing.T) {
	s := make(Set[int])
	s.Add(1)
	assert.Equal(t, New(1), s)
	s.Add(2)
	s.Add(1)
	assert.Equal(t, New(1, 2), s)
}

func TestAddNE(t *testing.T) {
	var s Set[int]
	s.AddNE(1)
	assert.Equal(t, New(1), s)
	s.AddNE(2)
	s.AddNE(1)
	assert.Equal(t, New(1, 2), s)
}

func TestAddAll(t *testing.T) {
	s := make(Set[int])
	s.AddAll(slices.Values([]int{1, 2}))
	assert.Equal(t, New(1, 2), s)
	s.AddAll(slices.Values([]int{3, 1}))
	assert.Equal(t, New(1, 2, 3), s)
}

func TestAddAllNE(t *testing.T) {
	var s Set[int]
	assert.True(t, s == nil)
	s.AddAllNE(slices.Values([]int{1, 2}))
	assert.Equal(t, New(1, 2), s)
	s.AddAllNE(slices.Values([]int{3, 1}))
	assert.Equal(t, New(1, 2, 3), s)
}

func TestRemove(t *testing.T) {
	// Removing from a nil set is a no-op.
	Set[int](nil).Remove(1)

	s := New(1, 2, 3)
	s.Remove(2)
	assert.Equal(t, New(1, 3), s)
	// Removing an absent element leaves the set unchanged.
	s.Remove(2)
	assert.Equal(t, New(1, 3), s)
}

func TestRemoveAll(t *testing.T) {
	// Removing from a nil set is a no-op.
	Set[int](nil).RemoveAll(slices.Values([]int{1, 2}))

	s := New(1, 2, 3, 4)
	s.RemoveAll(slices.Values([]int{2, 4, 5}))
	assert.Equal(t, New(1, 3), s)
}

func TestEqual(t *testing.T) {
	// A nil set equals any other empty set.
	assert.True(t, Set[int](nil).Equal(nil))
	assert.True(t, Set[int](nil).Equal(make(Set[int])))
	assert.True(t, New[int]().Equal(nil))

	assert.True(t, New(1, 2, 3).Equal(New(3, 2, 1)))
	assert.False(t, New(1, 2, 3).Equal(New(1, 2)))
	assert.False(t, New(1, 2).Equal(New(1, 2, 3)))
	// Same size, different elements.
	assert.False(t, New(1, 2, 3).Equal(New(1, 2, 4)))
}

func TestIsSubsetOf(t *testing.T) {
	// The empty set is a subset of every set, including itself.
	assert.True(t, Set[int](nil).IsSubsetOf(nil))
	assert.True(t, Set[int](nil).IsSubsetOf(New(1, 2)))

	// A non-empty set is not a subset of the empty set.
	assert.False(t, New(1).IsSubsetOf(nil))

	// A proper subset, an equal set, and a superset.
	assert.True(t, New(1, 2).IsSubsetOf(New(1, 2, 3)))
	assert.True(t, New(1, 2, 3).IsSubsetOf(New(1, 2, 3)))
	assert.False(t, New(1, 2, 3).IsSubsetOf(New(1, 2)))

	// Same size, but not a subset.
	assert.False(t, New(1, 2, 3).IsSubsetOf(New(1, 2, 4)))
}

func TestIntersect(t *testing.T) {
	// A disjoint intersection is nil.
	assert.True(t, New(1, 2).Intersect(New(3, 4)) == nil)
	assert.True(t, New(1, 2).Intersect(nil) == nil)
	assert.True(t, Set[int](nil).Intersect(New(1, 2)) == nil)

	assert.Equal(t, New(2, 3), New(1, 2, 3).Intersect(New(2, 3, 4)))
	// The result is the same regardless of the order of the operands.
	assert.Equal(t, New(2, 3), New(2, 3, 4).Intersect(New(1, 2, 3)))

	// The inputs are not modified.
	a, b := New(1, 2, 3), New(2, 3, 4)
	a.Intersect(b)
	assert.Equal(t, New(1, 2, 3), a)
	assert.Equal(t, New(2, 3, 4), b)
}

func TestSubtract(t *testing.T) {
	// Subtracting everything, or from nothing, gives nil.
	assert.True(t, New(1, 2).Subtract(New(1, 2, 3)) == nil)
	assert.True(t, Set[int](nil).Subtract(New(1, 2)) == nil)
	// Subtracting nothing leaves the elements, but in a new set.
	assert.Equal(t, New(1, 2), New(1, 2).Subtract(nil))

	assert.Equal(t, New(1), New(1, 2, 3).Subtract(New(2, 3, 4)))

	// The inputs are not modified.
	a, b := New(1, 2, 3), New(2, 3, 4)
	a.Subtract(b)
	assert.Equal(t, New(1, 2, 3), a)
	assert.Equal(t, New(2, 3, 4), b)
}

func TestUnion(t *testing.T) {
	// The union of two empty sets is nil.
	assert.True(t, Set[int](nil).Union(nil) == nil)

	assert.Equal(t, New(1, 2), Set[int](nil).Union(New(1, 2)))
	assert.Equal(t, New(1, 2), New(1, 2).Union(nil))
	assert.Equal(t, New(1, 2, 3, 4), New(1, 2, 3).Union(New(2, 3, 4)))

	// The inputs are not modified.
	a, b := New(1, 2, 3), New(2, 3, 4)
	a.Union(b)
	assert.Equal(t, New(1, 2, 3), a)
	assert.Equal(t, New(2, 3, 4), b)
}

func TestNilToEmpty(t *testing.T) {
	var s Set[int]
	got := s.NilToEmpty()
	assert.True(t, got != nil)
	assert.Zero(t, len(got))

	// A non-nil Set is returned unchanged.  It doesn't just have
	// the same contents, it is the same Set.  Maps aren't
	// comparable with ==, so compare the underlying map pointers
	// via reflect.
	s = New(1, 2)
	assert.True(t, reflect.ValueOf(s).Pointer() == reflect.ValueOf(s.NilToEmpty()).Pointer())
}

func TestReachable(t *testing.T) {
	graph := map[int][]int{
		1: {2, 3},
		2: {3},
		4: {5},
	}
	rel := func(n int) iter.Seq[int] { return slices.Values(graph[n]) }

	// An empty starting set reaches nothing.
	assert.True(t, Set[int](nil).Reachable(rel) == nil)

	// The result includes the starting elements and everything reachable
	// from them by following rel transitively.
	assert.Equal(t, New(1, 2, 3), New(1).Reachable(rel))

	// A node with no outgoing edges reaches only itself.
	assert.Equal(t, New(3), New(3).Reachable(rel))

	// Multiple roots union their reachable sets.
	assert.Equal(t, New(1, 2, 3, 4, 5), New(1, 4).Reachable(rel))

	// Cycles terminate: each node is visited once.
	cyclic := map[int][]int{1: {2}, 2: {1}}
	assert.Equal(t, New(1, 2), New(1).Reachable(func(n int) iter.Seq[int] {
		return slices.Values(cyclic[n])
	}))
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		format string
		arg    any
		want   string
	}{
		// The default verb formats as "{...}".  Elements of an ordered
		// type are sorted, regardless of insertion order.
		{"v/int", "%v", New(3, 1, 2), "{1, 2, 3}"},
		{"v/int-negative", "%v", New(2, -10, -1), "{-10, -1, 2}"},
		{"v/uint", "%v", New[uint](3, 1, 2), "{1, 2, 3}"},
		{"v/float", "%v", New(2.5, 0.5, 1.5), "{0.5, 1.5, 2.5}"},
		{"v/string", "%v", New("b", "a", "c"), "{a, b, c}"},

		// A nil or empty set formats as "{}".
		{"v/nil", "%v", Set[int](nil), "{}"},
		{"v/empty", "%v", make(Set[int]), "{}"},

		// A single element needs no sorting.
		{"v/single", "%v", New(5), "{5}"},

		// The verb and its flags apply to each element.
		{"d/int", "%d", New(3, 1, 2), "{1, 2, 3}"},
		{"q/string", "%q", New("b", "a", "c"), `{"a", "b", "c"}`},
		{"s/string", "%s", New("b", "a", "c"), "{a, b, c}"},

		// "%#v" yields a Go-syntax representation.
		{"#v/int", "%#v", New(3, 1, 2), "set.New[int](1, 2, 3)"},
		{"#v/string", "%#v", New("b", "a"), `set.New[string]("a", "b")`},
		{"#v/nil", "%#v", Set[int](nil), "set.Set[int](nil)"},
		{"#v/empty", "%#v", Set[int]{}, "set.Set[int]{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, fmt.Sprintf(tt.format, tt.arg))
		})
	}
}

func TestFormatUnordered(t *testing.T) {
	// Structs are comparable but not ordered, so their elements are not
	// sorted; with more than one element, either order is acceptable.
	type point struct{ X, Y int }
	got := fmt.Sprintf("%v", New(point{1, 2}, point{3, 4}))
	assert.True(t, got == "{{1 2}, {3 4}}" || got == "{{3 4}, {1 2}}",
		"unexpected formatting: %s", got)

	// A single element is deterministic.
	assert.Equal(t, "{{1 2}}", fmt.Sprintf("%v", New(point{1, 2})))
}
