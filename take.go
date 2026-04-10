package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Take emits the first n elements then closes the stream.
func (s Stream[T]) Take(n int) Stream[T] {
	if n <= 0 || s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, "goflow.take", func(ctx context.Context) error {
		defer close(source)

		count := 0
		for item := range s.source {
			if count >= n {
				return nil
			}

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
				count++
			}
		}

		return nil
	}, s.opts...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
