package goflow

import (
	"context"
	"slices"

	"github.com/foomo/gofuncy"
)

// Sort collects all elements, sorts them using cmp, and emits in sorted order.
func (s Stream[T]) Sort(cmp func(T, T) int) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		items := s.Collect()
		slices.SortFunc(items, cmp)

		for _, item := range items {
			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.sort"))...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
