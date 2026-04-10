package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Peek calls fn for each element as a side-effect and forwards the element unchanged.
func (s Stream[T]) Peek(fn func(context.Context, T)) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, "goflow.peek", func(ctx context.Context) error {
		defer close(source)

		for item := range s.source {
			fn(ctx, item)

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, s.opts...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
