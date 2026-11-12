package goflow

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// envelope carries an element together with its trace origin.
type envelope[T any] struct {
	v T
	// sc is the span context captured by Pipe at send time.
	sc trace.SpanContext
	// batch is set by Batch for each flushed batch.
	batch *batchMeta
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
