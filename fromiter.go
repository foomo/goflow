package goflow

import (
	"context"
	"iter"
)

// FromIter creates a Stream from an [iter.Seq] by pushing elements into a
// channel.
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
