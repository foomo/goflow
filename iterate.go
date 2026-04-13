package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Iterate returns an infinite stream starting with seed, then applying fn
// repeatedly: seed, fn(seed), fn(fn(seed)), ...
// The stream runs until the context is cancelled.
func Iterate[T any](ctx context.Context, seed T, fn func(T) T) Stream[T] {
	if ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T)

	gofuncy.Go(ctx, func(ctx context.Context) error {
		defer close(source)

		val := seed

		for {
			select {
			case <-ctx.Done():
				return nil
			case source <- val:
				val = fn(val)
			}
		}
	}, gofuncy.WithName("goflow.iterate"))

	return From[T](ctx, source)
}
