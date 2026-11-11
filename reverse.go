package goflow

import (
	"context"
	"slices"
)

// Reverse collects all elements and emits them in reverse order.
func (s Stream[T]) Reverse() Stream[T] {
	return stage(s, "goflow.reverse", func(_ context.Context, emit func(T) bool) error {
		for _, item := range slices.Backward(s.Collect()) {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
