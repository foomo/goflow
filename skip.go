package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Skip drops the first n elements and emits the rest.
func (s Stream[T]) Skip(n int) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, "goflow.skip", func(ctx context.Context) error {
		defer close(source)

		count := 0
		for item := range s.source {
			if count < n {
				count++
				continue
			}

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, s.opts...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
