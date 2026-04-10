package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Tee broadcasts every element to n output streams.
// Unlike FanOut which round-robins, Tee sends each element to all streams.
func (s Stream[T]) Tee(n int) []Stream[T] {
	if n <= 0 || s.ctx.Err() != nil {
		return nil
	}

	sources := make([]chan T, n)
	for i := range sources {
		sources[i] = make(chan T)
	}

	gofuncy.Go(s.ctx, "goflow.tee", func(ctx context.Context) error {
		defer func() {
			for _, ch := range sources {
				close(ch)
			}
		}()

		for item := range s.source {
			for _, ch := range sources {
				select {
				case <-ctx.Done():
					return nil
				case ch <- item:
				}
			}
		}

		return nil
	}, s.opts...)

	streams := make([]Stream[T], n)
	for i, ch := range sources {
		streams[i] = Stream[T]{ctx: s.ctx, source: ch, opts: s.opts}
	}

	return streams
}
