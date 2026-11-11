package goflow

import "context"

// Skip drops the first n elements and emits the rest.
func (s Stream[T]) Skip(n int) Stream[T] {
	return stage(s, "goflow.skip", func(_ context.Context, emit func(T) bool) error {
		count := 0

		for item := range s.items() {
			if count < n {
				count++

				continue
			}

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
