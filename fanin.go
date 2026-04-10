package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// FanIn combines multiple streams into a single stream.
// Elements arrive in non-deterministic order as they become available.
// Uses the context and options from the first stream.
func FanIn[T any](streams []Stream[T]) Stream[T] {
	if len(streams) == 0 {
		return Empty[T]()
	}

	ctx := streams[0].ctx
	opts := streams[0].opts

	if ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T)
	g := gofuncy.NewGroup(ctx, "goflow.fan-in")

	for _, s := range streams {
		g.Add("goflow.fan-in.worker", func(ctx context.Context) error {
			for item := range s.source {
				select {
				case <-ctx.Done():
					return nil
				case source <- item:
				}
			}

			return nil
		})
	}

	gofuncy.Go(ctx, "goflow.fan-in", func(ctx context.Context) error {
		defer close(source)
		return g.Wait()
	}, opts...)

	return Stream[T]{ctx: ctx, source: source, opts: opts}
}
