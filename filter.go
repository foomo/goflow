package goflow

import "context"

// Filter returns a stream containing only elements where fn returns true.
func (s Stream[T]) Filter(fn func(context.Context, T) bool) Stream[T] {
	return stage(s, "goflow.filter", func(ctx context.Context, emit func(T) bool) error {
		for item := range s.items() {
			if fn(ctx, item) && !emit(item) {
				return nil
			}
		}

		return nil
	})
}
