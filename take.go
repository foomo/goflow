package goflow

import "context"

// Take emits the first n elements then closes the stream.
func (s Stream[T]) Take(n int) Stream[T] {
	if n <= 0 {
		return closed[T](s.ctx)
	}

	return stage(s, "goflow.take", func(_ context.Context, emit func(T) bool) error {
		count := 0

		for item := range s.items() {
			if count >= n || !emit(item) {
				return nil
			}

			count++
		}

		return nil
	})
}
