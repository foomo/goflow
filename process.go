package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Process consumes the stream, dispatching each element to a worker pool of size n.
// All errors are collected and returned via errors.Join.
func (s Stream[T]) Process(n int, fn func(context.Context, T) error, opts ...gofuncy.GroupOption) error {
	g := gofuncy.NewGroup(s.ctx, append([]gofuncy.GroupOption{
		gofuncy.WithName("goflow.process"),
		gofuncy.WithLimit(n),
	}, opts...)...)
	for item := range s.source {
		g.Add(func(ctx context.Context) error {
			return fn(ctx, item)
		}, gofuncy.WithName("goflow.process.worker"))
	}

	return g.Wait()
}
