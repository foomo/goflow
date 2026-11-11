package goflow

import (
	"context"
	"iter"
)

// Iter returns an iter.Seq that yields each element of the stream.
// The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter() iter.Seq[T] {
	return s.all()
}

// Iter2 returns an iter.Seq2 that yields each element with its zero-based index.
// The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter2() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		var i int

		for v := range s.all() {
			if !yield(i, v) {
				return
			}

			i++
		}
	}
}

// FromIter creates a Stream from an iter.Seq by pushing elements into a channel.
func FromIter[T any](ctx context.Context, seq iter.Seq[T]) Stream[T] {
	return produce(ctx, "goflow.from-iter", 0, func(_ context.Context, emit func(T) bool) error {
		for v := range seq {
			if !emit(v) {
				return nil
			}
		}

		return nil
	})
}
