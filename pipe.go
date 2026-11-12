package goflow

import (
	"context"
)

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
