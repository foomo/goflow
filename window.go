package goflow

import "context"

// Window emits sliding windows of n consecutive elements.
// If the source has fewer than n elements, no windows are emitted.
func Window[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		return closed[[]T](s.ctx)
	}

	return stage(s, "goflow.window", func(_ context.Context, emit func([]T) bool) error {
		buf := make([]T, 0, n)

		for item := range s.items() {
			if len(buf) == n {
				buf = buf[1:]
			}

			buf = append(buf, item)
			if len(buf) < n {
				continue
			}

			win := make([]T, n)
			copy(win, buf)

			if !emit(win) {
				return nil
			}
		}

		return nil
	})
}
