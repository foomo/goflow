package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Filter returns a stream containing only elements where fn returns true.
func (s Stream[T]) Filter(fn func(context.Context, T) bool) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		for item := range s.source {
			if !fn(ctx, item) {
				continue
			}

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.filter"))...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
