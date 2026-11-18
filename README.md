# go-util

[![Go Reference](https://pkg.go.dev/badge/github.com/dpw/go-util.svg)](https://pkg.go.dev/github.com/dpw/go-util)
[![Tests](https://github.com/dpw/go-util/actions/workflows/test.yml/badge.svg)](https://github.com/dpw/go-util/actions/workflows/test.yml)

A collection of small, general-purpose Go packages, filling gaps
around the standard library.  Each package is independent and focused
on a single concern.

## Installation

```sh
go get github.com/dpw/go-util
```

Requires Go 1.27 or later.

## Packages

| Package | Description |
| ------- | ----------- |
| [`cmpx`](https://pkg.go.dev/github.com/dpw/go-util/cmpx) | Helpers for comparison functions, complementing `cmp`. |
| [`errorsx`](https://pkg.go.dev/github.com/dpw/go-util/errorsx) | Error wrapping with stack traces and rich formatting. |
| [`iterx`](https://pkg.go.dev/github.com/dpw/go-util/iterx) | Combinators for `iter.Seq` and `iter.Seq2` iterators. |
| [`set`](https://pkg.go.dev/github.com/dpw/go-util/set) | A generic set type backed by a `map[T]struct{}`. |

See each package's documentation for details and examples.

## License

BSD 3-Clause. See [LICENSE](LICENSE).
