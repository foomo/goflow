package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Flatten flattens a stream of slices into a stream of individual elements.
func Flatten[T any](s Stream[[]T]) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, "goflow.flatten", func(ctx context.Context) error {
		defer close(source)

		for batch := range s.source {
			for _, item := range batch {
				select {
				case <-ctx.Done():
					return nil
				case source <- item:
				}
			}
		}

		return nil
	}, s.opts...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
