package goflow

import (
	"context"
	"slices"

	"github.com/foomo/gofuncy"
)

// Reverse collects all elements and emits them in reverse order.
func (s Stream[T]) Reverse() Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		items := s.Collect()
		for _, item := range slices.Backward(items) {
			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.reverse"))...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
