package goflow

import (
	"context"
	"slices"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
)

// maxLinks caps the number of span links per consumer span.
const maxLinks = 128

// pipeline holds the telemetry shared by all streams derived from one Pipe.
// It has no lifecycle of its own: Pipe streams stop with their context, and
// PipeFunc owns draining (see PipeFunc).
type pipeline struct {
	name         string
	spanName     string
	stageTracing bool
	tracer       trace.Tracer
	m            *metrics
}

func newPipeline(c config) *pipeline {
	name := c.name
	if name == "" {
		name = "goflow.pipe"
	}

	return &pipeline{
		name:         name,
		spanName:     "process " + name,
		stageTracing: c.stageTracing,
		tracer:       newTracer(c.tracerProvider),
		m:            newMetrics(c.meterProvider, name),
	}
}

// stageOpts returns the gofuncy options for stage goroutines (nil-safe).
func (p *pipeline) stageOpts() []gofuncy.GoOption {
	if p == nil || p.stageTracing {
		return nil
	}

	return []gofuncy.GoOption{gofuncy.WithoutTracing()}
}

// groupOpts returns the gofuncy options for a stage group named name (nil-safe).
func (p *pipeline) groupOpts(name string) []gofuncy.GroupOption {
	opts := []gofuncy.GroupOption{gofuncy.WithName(name)}
	if p != nil && !p.stageTracing {
		opts = append(opts, gofuncy.WithoutTracing())
	}

	return opts
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

	timed := c.p.m.duration.Enabled(ctx)
	if timed {
		start = time.Now()
	}

	err := c.fn(ctx, e.v)

	if timed {
		c.p.m.stage(ctx, c.stage, time.Since(start), err)
	}

	if err != nil && span != nil {
		span.SetAttributes(c.p.m.duration.AttrErrorType(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}
