package goflow

import (
	"context"
	"time"

	"github.com/foomo/gofuncy"
)

// Throttle rate-limits the stream to at most one element per duration d.
func (s Stream[T]) Throttle(d time.Duration) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, func(ctx context.Context) error {
		defer close(source)

		ticker := time.NewTicker(d)
		defer ticker.Stop()

		first := true
		for item := range s.source {
			if !first {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}

			first = false

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, append(s.opts, gofuncy.WithName("goflow.throttle"))...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
