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

	return stage(streams[0], "goflow.fan-in", func(ctx context.Context, emit func(T) bool) error {
		g := gofuncy.NewGroup(ctx, streams[0].p.groupOpts("goflow.fan-in")...)

		for _, s := range streams {
			g.Add(func(context.Context) error {
				for item := range s.items() {
					if !emit(item) {
						return nil
					}
				}

				return nil
			}, gofuncy.WithName("goflow.fan-in.worker"))
		}

		return g.Wait()
	})
}
