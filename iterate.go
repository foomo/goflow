package goflow

import "context"

// Iterate returns an infinite stream starting with seed, then applying fn
// repeatedly: seed, fn(seed), fn(fn(seed)), ...
// The stream runs until the context is cancelled.
func Iterate[T any](ctx context.Context, seed T, fn func(T) T) Stream[T] {
	return produce(ctx, "goflow.iterate", 0, func(_ context.Context, emit func(T) bool) error {
		for val := seed; emit(val); val = fn(val) {
		}

		return nil
	})
}
