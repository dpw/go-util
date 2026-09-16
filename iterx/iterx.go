package iterx

import (
	"iter"
)

// Empty returns a sequence that yields no values.  It is useful where
// an [iter.Seq] is required but there is nothing to iterate over.
func Empty[T any]() iter.Seq[T] {
	return func(func(T) bool) {}
}

// Transform returns a sequence that yields f applied to each value of
// s. If consumption of the output sequence stops, consumption of the
// input sequence stops.
func Transform[In, Out any](s iter.Seq[In], f func(In) Out) iter.Seq[Out] {
	return func(yield func(Out) bool) {
		for v := range s {
			if !yield(f(v)) {
				return
			}
		}
	}
}

// Filter returns a sequence that yields the values of s for which f
// returns true, preserving their order.  If consumption of the output
// sequence stops, consumption of the input sequence stops.
func Filter[T any](s iter.Seq[T], f func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s {
			if f(v) && !yield(v) {
				return
			}
		}
	}
}

// Flatten concatenates sequences generates by the function f, passing
// it each value of the input sequence in turn.  If consumption of the
// output sequence stops, consumption of the input sequence stops.
func Flatten[In, Out any](s iter.Seq[In], f func(In) iter.Seq[Out]) iter.Seq[Out] {
	return func(yield func(Out) bool) {
		for v := range s {
			for w := range f(v) {
				if !yield(w) {
					return
				}
			}
		}
	}
}

// Transform2 returns a sequence that yields f applied to each
// key-value pair of s, mapping each pair to a new pair.  If
// consumption of the output sequence stops, consumption of the input
// sequence stops.
func Transform2[InK, InV, OutK, OutV any](s iter.Seq2[InK, InV], f func(InK, InV) (OutK, OutV)) iter.Seq2[OutK, OutV] {
	return func(yield func(OutK, OutV) bool) {
		for k, v := range s {
			if !yield(f(k, v)) {
				return
			}
		}
	}
}

// Filter2 returns a sequence that yields the key-value pairs of s for
// which f returns true, preserving their order.  If consumption of the
// output sequence stops, consumption of the input sequence stops.
func Filter2[K, V any](s iter.Seq2[K, V], f func(K, V) bool) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range s {
			if f(k, v) && !yield(k, v) {
				return
			}
		}
	}
}

// Transform2To1 returns a sequence that yields f applied to each
// key-value pair of s, collapsing an [iter.Seq2] into an [iter.Seq].
// If consumption of the output sequence stops, consumption of the
// input sequence stops.
func Transform2To1[InK, InV, Out any](s iter.Seq2[InK, InV], f func(InK, InV) Out) iter.Seq[Out] {
	return func(yield func(Out) bool) {
		for k, v := range s {
			if !yield(f(k, v)) {
				return
			}
		}
	}
}

// Enumerate returns a sequence that pairs each value of s with its
// index, starting from 0.  If consumption of the output sequence
// stops, consumption of the input sequence stops.
func Enumerate[T any](s iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		i := 0
		for v := range s {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}
