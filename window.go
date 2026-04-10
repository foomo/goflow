package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Window emits sliding windows of n consecutive elements.
// If the source has fewer than n elements, no windows are emitted.
func Window[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 || s.ctx.Err() != nil {
		return closed[[]T](s.ctx)
	}

	source := make(chan []T)

	gofuncy.Go(s.ctx, "goflow.window", func(ctx context.Context) error {
		defer close(source)

		buf := make([]T, 0, n)
		for item := range s.source {
			if len(buf) == n {
				buf = buf[1:]
			}

			buf = append(buf, item)
			if len(buf) == n {
				win := make([]T, n)
				copy(win, buf)

				select {
				case <-ctx.Done():
					return nil
				case source <- win:
				}
			}
		}

		return nil
	}, s.opts...)

	return Stream[[]T]{ctx: s.ctx, source: source, opts: s.opts}
}
