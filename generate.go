package goflow

import "context"

// Generate returns an infinite stream where each element is produced by fn.
// The stream runs until the context is cancelled.
func Generate[T any](ctx context.Context, fn func() T) Stream[T] {
	return produce(ctx, "goflow.generate", 0, func(_ context.Context, emit func(T) bool) error {
		for emit(fn()) {
		}

		return nil
	})
}
