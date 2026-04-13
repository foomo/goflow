package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// FromFunc creates a Stream from a blocking function that sends items via the
// provided send callback. The function should block until it is done producing
// items and must respect context cancellation.
//
// The stream closes automatically when fn returns. If fn returns a non-nil
// error it is handled by gofuncy.Go (logged via the configured error handler).
//
// bufSize controls backpressure: a full buffer blocks the send callback until
// the stream consumer catches up.
//
// Example — bridging a message subscriber into a stream:
//
//	s := stream.FromFunc(ctx, 16, func(ctx context.Context, send func(Event) error) error {
//	    return sub.Subscribe(ctx, "events", func(ctx context.Context, msg courier.Message[Event]) error {
//	        return send(msg.Payload)
//	    })
//	})
func FromFunc[T any](ctx context.Context, bufSize int, fn func(ctx context.Context, send func(T) error) error, opts ...gofuncy.GoOption) Stream[T] {
	ch := make(chan T, bufSize)
	send := func(v T) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ch <- v:
			return nil
		}
	}

	gofuncy.Go(ctx, func(ctx context.Context) error {
		defer close(ch)
		return fn(ctx, send)
	}, append(opts, gofuncy.WithName("goflow.from-func"))...)

	return Stream[T]{ctx: ctx, source: ch, opts: opts}
}
