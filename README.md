# go-util

[![Go Reference](https://pkg.go.dev/badge/github.com/dpw/go-util.svg)](https://pkg.go.dev/github.com/dpw/go-util)
[![Tests](https://github.com/dpw/go-util/actions/workflows/test.yml/badge.svg)](https://github.com/dpw/go-util/actions/workflows/test.yml)

A collection of simple Go utility packages, providing generic helpers
for working with iterators, sets, and comparisons.

## Installation

```sh
go get github.com/dpw/go-util
```

Requires Go 1.27 or later (the packages build on the standard `iter`,
`maps`, and `slices` support for generics and range-over-func).

## Packages

### `cmpx`

Helpers for comparison functions.

- `By(f)` adapts a key-extraction function into a comparison function
  suitable for APIs such as `slices.SortFunc` and `slices.SortedFunc`.

```go
people := []Person{{"Bob", 30}, {"Alice", 25}}
slices.SortFunc(people, cmpx.By(func(p Person) int { return p.Age }))
```

### `iterx`

Combinators for `iter.Seq` and `iter.Seq2` iterators.

| Function        | Description                                                |
| --------------- | ---------------------------------------------------------- |
| `Empty`         | A sequence that yields no values.                          |
| `Transform`     | Apply a function to each value of a sequence.              |
| `Filter`        | Keep only the values for which a predicate returns true.   |
| `Flatten`       | Map each value to a sequence and concatenate the results.  |
| `Transform2`    | Apply a function to each pair, producing a new pair.       |
| `Filter2`       | Keep only the pairs for which a predicate returns true.    |
| `Transform2To1` | Map each pair to a single value (`Seq2` to `Seq`).         |
| `Enumerate`     | Pair each value with its index, producing a `Seq2[int, T]`.|

```go
nums := slices.Values([]int{1, 2, 3, 4})
evens := iterx.Filter(nums, func(n int) bool { return n%2 == 0 })
for n := range evens {
    fmt.Println(n) // 2, 4
}
```

### `set`

A generic set type, `Set[T]`, backed by a `map[T]struct{}`.  This has
the same characteristics as direct use of a `map[T]struct{}` to
represent a set.  But `Set[T]` provides convenience methods to allow
clearer, more concise code:

- Construction: `New`, `FromSlice`, `Collect` (all return nil for empty
  inputs)
- Query: `Contains`, `ContainsAll`, `ContainsAny`, `Equal`,
  `IsSubsetOf`, `Elements` (an iterator over the members), `Clone`
- Mutation: `Add`, `AddAll`, `Remove`, `RemoveAll`; and `AddNE`,
  `AddAllNE`, which allocate the set if it is nil
- Combination (each returns a new set, leaving its operands unchanged):
  `Intersect`, `Subtract`, `Union`
- Utilities: `NilToEmpty`, `Reachable` (reachability under a relation)

The methods with the suffix `All` accept an `iter.Seq`, for
convenience when mixing sets with other collection types.

```go
s := set.New(1, 2, 3)
s.Add(4)
fmt.Println(s.Contains(2))          // true
fmt.Println(s.Union(set.New(4, 5))) // {1, 2, 3, 4, 5}
```

A nil `Set[T]` is a valid, empty, read-only set, just like any nil map.
`NilToEmpty` and `AddNE` are convenience methods to simplify code
modifying nil sets.

Because `Set[T]` is simply a map, you can treat it as such when
convenient.  For example,

- `len(s)` to get the size of a set
- `make(Set[string])` to allocate a new set of strings, and
  `make(Set[string], n)` with space pre-allocated for `n` elements
- `for x := range s { ... }` to iterate over the elements of a set

#### How this generic set package compares to alternatives

Some other open source generic set types for Go are available.  None
fill exactly the same niche as this one.

- The original Go generics proposal included [a sketch of a similar
  set type as an
  example](https://go.googlesource.com/proposal/+/refs/heads/master/design/43651-type-parameters.md#sets).
  This package could be seen as a fully elaborated descendant of that
  example.
- [deckarep/golang-set](https://github.com/deckarep/golang-set)
  defines its `Set[T]` as an interface, and provides thread-safe and
  non-thread-safe implementations.  That's overkill for most contexts
  where a concrete non-thread-safe variant suffices.
- [hashicorp/go-set](https://github.com/hashicorp/go-set) provides a
  suite of set implementations, including one backed by a
  `map[T]struct{}`.  These implementations implement a common
  `Collection[T]` interface.  Again, this is overkill for most
  contexts.  Furthermore, the [addition of iterators in Go
  1.23](https://go.dev/blog/range-functions) supersedes the
  `Collection[T]` interface in many ways.
- More generally, this set package is designed to fit well into
  recent Go releases.  It omits features that are made redundant by
  the iterators of Go 1.23.  And it follows the same conventions as the
  `slices` and `maps` packages (e.g. `Collect` returns `nil` rather
  than an empty instance).

## License

BSD 3-Clause. See [LICENSE](LICENSE).
