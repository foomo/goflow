package goflow

import (
	"context"
)

// Of returns a Stream based on the given elements.
func Of[T any](ctx context.Context, items ...T) Stream[T] {
	if len(items) == 0 {
		return closed[T](ctx)
	}

	return produce(ctx, "goflow.of", len(items), func(_ context.Context, emit func(T) bool) error {
		for _, item := range items {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
