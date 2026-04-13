package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Generate returns an infinite stream where each element is produced by fn.
// The stream runs until the context is cancelled.
func Generate[T any](ctx context.Context, fn func() T) Stream[T] {
	if ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T)

	gofuncy.Go(ctx, func(ctx context.Context) error {
		defer close(source)

		for {
			select {
			case <-ctx.Done():
				return nil
			case source <- fn():
			}
		}
	}, gofuncy.WithName("goflow.generate"))

	return From[T](ctx, source)
}
