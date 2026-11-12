package goflow

import (
	"context"
	"errors"

	"github.com/foomo/goflow/semconv"
	"github.com/foomo/gofuncy"
)

// goStage runs fn in a gofuncy goroutine that always starts.
//
// gofuncy.Go skips fn when ctx is already cancelled at goroutine start, so a
// stage's deferred close(out) would never run and its reader would hang.
// goStage hands gofuncy a context without cancellation (keeping values and
// the span) and re-attaches ctx's cancellation to fn's context: a deadline
// stays a deadline (Err is DeadlineExceeded) and a cancel keeps its cause.
func goStage(ctx context.Context, fn gofuncy.Func, opts ...gofuncy.GoOption) {
	gofuncy.Go(context.WithoutCancel(ctx), func(gctx context.Context) error {
		gctx, cancel := context.WithCancelCause(gctx)
		defer cancel(nil)

		if d, ok := ctx.Deadline(); ok {
			var cancelDeadline context.CancelFunc

			gctx, cancelDeadline = context.WithDeadline(gctx, d)
			defer cancelDeadline()
		}

		stop := context.AfterFunc(ctx, func() {
			// A deadline is handled by gctx's own deadline above.
			if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				cancel(context.Cause(ctx))
			}
		})
		defer stop()

		return fn(gctx)
	}, opts...)
}

// send queues v on ch and reports false once ctx ends first.
func send[T any](ctx context.Context, ch chan<- T, v T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- v:
		return true
	}
}

// stage runs body as an operator goroutine that emits elements of type U.
// It owns the cancelled-source fast path, the output channel and its close,
// and derives the output stream from s. emit reports false once ctx ends;
// body should then return.
func stage[U, T any](s Stream[T], name string, body func(ctx context.Context, emit func(U) bool) error) Stream[U] {
	return stageBuf(s, name, 0, body)
}

// stageBuf is stage with an output buffer of size buf.
func stageBuf[U, T any](s Stream[T], name string, buf int, body func(ctx context.Context, emit func(U) bool) error) Stream[U] {
	if s.ctx.Err() != nil {
		return derive[U](s, closedChan[U]())
	}

	out := make(chan U, buf)

	goStage(s.ctx, func(ctx context.Context) error {
		defer close(out)

		return body(ctx, func(v U) bool { return send(ctx, out, v) })
	}, s.stageOpts(name)...)

	return derive[U](s, out)
}

// produce is stage for constructors: a root stream on ctx whose elements come
// from body, with an output buffer of size buf.
func produce[T any](ctx context.Context, name string, buf int, body func(ctx context.Context, emit func(T) bool) error) Stream[T] {
	return stageBuf(Stream[struct{}]{ctx: ctx}, name, buf, body)
}

// derive creates a stream for an operator output that keeps ctx, opts and pipeline.
func derive[U, T any](s Stream[T], source <-chan U) Stream[U] {
	return Stream[U]{ctx: s.ctx, source: source, opts: s.opts, p: s.p}
}

// deriveEnv is derive for operators whose output carries trace origins.
func deriveEnv[U, T any](s Stream[T], ch <-chan envelope[U]) Stream[U] {
	return Stream[U]{ctx: s.ctx, opts: s.opts, p: s.p, env: &envSource[U]{ch: ch}}
}

func closedChan[T any]() <-chan T {
	ch := make(chan T)
	close(ch)

	return ch
}

// closed returns an empty, closed stream that preserves the given context.
func closed[T any](ctx context.Context) Stream[T] {
	return Stream[T]{ctx: ctx, source: closedChan[T]()}
}

// consume feeds the stream's elements to fn until the stream closes or ctx
// ends, returning ctx's error in the latter case. Each error from fn goes to
// onErr; when onErr returns an error, consumption stops with it.
func consume[T any](ctx context.Context, s Stream[T], stage semconv.StageNameAttr, fn func(context.Context, T) error, onErr func(context.Context, error) error) error {
	c := newConsumer(s, stage, fn)

	for e := range s.envItems() {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := c.run(ctx, e); err != nil {
			if err := onErr(ctx, err); err != nil {
				return err
			}
		}
	}

	return nil
}
