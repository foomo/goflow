package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// MapFilter maps and filters elements. The bool controls emission:
// (val, true, nil) emits val; (_, false, nil) skips the item; (_, _, err) stops the stream.
func MapFilter[T, U any](s Stream[T], fn func(context.Context, T) (U, bool, error)) Stream[U] {
	if s.ctx.Err() != nil {
		return closed[U](s.ctx)
	}

	source := make(chan U)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		for item := range s.source {
			v, ok, err := fn(ctx, item)
			if err != nil {
				return err
			}

			if !ok {
				continue
			}

			select {
			case <-ctx.Done():
				return nil
			case source <- v:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.map-filter"))...)

	return Stream[U]{ctx: s.ctx, source: source, opts: s.opts}
}
