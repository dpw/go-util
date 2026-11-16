package set_test

import (
	"fmt"
	"iter"
	"slices"

	"github.com/dpw/go-util/set"
)

func Example() {
	s := set.New(1, 2, 3)
	s.Add(4)
	fmt.Println(s.Contains(2))
	fmt.Println(s.Union(set.New(4, 5)))
	// Output:
	// true
	// {1, 2, 3, 4, 5}
}

func ExampleSet_Reachable() {
	// A directed graph represented as an adjacency map.
	graph := map[int][]int{1: {2, 3}, 2: {3}}

	// Everything reachable from node 1 by following the graph's edges.
	reachable := set.New(1).Reachable(func(n int) iter.Seq[int] {
		return slices.Values(graph[n])
	})

	// Reachable returns a set, whose iteration order is unspecified, so
	// sort the elements for a deterministic result.
	fmt.Println(slices.Sorted(reachable.Elements()))
	// Output: [1 2 3]
}
