package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Split groups consecutive elements into batches of size n.
// The last batch may contain fewer than n elements.
func Split[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 || s.ctx.Err() != nil {
		return closed[[]T](s.ctx)
	}

	source := make(chan []T)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		batch := make([]T, 0, n)
		for item := range s.source {
			batch = append(batch, item)
			if len(batch) == n {
				select {
				case <-ctx.Done():
					return nil
				case source <- batch:
				}

				batch = make([]T, 0, n)
			}
		}

		if len(batch) > 0 {
			select {
			case <-ctx.Done():
				return nil
			case source <- batch:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.split"))...)

	return Stream[[]T]{ctx: s.ctx, source: source, opts: s.opts}
}
