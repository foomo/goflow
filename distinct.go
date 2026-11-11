package goflow

import "context"

// Distinct deduplicates elements using a key function. First occurrence wins.
func (s Stream[T]) Distinct(key func(T) string) Stream[T] {
	return stage(s, "goflow.distinct", func(_ context.Context, emit func(T) bool) error {
		seen := make(map[string]struct{})

		for item := range s.items() {
			k := key(item)
			if _, ok := seen[k]; ok {
				continue
			}

			seen[k] = struct{}{}

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
