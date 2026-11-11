package goflow

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
)

// Stream is a sequence of elements of type T that flows through a channel and
// is bound to a context.
//
// Create a Stream with a constructor such as [Of], [From], [FromFunc] or
// [Pipe]. Operators that keep the element type are methods; operators that
// change it are functions such as [Map]. Each operator starts a goroutine and
// returns a new Stream; the stages stop when the context ends.
//
// A Stream is single-use: read it with exactly one operator or terminal
// method (Collect, Count, Chan, Iter, ForEach, ...). The zero value is not
// usable.
type Stream[T any] struct {
	ctx    context.Context //nolint:containedctx // by design
	source <-chan T
	opts   []gofuncy.GoOption
	// p is the pipeline handle shared by all streams derived from a Pipe (nil otherwise).
	p *pipeline
	// env is set instead of source by Pipe and Batch so that link-aware consumers
	// (Batch, ForEach, Process) can read trace links alongside each element.
	env *envSource[T]
}

// envelope carries an element together with its trace origin.
type envelope[T any] struct {
	v T
	// sc is the span context captured by Pipe at send time.
	sc trace.SpanContext
	// batch is set by Batch for each flushed batch.
	batch *batchMeta
}

// batchMeta describes a flushed batch.
type batchMeta struct {
	// links are the aggregated origins of the batch, capped at maxLinks.
	links        []trace.Link
	droppedLinks int
	reason       semconv.FlushReasonAttr
	count        int
}

func newEnvelope[T any](ctx context.Context, v T) envelope[T] {
	return envelope[T]{v: v, sc: trace.SpanContextFromContext(ctx)}
}

func (e envelope[T]) spanLinks() []trace.Link {
	if e.batch != nil {
		return e.batch.links
	}

	if e.sc.IsValid() {
		return []trace.Link{{SpanContext: e.sc}}
	}

	return nil
}

type envSource[T any] struct {
	ch    <-chan envelope[T]
	once  sync.Once
	plain <-chan T
}

// ch returns the plain element channel, lazily unwrapping envelopes if needed.
func (s Stream[T]) ch() <-chan T {
	if s.env == nil {
		return s.source
	}

	s.env.once.Do(func() {
		out := make(chan T, cap(s.env.ch))
		s.env.plain = out

		goStage(s.ctx, func(ctx context.Context) error {
			defer close(out)

			for e := range s.env.ch {
				if !send(ctx, out, e.v) {
					return nil
				}
			}

			return nil
		}, s.stageOpts("goflow.unwrap")...)
	})

	return s.env.plain
}

// items yields the elements in order until the source closes. Unlike ch it
// starts no goroutine and does not watch the context, so elements already
// buffered in a closed Pipe are always delivered.
func (s Stream[T]) items() iter.Seq[T] {
	return func(yield func(T) bool) {
		if s.env == nil {
			for v := range s.source {
				if !yield(v) {
					return
				}
			}

			return
		}

		for e := range s.env.ch {
			if !yield(e.v) {
				return
			}
		}
	}
}

// envItems is items for link-aware consumers: plain elements arrive in an
// envelope without origin.
func (s Stream[T]) envItems() iter.Seq[envelope[T]] {
	return func(yield func(envelope[T]) bool) {
		if s.env == nil {
			for v := range s.source {
				if !yield(envelope[T]{v: v}) {
					return
				}
			}

			return
		}

		for e := range s.env.ch {
			if !yield(e) {
				return
			}
		}
	}
}

// envelopes returns exactly one non-nil channel: the envelope channel when the
// stream carries trace origins, the plain channel otherwise.
func (s Stream[T]) envelopes() (<-chan envelope[T], <-chan T) {
	if s.env != nil {
		return s.env.ch, nil
	}

	return nil, s.source
}

// stageOpts returns the stream's gofuncy options plus the stage name, without
// writing into the spare capacity of s.opts.
func (s Stream[T]) stageOpts(name string) []gofuncy.GoOption {
	return append(slices.Clip(s.opts), gofuncy.WithName(name))
}

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

// Context returns the stream's bound context.
func (s Stream[T]) Context() context.Context {
	return s.ctx
}

// WithOptions returns a shallow copy of the stream with the given options appended.
func (s Stream[T]) WithOptions(opts ...gofuncy.GoOption) Stream[T] {
	s.opts = slices.Concat(s.opts, opts)

	return s
}

// Chan returns the channel that carries the stream's elements. The channel
// closes when the source ends. For streams from [Pipe] or [Batch], a stage
// goroutine feeds the channel and stops when the stream's context ends.
func (s Stream[T]) Chan() <-chan T {
	return s.ch()
}

// Count drains the stream and returns the number of elements read. It stops
// when the source closes or the stream's context ends. See [Pipe] for
// elements still buffered when the context ends.
func (s Stream[T]) Count() int {
	var count int

	for range s.all() {
		count++
	}

	return count
}

// Collect drains the stream and returns its elements in order. It stops when
// the source closes or the stream's context ends, and returns the elements
// read so far. See [Pipe] for elements still buffered when the context ends.
// It returns nil for an empty stream.
func (s Stream[T]) Collect() []T {
	var out []T

	for v := range s.all() {
		out = append(out, v)
	}

	return out
}

// all yields the elements for the draining terminals (Collect, Count, Iter,
// Iter2). Envelope sources (Pipe, Batch) always close, so they are read to
// the end. Plain sources are read until they close or ctx ends, preferring
// elements that are already available over cancellation.
func (s Stream[T]) all() iter.Seq[T] {
	if s.env != nil {
		return s.items()
	}

	return func(yield func(T) bool) {
		for {
			var (
				v  T
				ok bool
			)

			select {
			case v, ok = <-s.source:
			default:
				select {
				case <-s.ctx.Done():
					return
				case v, ok = <-s.source:
				}
			}

			if !ok || !yield(v) {
				return
			}
		}
	}
}

func closedChan[T any]() <-chan T {
	ch := make(chan T)
	close(ch)

	return ch
}

// Empty returns a closed stream with no elements, bound to
// [context.Background].
func Empty[T any]() Stream[T] {
	return Stream[T]{ctx: context.Background(), source: closedChan[T]()}
}

// closed returns an empty, closed stream that preserves the given context.
func closed[T any](ctx context.Context) Stream[T] {
	return Stream[T]{ctx: ctx, source: closedChan[T]()}
}

// Of returns a Stream based on the given elements.
func Of[T any](ctx context.Context, items ...T) Stream[T] {
	if len(items) == 0 {
		return closed[T](ctx)
	}

	return produce(ctx, "goflow.of", len(items), func(_ context.Context, emit func(T) bool) error {
		for _, item := range items {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// From wraps an existing channel as a Stream with the given context.
func From[T any](ctx context.Context, source <-chan T) Stream[T] {
	return Stream[T]{ctx: ctx, source: source}
}
