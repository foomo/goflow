package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Map returns a new Stream by applying fn to each element of the source stream.
// If fn returns an error, the stream closes and the error is handled by gofuncy.Go.
func Map[T, U any](s Stream[T], fn func(context.Context, T) (U, error)) Stream[U] {
	if s.ctx.Err() != nil {
		return closed[U](s.ctx)
	}

	source := make(chan U)

	gofuncy.Go(s.ctx, "goflow.map", func(ctx context.Context) error {
		defer close(source)

		for item := range s.source {
			v, err := fn(ctx, item)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return nil
			case source <- v:
			}
		}

		return nil
	}, s.opts...)

	return Stream[U]{ctx: s.ctx, source: source, opts: s.opts}
}
