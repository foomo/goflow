package goflow

import (
	"context"
	"slices"
)

// Sort collects all elements, sorts them using cmp, and emits in sorted order.
func (s Stream[T]) Sort(cmp func(T, T) int) Stream[T] {
	return stage(s, "goflow.sort", func(_ context.Context, emit func(T) bool) error {
		items := s.Collect()
		slices.SortFunc(items, cmp)

		for _, item := range items {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
