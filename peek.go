package goflow

import "context"

// Peek calls fn for each element as a side-effect and forwards the element unchanged.
func (s Stream[T]) Peek(fn func(context.Context, T)) Stream[T] {
	return stage(s, "goflow.peek", func(ctx context.Context, emit func(T) bool) error {
		for item := range s.items() {
			fn(ctx, item)

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
