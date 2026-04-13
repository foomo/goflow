package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Concat returns a stream that emits all elements from each input stream
// in order: first all elements from streams[0], then streams[1], etc.
// Uses the context and options from the first stream.
func Concat[T any](streams ...Stream[T]) Stream[T] {
	if len(streams) == 0 {
		return Empty[T]()
	}

	ctx := streams[0].ctx
	opts := streams[0].opts

	if ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T)

	gofuncy.Go(ctx, func(ctx context.Context) error {
		defer close(source)

		for _, s := range streams {
			for item := range s.source {
				select {
				case <-ctx.Done():
					return nil
				case source <- item:
				}
			}
		}

		return nil
	}, append(opts, gofuncy.WithName("goflow.concat"))...)

	return Stream[T]{ctx: ctx, source: source, opts: opts}
}
