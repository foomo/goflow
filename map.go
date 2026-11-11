package goflow

import "context"

// Map returns a new Stream by applying fn to each element of the source stream.
// If fn returns an error, the stream closes and the error is handled by gofuncy.Go.
func Map[T, U any](s Stream[T], fn func(context.Context, T) (U, error)) Stream[U] {
	return stage(s, "goflow.map", func(ctx context.Context, emit func(U) bool) error {
		for item := range s.items() {
			v, err := fn(ctx, item)
			if err != nil {
				return err
			}

			if !emit(v) {
				return nil
			}
		}

		return nil
	})
}
