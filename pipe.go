package goflow

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/foomo/gofuncy"

	"github.com/foomo/goflow/semconv"
)

var (
	// ErrFull is returned by [Sender.Try] when the pipe buffer is full.
	ErrFull = errors.New("goflow: pipe full")
	// ErrClosed is returned when sending to a closed pipe.
	ErrClosed = errors.New("goflow: pipe closed")
)

// Sender is the writable entry point of a [Pipe] or [PipeFunc]. It is safe for
// concurrent use.
type Sender[T any] struct {
	ch      chan envelope[T]
	closing chan struct{}
	// mu is held shared by senders and exclusively by close, so that the
	// channel is never closed while a send is in progress. Senders do not
	// serialize behind each other.
	mu sync.RWMutex
	// closedErr is returned to senders once the pipe is closed.
	closedErr error
	once      sync.Once
	p         *pipeline

	// The fields below are set before the Sender is returned and never change.

	// onClose runs once, right after the entry point closed.
	onClose func()
	// stopWatch stops closing the entry point when the parent ctx ends.
	stopWatch func() bool
	// wait is set by PipeFunc and waits for its fn after the entry point closed.
	wait func(ctx context.Context) error
}

// newSender creates the entry point.
//
// Queue gauges are only registered for named pipes: two live pipes with the
// same name would overwrite each other's value. They are removed when the
// entry point closes.
func newSender[T any](c config) *Sender[T] {
	s := &Sender[T]{
		ch:      make(chan envelope[T], c.buffer),
		closing: make(chan struct{}),
		p:       newPipeline(c),
		onClose: func() {},
	}

	if c.name != "" {
		s.onClose = s.p.observeQueue(func() int { return len(s.ch) }, func() int { return cap(s.ch) })
	}

	return s
}

// stream returns the readable side of the entry point, bound to ctx.
func (s *Sender[T]) stream(ctx context.Context) Stream[T] {
	return Stream[T]{ctx: ctx, opts: s.p.stageOpts(), p: s.p, env: &envSource[T]{ch: s.ch}}
}

// watch closes the entry point when ctx ends. Call it last, once all fields are set.
func (s *Sender[T]) watch(ctx context.Context) {
	// Closing because ctx ended reports both ErrClosed and the ctx cause.
	closeByCtx := func() { s.close(fmt.Errorf("%w: %w", ErrClosed, context.Cause(ctx))) }
	if ctx.Err() != nil {
		closeByCtx()

		return
	}

	s.stopWatch = context.AfterFunc(ctx, closeByCtx)
}

// Pipe creates a writable stream entry point.
// Returns a Sender and the readable stream.
//
// The entry point closes when ctx is done or when [Sender.Close] is called. The
// stream and all stages derived from it run on ctx, so they stop when ctx ends.
// After that, [Stream.Collect], [Stream.Count], [Stream.Iter], [Stream.Iter2]
// and [Stream.FindFirst] still read the elements buffered in the entry point;
// [Stream.ForEach], [Stream.Process], [Stream.Chan] and the other stages stop
// with ctx and drop them. To drain a long-lived pipeline on shutdown, use
// [PipeFunc] instead.
//
// Queue depth and capacity gauges are only reported when [WithName] is set;
// the name must then be unique among live pipes. They are removed when the
// entry point closes.
//
// Options: [WithBuffer], WithName, [WithMeterProvider], [WithTracerProvider],
// [WithStageTracing].
func Pipe[T any](ctx context.Context, opts ...Option) (*Sender[T], Stream[T]) {
	s := newSender[T](newConfig(opts))
	s.watch(ctx)

	return s, s.stream(ctx)
}

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

// closeAfterConsumer closes the entry point once PipeFunc's fn returned:
// nobody reads it any more, so further sends must fail instead of filling the
// buffer and blocking. Elements left in the buffer are counted as dropped.
func (s *Sender[T]) closeAfterConsumer(ctx context.Context) {
	if s.stopWatch != nil {
		s.stopWatch()
	}

	s.close(fmt.Errorf("%w: consumer returned", ErrClosed))

	var n int64

	for range s.ch {
		n++
	}

	if n > 0 {
		s.p.dropped.AddSet(ctx, n, s.p.dropSets[semconv.DropReasonClosed])
	}
}

// rejectIfClosed reports the closed error once the entry point is closing.
// Closing wins over an enqueue that would also be ready.
func (s *Sender[T]) rejectIfClosed(ctx context.Context) error {
	select {
	case <-s.closing:
		s.p.dropped.AddSet(ctx, 1, s.p.dropSets[semconv.DropReasonClosed])

		return s.closedErr
	default:
		return nil
	}
}

// Send blocks until v is queued, ctx is done or the pipe closes.
func (s *Sender[T]) Send(ctx context.Context, v T) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.rejectIfClosed(ctx); err != nil {
		return err
	}

	select {
	case <-s.closing:
		s.p.dropped.AddSet(ctx, 1, s.p.dropSets[semconv.DropReasonClosed])

		return s.closedErr
	case <-ctx.Done():
		s.p.dropped.AddSet(ctx, 1, s.p.dropSets[semconv.DropReasonCanceled])

		return ctx.Err()
	case s.ch <- newEnvelope(ctx, v):
		s.p.enqueued.AddSet(ctx, 1, s.p.nameSet)

		return nil
	}
}

// Try queues v without blocking. It returns [ErrFull] when the buffer is full
// and [ErrClosed] when the pipe is closed. Rejected elements count as dropped.
func (s *Sender[T]) Try(ctx context.Context, v T) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.rejectIfClosed(ctx); err != nil {
		return err
	}

	select {
	case s.ch <- newEnvelope(ctx, v):
		s.p.enqueued.AddSet(ctx, 1, s.p.nameSet)

		return nil
	default:
		s.p.dropped.AddSet(ctx, 1, s.p.dropSets[semconv.DropReasonFull])

		return ErrFull
	}
}

// Close closes the entry point and implements a graceful closer (e.g. keel's
// Close(ctx) error).
//
// For a [Pipe] sender it returns nil at once. For a [PipeFunc] sender it then
// waits until fn has returned and returns fn's error; if ctx ends first, it
// cancels fn's context and returns ctx.Err().
func (s *Sender[T]) Close(ctx context.Context) error {
	if s.stopWatch != nil {
		s.stopWatch()
	}

	s.close(ErrClosed)

	if s.wait == nil {
		return nil
	}

	return s.wait(ctx)
}

// close closes the entry point once; err is what later sends return.
func (s *Sender[T]) close(err error) {
	s.once.Do(func() {
		s.closedErr = err
		close(s.closing)
		s.mu.Lock()
		close(s.ch)
		s.mu.Unlock()
		s.onClose()
	})
}
