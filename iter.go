package goflow

import (
	"context"
	"iter"

	"github.com/foomo/gofuncy"
)

// Iter returns an iter.Seq that yields each element of the stream.
// The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter() iter.Seq[T] {
	return func(yield func(T) bool) {
		for {
			select {
			case <-s.ctx.Done():
				return
			case v, ok := <-s.source:
				if !ok {
					return
				}

				if !yield(v) {
					return
				}
			}
		}
	}
}

// Iter2 returns an iter.Seq2 that yields each element with its zero-based index.
// The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter2() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		var i int

		for {
			select {
			case <-s.ctx.Done():
				return
			case v, ok := <-s.source:
				if !ok {
					return
				}

				if !yield(i, v) {
					return
				}

				i++
			}
		}
	}
}

// FromIter creates a Stream from an iter.Seq by pushing elements into a channel.
func FromIter[T any](ctx context.Context, seq iter.Seq[T]) Stream[T] {
	if ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T)

	gofuncy.Go(ctx, "goflow.from-iter", func(ctx context.Context) error {
		defer close(source)

		for v := range seq {
			select {
			case <-ctx.Done():
				return nil
			case source <- v:
			}
		}

		return nil
	})

	return From[T](ctx, source)
}
