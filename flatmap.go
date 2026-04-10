package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// FlatMap applies fn to each element of the source stream, producing a sub-stream
// per element, and flattens the results into a single output stream sequentially.
func FlatMap[T, U any](s Stream[T], fn func(context.Context, T) Stream[U]) Stream[U] {
	if s.ctx.Err() != nil {
		return closed[U](s.ctx)
	}

	source := make(chan U)

	gofuncy.Go(s.ctx, "goflow.flat-map", func(ctx context.Context) error {
		defer close(source)

		for item := range s.source {
			sub := fn(ctx, item)
			for v := range sub.source {
				select {
				case <-ctx.Done():
					return nil
				case source <- v:
				}
			}
		}

		return nil
	}, s.opts...)

	return Stream[U]{ctx: s.ctx, source: source, opts: s.opts}
}
