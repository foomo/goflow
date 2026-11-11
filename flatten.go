package goflow

import "context"

// Flatten flattens a stream of slices into a stream of individual elements.
func Flatten[T any](s Stream[[]T]) Stream[T] {
	return stage(s, "goflow.flatten", func(_ context.Context, emit func(T) bool) error {
		for batch := range s.items() {
			for _, item := range batch {
				if !emit(item) {
					return nil
				}
			}
		}

		return nil
	})
}
