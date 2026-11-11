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
// For example, to bridge a callback-based subscriber into a stream:
//
//	s := goflow.FromFunc(ctx, 16, func(ctx context.Context, send func(Event) error) error {
//		return sub.Subscribe(ctx, "events", func(ctx context.Context, e Event) error {
//			return send(e)
//		})
//	})
func FromFunc[T any](ctx context.Context, bufSize int, fn func(ctx context.Context, send func(T) error) error, opts ...gofuncy.GoOption) Stream[T] {
	return stageBuf(Stream[T]{ctx: ctx, opts: opts}, "goflow.from-func", bufSize, func(ctx context.Context, emit func(T) bool) error {
		return fn(ctx, func(v T) error {
			if !emit(v) {
				return ctx.Err()
			}

			return nil
		})
	})
}
