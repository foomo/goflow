package goflow

import (
	"context"
	"slices"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
)

// consumer runs fn for the elements of one consuming stage and records the
// stage telemetry. It is built once per stage, so the constant span
// attributes are computed once.
type consumer[T any] struct {
	p     *pipeline
	stage semconv.StageNameAttr
	attrs []attribute.KeyValue
	fn    func(context.Context, T) error
}

func newConsumer[T any](s Stream[T], stage semconv.StageNameAttr, fn func(context.Context, T) error) consumer[T] {
	c := consumer[T]{p: s.p, stage: stage, fn: fn}
	if s.p != nil {
		c.attrs = []attribute.KeyValue{semconv.PipelineName(s.p.name), semconv.StageName(stage)}
	}

	return c
}

// run runs fn for one element and records the stage duration.
//
// When the element has origin spans (it was sent with a traced ctx), it
// starts a new root consumer span named "process <pipeline>" linked to them.
// Elements without an origin get no span: tracing is opt-in at the producer.
// The stream context carries no span of its own (PipeFunc detaches it, and a
// Pipe's producer spans end long before the element is consumed), so a child
// span would have no parent.
func (c consumer[T]) run(ctx context.Context, e envelope[T]) error {
	if c.p == nil {
		return c.fn(ctx, e.v)
	}

	var span trace.Span

	if links := e.spanLinks(); len(links) > 0 {
		attrs := c.attrs
		if b := e.batch; b != nil {
			attrs = append(slices.Clip(attrs), semconv.FlushReason(b.reason), semconv.BatchItemCount(b.count))
			if b.droppedLinks > 0 {
				attrs = append(attrs, semconv.LinkDroppedCount(b.droppedLinks))
			}
		}

		ctx, span = c.p.tracer.Start(ctx, c.p.spanName,
			trace.WithNewRoot(),
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithLinks(links...),
			trace.WithAttributes(attrs...),
		)
		defer span.End()
	}

	var start time.Time

	timed := c.p.duration.Enabled(ctx)
	if timed {
		start = time.Now()
	}

	err := c.fn(ctx, e.v)

	if timed {
		if err != nil {
			c.p.duration.Record(ctx, time.Since(start).Seconds(), c.p.name, c.stage, c.p.duration.AttrErrorType(err))
		} else {
			c.p.duration.RecordSet(ctx, time.Since(start).Seconds(), c.p.stageSets[c.stage])
		}
	}

	if err != nil && span != nil {
		span.SetAttributes(c.p.duration.AttrErrorType(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}
