package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// FanOut distributes elements round-robin across n output streams.
func (s Stream[T]) FanOut(n int) []Stream[T] {
	if n <= 0 || s.ctx.Err() != nil {
		return nil
	}

	sources := make([]chan T, n)
	for i := range sources {
		sources[i] = make(chan T)
	}

	gofuncy.Go(s.ctx, "goflow.fan-out", func(ctx context.Context) error {
		defer func() {
			for _, ch := range sources {
				close(ch)
			}
		}()

		i := 0

		for item := range s.source {
			select {
			case <-ctx.Done():
				return nil
			case sources[i%n] <- item:
			}

			i++
		}

		return nil
	}, s.opts...)

	streams := make([]Stream[T], n)
	for i, ch := range sources {
		streams[i] = Stream[T]{ctx: s.ctx, source: ch, opts: s.opts}
	}

	return streams
}
