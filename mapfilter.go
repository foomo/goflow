package goflow

import "context"

// MapFilter maps and filters elements. The bool controls emission:
// (val, true, nil) emits val; (_, false, nil) skips the item; (_, _, err) stops the stream.
func MapFilter[T, U any](s Stream[T], fn func(context.Context, T) (U, bool, error)) Stream[U] {
	return stage(s, "goflow.map-filter", func(ctx context.Context, emit func(U) bool) error {
		for item := range s.items() {
			v, ok, err := fn(ctx, item)
			if err != nil {
				return err
			}

			if ok && !emit(v) {
				return nil
			}
		}

		return nil
	})
}
