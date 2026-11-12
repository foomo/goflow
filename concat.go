package goflow

import "context"

// Concat returns a stream that emits all elements from each input stream
// in order: first all elements from streams[0], then streams[1], etc.
// Uses the context and options from the first stream.
func Concat[T any](streams ...Stream[T]) Stream[T] {
	if len(streams) == 0 {
		return Empty[T]()
	}

	return stage(streams[0], "goflow.concat", func(_ context.Context, emit func(T) bool) error {
		for _, s := range streams {
			for item := range s.items() {
				if !emit(item) {
					return nil
				}
			}
		}

		return nil
	})
}
