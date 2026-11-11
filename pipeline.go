package goflow

import (
	"context"
	"runtime/debug"
	"slices"
	"sync"
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
	"github.com/foomo/goflow/semconv/goflowconv"
)

// ScopeName is the OpenTelemetry instrumentation scope name used by goflow.
const ScopeName = "github.com/foomo/goflow"

// maxLinks caps the number of span links per consumer span.
const maxLinks = 128

// version is the goflow module version from the build info, used as the
// instrumentation scope version.
var version = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}

	if info.Main.Path == ScopeName {
		return info.Main.Version
	}

	for _, dep := range info.Deps {
		if dep.Path == ScopeName {
			return dep.Version
		}
	}

	return ""
})

// pipeline holds the telemetry shared by all streams derived from one Pipe.
// It has no lifecycle of its own: Pipe streams stop with their context, and
// PipeFunc owns draining (see PipeFunc).
type pipeline struct {
	name         string
	spanName     string
	stageTracing bool
	tracer       trace.Tracer
	meter        metric.Meter

	enqueued  goflowconv.PipeEnqueued
	dropped   goflowconv.PipeDropped
	depth     goflowconv.PipeQueueDepth
	capacity  goflowconv.PipeQueueCapacity
	batchSize goflowconv.BatchSize
	batchBody goflowconv.BatchBodySize
	duration  goflowconv.StageDuration

	// The hot-path attribute sets are fixed per pipeline, so they are built
	// once here instead of once per element.
	nameSet   attribute.Set
	dropSets  map[semconv.DropReasonAttr]attribute.Set
	flushSets map[semconv.FlushReasonAttr]attribute.Set
	stageSets map[semconv.StageNameAttr]attribute.Set
}

func newPipeline(c config) *pipeline {
	name := c.name
	if name == "" {
		name = "goflow.pipe"
	}

	tp := c.tracerProvider
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	mp := c.meterProvider
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	p := &pipeline{
		name:         name,
		spanName:     "process " + name,
		stageTracing: c.stageTracing,
		tracer: tp.Tracer(ScopeName,
			trace.WithInstrumentationVersion(version()),
			trace.WithSchemaURL(otelsemconv.SchemaURL),
		),
		meter: mp.Meter(ScopeName,
			metric.WithInstrumentationVersion(version()),
			metric.WithSchemaURL(otelsemconv.SchemaURL),
		),
		dropSets:  map[semconv.DropReasonAttr]attribute.Set{},
		flushSets: map[semconv.FlushReasonAttr]attribute.Set{},
		stageSets: map[semconv.StageNameAttr]attribute.Set{},
	}

	var errs [7]error

	p.enqueued, errs[0] = goflowconv.NewPipeEnqueued(p.meter)
	p.dropped, errs[1] = goflowconv.NewPipeDropped(p.meter)
	p.depth, errs[2] = goflowconv.NewPipeQueueDepth(p.meter)
	p.capacity, errs[3] = goflowconv.NewPipeQueueCapacity(p.meter)
	p.batchSize, errs[4] = goflowconv.NewBatchSize(p.meter)
	p.batchBody, errs[5] = goflowconv.NewBatchBodySize(p.meter)
	p.duration, errs[6] = goflowconv.NewStageDuration(p.meter)

	for _, err := range errs {
		if err != nil {
			otel.Handle(err)
		}
	}

	p.nameSet = p.attrSet()

	for _, r := range []semconv.DropReasonAttr{semconv.DropReasonFull, semconv.DropReasonClosed, semconv.DropReasonCanceled} {
		p.dropSets[r] = p.attrSet(semconv.DropReason(r))
	}

	for _, r := range []semconv.FlushReasonAttr{semconv.FlushReasonCount, semconv.FlushReasonBytes, semconv.FlushReasonInterval, semconv.FlushReasonClose} {
		p.flushSets[r] = p.attrSet(semconv.FlushReason(r))
	}

	for _, s := range []semconv.StageNameAttr{semconv.StageNameForEach, semconv.StageNameProcess} {
		p.stageSets[s] = p.attrSet(semconv.StageName(s))
	}

	return p
}

// attrSet returns the pipeline name attribute plus kv as a set.
func (p *pipeline) attrSet(kv ...attribute.KeyValue) attribute.Set {
	return attribute.NewSet(append([]attribute.KeyValue{semconv.PipelineName(p.name)}, kv...)...)
}

// observeQueue registers the queue callbacks. The returned func unregisters them.
func (p *pipeline) observeQueue(length, capacity func() int) func() {
	opt := metric.WithAttributeSet(p.nameSet)

	reg, err := p.meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(p.depth.Inst(), int64(length()), opt)
		o.ObserveInt64(p.capacity.Inst(), int64(capacity()), opt)

		return nil
	}, p.depth.Inst(), p.capacity.Inst())
	if err != nil {
		otel.Handle(err)

		return func() {}
	}

	return func() {
		if err := reg.Unregister(); err != nil {
			otel.Handle(err)
		}
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
