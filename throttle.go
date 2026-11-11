package goflow

import (
	"context"
	"time"
)

// Throttle rate-limits the stream to at most one element per duration d.
func (s Stream[T]) Throttle(d time.Duration) Stream[T] {
	return stage(s, "goflow.throttle", func(ctx context.Context, emit func(T) bool) error {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		first := true

		for item := range s.items() {
			if !first {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}

			first = false

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}
