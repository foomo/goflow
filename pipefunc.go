package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// PipeFunc creates a draining pipeline: an entry point plus the consumer fn,
// which runs in its own goroutine and owns the pipeline. Returns the Sender.
//
// fn and all stages it builds run on a context that ctx does not cancel. When
// ctx ends or Close is called, the entry point closes; fn keeps consuming the
// elements already queued and must return once its stream ends. Close(ctx)
// waits for that. When fn returns, or Close's ctx ends first, the stage
// context is cancelled.
//
// An fn that only waits for ctx.Done() keeps running until Close times out.
// When fn returns, the entry point closes too: later sends return [ErrClosed],
// and elements still buffered are dropped (counted in goflow.pipe.dropped).
//
// Options: as for [Pipe].
func PipeFunc[T any](ctx context.Context, fn func(context.Context, Stream[T]) error, opts ...Option) *Sender[T] {
	s := newSender[T](newConfig(opts))

	sctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stream := s.stream(sctx)

	var fnErr error

	done := make(chan struct{})

	// Watch before starting fn, so that fn's goroutine may read stopWatch.
	s.watch(ctx)

	goStage(sctx, func(ctx context.Context) error {
		defer close(done)
		defer cancel()
		defer s.closeAfterConsumer(ctx)

		fnErr = fn(ctx, stream)

		return fnErr
	}, append(stream.opts, gofuncy.WithName("goflow.pipe-func"))...)

	s.wait = func(ctx context.Context) error {
		// Prefer done over an also-expired ctx.
		select {
		case <-done:
			return fnErr
		default:
		}

		select {
		case <-done:
			return fnErr
		case <-ctx.Done():
			cancel()

			return ctx.Err()
		}
	}

	return s
}
