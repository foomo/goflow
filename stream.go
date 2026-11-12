package goflow

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"
	"time"

	"github.com/foomo/goflow/semconv"
	"github.com/foomo/gofuncy"
)

// Stream is a sequence of elements of type T that flows through a channel and
// is bound to a context.
//
// Create a Stream with a constructor such as [Of], [From], [FromFunc] or
// [Pipe]. Operators that keep the element type are methods; operators that
// change it are functions such as [Map]. Each operator starts a goroutine and
// returns a new Stream; the stages stop when the context ends.
//
// A Stream is single-use: read it with exactly one operator or terminal method
// ([Stream.Collect], [Stream.Count], [Stream.Chan], [Stream.Iter],
// [Stream.ForEach], ...). The zero value is not usable.
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

// AllMatch reports whether all elements match fn. It stops at the first
// element that does not match. It reports true for an empty stream and false
// when the stream's context ends first.
func (s Stream[T]) AllMatch(fn func(context.Context, T) bool) bool {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			return false
		}

		if !fn(s.ctx, item) {
			return false
		}
	}

	return true
}

// AnyMatch reports whether any element matches fn. It stops at the first
// element that matches. It reports false for an empty stream and when the
// stream's context ends first.
func (s Stream[T]) AnyMatch(fn func(context.Context, T) bool) bool {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			return false
		}

		if fn(s.ctx, item) {
			return true
		}
	}

	return false
}

// Chan returns the channel that carries the stream's elements. The channel
// closes when the source ends. For streams from [Pipe] or [Batch], a stage
// goroutine feeds the channel and stops when the stream's context ends.
func (s Stream[T]) Chan() <-chan T {
	return s.ch()
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

// Context returns the stream's bound context.
func (s Stream[T]) Context() context.Context {
	return s.ctx
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

// Distinct deduplicates elements using a key function. First occurrence wins.
func (s Stream[T]) Distinct(key func(T) string) Stream[T] {
	return stage(s, "goflow.distinct", func(_ context.Context, emit func(T) bool) error {
		seen := make(map[string]struct{})

		for item := range s.items() {
			k := key(item)
			if _, ok := seen[k]; ok {
				continue
			}

			seen[k] = struct{}{}

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// FanOut distributes elements round-robin across n output streams.
func (s Stream[T]) FanOut(n int) []Stream[T] {
	return s.spread(n, "goflow.fan-out", func(i int, _ []chan T) int { return i % n })
}

// Filter returns a stream containing only elements where fn returns true.
func (s Stream[T]) Filter(fn func(context.Context, T) bool) Stream[T] {
	return stage(s, "goflow.filter", func(ctx context.Context, emit func(T) bool) error {
		for item := range s.items() {
			if fn(ctx, item) && !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// FindFirst returns the first element of the stream and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) FindFirst() (T, bool) {
	for item := range s.items() {
		return item, true
	}

	var zero T

	return zero, false
}

// FindFirstMatch returns the first element matching the predicate and true,
// or the zero value and false if no element matches.
func (s Stream[T]) FindFirstMatch(fn func(context.Context, T) bool) (T, bool) {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			break
		}

		if fn(s.ctx, item) {
			return item, true
		}
	}

	var zero T

	return zero, false
}

// ForEach consumes the stream, calling fn for each element.
// Returns the first error from fn or ctx, nil when fully consumed.
//
// With [WithErrorHandler], errors from fn are passed to the handler and
// consumption continues; ForEach then only returns ctx errors or nil.
func (s Stream[T]) ForEach(fn func(context.Context, T) error, opts ...Option) error {
	c := newConfig(opts)

	return consume(s.ctx, s, semconv.StageNameForEach, fn, func(ctx context.Context, err error) error {
		if c.errorHandler == nil {
			return err
		}

		c.errorHandler(ctx, err)

		return nil
	})
}

// Iter returns an [iter.Seq] that yields each element of the stream.
// The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter() iter.Seq[T] {
	return s.all()
}

// Iter2 returns an [iter.Seq2] that yields each element with its zero-based
// index. The returned iterator drains the stream; it can only be used once.
func (s Stream[T]) Iter2() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		var i int

		for v := range s.all() {
			if !yield(i, v) {
				return
			}

			i++
		}
	}
}

// Max returns the maximum element according to cmp and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) Max(cmp func(T, T) int) (T, bool) {
	var best T

	found := false

	for item := range s.items() {
		if s.ctx.Err() != nil {
			break
		}

		if !found || cmp(item, best) > 0 {
			best = item
			found = true
		}
	}

	return best, found
}

// Min returns the minimum element according to cmp and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) Min(cmp func(T, T) int) (T, bool) {
	var best T

	found := false

	for item := range s.items() {
		if s.ctx.Err() != nil {
			break
		}

		if !found || cmp(item, best) < 0 {
			best = item
			found = true
		}
	}

	return best, found
}

// NoneMatch reports whether no element matches fn. It is the negation of
// [Stream.AnyMatch], so it reports true for an empty stream and when the
// stream's context ends first.
func (s Stream[T]) NoneMatch(fn func(context.Context, T) bool) bool {
	return !s.AnyMatch(fn)
}

// Peek calls fn for each element as a side-effect and forwards the element unchanged.
func (s Stream[T]) Peek(fn func(context.Context, T)) Stream[T] {
	return stage(s, "goflow.peek", func(ctx context.Context, emit func(T) bool) error {
		for item := range s.items() {
			fn(ctx, item)

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// Process consumes the stream with n concurrent workers.
// Errors are collected and returned via [errors.Join] once the stream ends.
//
// With [WithErrorHandler], errors are passed to the handler instead and not
// retained, which suits long-lived streams. Use [WithGroupOptions] to configure
// the underlying [gofuncy.Group].
func (s Stream[T]) Process(n int, fn func(context.Context, T) error, opts ...Option) error {
	c := newConfig(opts)

	if n < 1 {
		n = 1
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	onErr := func(ctx context.Context, err error) error {
		if c.errorHandler != nil {
			c.errorHandler(ctx, err)

			return nil
		}

		mu.Lock()
		defer mu.Unlock()

		errs = append(errs, err)

		return nil
	}

	g := gofuncy.NewGroup(s.ctx, append(s.p.groupOpts("goflow.process"), c.groupOpts...)...)
	for range n {
		g.Add(func(ctx context.Context) error {
			// Same cancel policy as ForEach; the error is reported once below.
			if err := consume(ctx, s, semconv.StageNameProcess, fn, onErr); err != nil && ctx.Err() == nil {
				return err
			}

			return nil
		}, gofuncy.WithName("goflow.process.worker"))
	}

	errs = append(errs, g.Wait(), s.ctx.Err())

	return errors.Join(errs...)
}

// Reverse collects all elements and emits them in reverse order.
func (s Stream[T]) Reverse() Stream[T] {
	return stage(s, "goflow.reverse", func(_ context.Context, emit func(T) bool) error {
		for _, item := range slices.Backward(s.Collect()) {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// Skip drops the first n elements and emits the rest.
func (s Stream[T]) Skip(n int) Stream[T] {
	return stage(s, "goflow.skip", func(_ context.Context, emit func(T) bool) error {
		count := 0

		for item := range s.items() {
			if count < n {
				count++

				continue
			}

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// Sort collects all elements, sorts them using cmp, and emits in sorted order.
func (s Stream[T]) Sort(cmp func(T, T) int) Stream[T] {
	return stage(s, "goflow.sort", func(_ context.Context, emit func(T) bool) error {
		items := s.Collect()
		slices.SortFunc(items, cmp)

		for _, item := range items {
			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// Take emits the first n elements then closes the stream.
func (s Stream[T]) Take(n int) Stream[T] {
	if n <= 0 {
		return closed[T](s.ctx)
	}

	return stage(s, "goflow.take", func(_ context.Context, emit func(T) bool) error {
		count := 0

		for item := range s.items() {
			if count >= n || !emit(item) {
				return nil
			}

			count++
		}

		return nil
	})
}

// Tee broadcasts every element to n output streams. Unlike [Stream.FanOut]
// which round-robins, Tee sends each element to all streams.
func (s Stream[T]) Tee(n int) []Stream[T] {
	return s.spread(n, "goflow.tee", func(int, []chan T) int { return -1 })
}

// Throttle rate-limits the stream to at most one element per duration d.
func (s Stream[T]) Throttle(d time.Duration) Stream[T] {
	return stage(s, "goflow.throttle", func(ctx context.Context, emit func(T) bool) error {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		first := true

		for item := range s.items() {
			if !first {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}

			first = false

			if !emit(item) {
				return nil
			}
		}

		return nil
	})
}

// WithOptions returns a shallow copy of the stream with the given options appended.
func (s Stream[T]) WithOptions(opts ...gofuncy.GoOption) Stream[T] {
	s.opts = slices.Concat(s.opts, opts)

	return s
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

// envelopes returns exactly one non-nil channel: the envelope channel when the
// stream carries trace origins, the plain channel otherwise.
func (s Stream[T]) envelopes() (<-chan envelope[T], <-chan T) {
	if s.env != nil {
		return s.env.ch, nil
	}

	return nil, s.source
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

// spread starts one stage that feeds n output streams. pick returns the index
// of the output for the i-th element, or -1 to send it to every output.
func (s Stream[T]) spread(n int, name string, pick func(i int, outs []chan T) int) []Stream[T] {
	if n <= 0 || s.ctx.Err() != nil {
		return nil
	}

	outs := make([]chan T, n)
	for i := range outs {
		outs[i] = make(chan T)
	}

	goStage(s.ctx, func(ctx context.Context) error {
		defer func() {
			for _, ch := range outs {
				close(ch)
			}
		}()

		i := 0

		for item := range s.items() {
			if j := pick(i, outs); j >= 0 {
				if !send(ctx, outs[j], item) {
					return nil
				}
			} else {
				for _, ch := range outs {
					if !send(ctx, ch, item) {
						return nil
					}
				}
			}

			i++
		}

		return nil
	}, s.stageOpts(name)...)

	streams := make([]Stream[T], n)
	for i, ch := range outs {
		streams[i] = derive[T](s, ch)
	}

	return streams
}

// stageOpts returns the stream's gofuncy options plus the stage name, without
// writing into the spare capacity of s.opts.
func (s Stream[T]) stageOpts(name string) []gofuncy.GoOption {
	return append(slices.Clip(s.opts), gofuncy.WithName(name))
}
