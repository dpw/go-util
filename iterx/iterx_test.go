package iterx

import (
	"iter"
	"maps"
	"slices"
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestEmpty(t *testing.T) {
	assert.Zero(t, len(slices.Collect(Empty[int]())))

	// Ranging over it yields nothing, so the body never runs.
	calls := 0
	for range Empty[string]() {
		calls++
	}
	assert.Zero(t, calls)
}

func TestTransform(t *testing.T) {
	got := slices.Collect(Transform(slices.Values([]int{1, 2, 3}),
		func(n int) string { return strconv.Itoa(n * 2) }))
	assert.Equal(t, []string{"2", "4", "6"}, got)
}

func TestTransform_Empty(t *testing.T) {
	got := slices.Collect(Transform(slices.Values([]int(nil)),
		func(n int) int { return n }))
	assert.Zero(t, len(got))
}

// Transform is lazy: it must stop pulling from the source and stop
// calling f as soon as the consumer stops ranging.
func TestTransform_EarlyStop(t *testing.T) {
	calls := 0
	f := func(n int) int { calls++; return n }

	counter, consumed := countUp()

	var got []int
	for v := range Transform(counter, f) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}

	assert.Equal(t, []int{0, 1, 2}, got)
	assert.Equal(t, 3, calls, "f called once per yielded value")
	assert.Equal(t, 3, *consumed, "source not advanced past the break")
}

func TestFilter(t *testing.T) {
	got := slices.Collect(Filter(slices.Values([]int{1, 2, 3, 4, 5, 6}),
		func(n int) bool { return n%2 == 0 }))
	assert.Equal(t, []int{2, 4, 6}, got)
}

func TestFilter_Empty(t *testing.T) {
	got := slices.Collect(Filter(slices.Values([]int(nil)),
		func(int) bool { return true }))
	assert.Zero(t, len(got))
}

// Filter is lazy: it must stop pulling from the source as soon as the
// consumer stops ranging.
func TestFilter_EarlyStop(t *testing.T) {
	counter, consumed := countUp()

	var got []int
	for v := range Filter(counter, func(int) bool { return true }) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}

	assert.Equal(t, []int{0, 1, 2}, got)
	assert.Equal(t, 3, *consumed, "source not advanced past the break")
}

func TestFlatten(t *testing.T) {
	got := slices.Collect(Flatten(slices.Values([]int{1, 2, 3}),
		func(n int) iter.Seq[int] { return slices.Values([]int{n, -n}) }))
	assert.Equal(t, []int{1, -1, 2, -2, 3, -3}, got)
}

// An input whose f yields nothing contributes nothing and does not break the
// concatenation of the others.
func TestFlatten_EmptyInner(t *testing.T) {
	got := slices.Collect(Flatten(slices.Values([]int{1, 2, 3, 4}),
		func(n int) iter.Seq[int] {
			if n%2 != 0 {
				return Empty[int]()
			}
			return slices.Values([]int{n})
		}))
	assert.Equal(t, []int{2, 4}, got)
}

// Flatten is lazy: stopping mid-way through an inner sequence must stop the
// walk entirely, without pulling the next value from the outer source.
func TestFlatten_EarlyStop(t *testing.T) {
	counter, consumed := countUp()
	f := func(n int) iter.Seq[int] { return slices.Values([]int{n, n}) }

	var got []int
	for v := range Flatten(counter, f) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}

	assert.Equal(t, []int{0, 0, 1}, got)
	assert.Equal(t, 2, *consumed, "outer source not advanced past the break")
}

func TestTransform2(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	got := maps.Collect(Transform2(maps.All(m),
		func(k string, v int) (string, int) { return k + "!", v * 10 }))
	assert.Equal(t, map[string]int{"a!": 10, "b!": 20, "c!": 30}, got)
}

func TestTransform2_EarlyStop(t *testing.T) {
	calls := 0
	f := func(k, v int) (int, int) { calls++; return k, v * 10 }

	// Enumerate turns the counter into a Seq2 yielding (0,0),(1,1),(2,2),...
	counter, consumed := countUp()

	var gotK, gotV []int
	for k, v := range Transform2(Enumerate(counter), f) {
		gotK = append(gotK, k)
		gotV = append(gotV, v)
		if len(gotK) == 2 {
			break
		}
	}

	assert.Equal(t, []int{0, 1}, gotK)
	assert.Equal(t, []int{0, 10}, gotV) // 0*10, 1*10
	assert.Equal(t, 2, calls)
	assert.Equal(t, 2, *consumed)
}

func TestFilter2(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4}
	got := maps.Collect(Filter2(maps.All(m),
		func(_ string, v int) bool { return v%2 == 0 }))
	assert.Equal(t, map[string]int{"b": 2, "d": 4}, got)
}

func TestFilter2_Empty(t *testing.T) {
	got := maps.Collect(Filter2(maps.All(map[string]int(nil)),
		func(string, int) bool { return true }))
	assert.Zero(t, len(got))
}

// Filter2 is lazy: it must stop pulling from the source as soon as the
// consumer stops ranging.
func TestFilter2_EarlyStop(t *testing.T) {
	counter, consumed := countUp()

	var got []int
	for _, v := range Filter2(Enumerate(counter), func(int, int) bool { return true }) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}

	assert.Equal(t, []int{0, 1, 2}, got)
	assert.Equal(t, 3, *consumed, "source not advanced past the break")
}

func TestTransform2To1(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	got := slices.Sorted(slices.Values(
		slices.Collect(Transform2To1(maps.All(m),
			func(k string, v int) string { return k + strconv.Itoa(v) }))))
	assert.Equal(t, []string{"a1", "b2", "c3"}, got)
}

func TestTransform2To1_EarlyStop(t *testing.T) {
	calls := 0
	f := func(k, v int) int { calls++; return k + v }

	// Enumerate turns the counter into a Seq2 yielding (0,0),(1,1),(2,2),...
	counter, consumed := countUp()

	var got []int
	for v := range Transform2To1(Enumerate(counter), f) {
		got = append(got, v)
		if len(got) == 2 {
			break
		}
	}

	assert.Equal(t, []int{0, 2}, got) // 0+0, 1+1
	assert.Equal(t, 2, calls)
	assert.Equal(t, 2, *consumed)
}

func TestEnumerate(t *testing.T) {
	var keys []int
	var vals []string
	for i, v := range Enumerate(slices.Values([]string{"a", "b", "c"})) {
		keys = append(keys, i)
		vals = append(vals, v)
	}
	assert.Equal(t, []int{0, 1, 2}, keys)
	assert.Equal(t, []string{"a", "b", "c"}, vals)
}

func TestEnumerate_Empty(t *testing.T) {
	calls := 0
	for range Enumerate(Empty[int]()) {
		calls++
	}
	assert.Zero(t, calls)
}

// Enumerate is lazy: it must stop pulling from the source as soon as the
// consumer stops ranging.
func TestEnumerate_EarlyStop(t *testing.T) {
	counter, consumed := countUp()

	var got []int
	for _, v := range Enumerate(counter) {
		got = append(got, v)
		if len(got) == 3 {
			break
		}
	}

	assert.Equal(t, []int{0, 1, 2}, got)
	assert.Equal(t, 3, *consumed, "source not advanced past the break")
}

// countUp returns the infinite sequence 0, 1, 2, ... together with a pointer
// to a count of how many values have been pulled from it. It lets a test check
// that a lazy operation stops consuming its source as soon as the consumer
// stops ranging.
func countUp() (iter.Seq[int], *int) {
	consumed := 0
	seq := func(yield func(int) bool) {
		for i := 0; ; i++ {
			consumed = i + 1
			if !yield(i) {
				return
			}
		}
	}
	return seq, &consumed
}
