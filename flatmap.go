package goflow

import "context"

// FlatMap applies fn to each element of the source stream, producing a sub-stream
// per element, and flattens the results into a single output stream sequentially.
func FlatMap[T, U any](s Stream[T], fn func(context.Context, T) Stream[U]) Stream[U] {
	return stage(s, "goflow.flat-map", func(ctx context.Context, emit func(U) bool) error {
		for item := range s.items() {
			for v := range fn(ctx, item).items() {
				if !emit(v) {
					return nil
				}
			}
		}

		return nil
	})
}
