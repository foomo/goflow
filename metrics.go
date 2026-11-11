package goflow

import (
	"context"
	"runtime/debug"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
	"github.com/foomo/goflow/semconv/goflowconv"
)

// ScopeName is the OpenTelemetry instrumentation scope name used by goflow.
const ScopeName = "github.com/foomo/goflow"

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

func newTracer(tp trace.TracerProvider) trace.Tracer {
	if tp == nil {
		tp = otel.GetTracerProvider()
	}

	return tp.Tracer(ScopeName,
		trace.WithInstrumentationVersion(version()),
		trace.WithSchemaURL(otelsemconv.SchemaURL),
	)
}

// metrics holds the instruments of one pipeline. The attribute sets of the
// hot-path measurements are fixed per pipeline, so they are built once here
// instead of once per element.
type metrics struct {
	meter     metric.Meter
	name      string
	enqueued  goflowconv.PipeEnqueued
	dropped   goflowconv.PipeDropped
	depth     goflowconv.PipeQueueDepth
	capacity  goflowconv.PipeQueueCapacity
	batchSize goflowconv.BatchSize
	batchBody goflowconv.BatchBodySize
	duration  goflowconv.StageDuration

	nameAttrs  metric.MeasurementOption
	dropAttrs  map[semconv.DropReasonAttr]metric.MeasurementOption
	flushAttrs map[semconv.FlushReasonAttr]metric.MeasurementOption
	stageAttrs map[semconv.StageNameAttr]metric.MeasurementOption
}

func newMetrics(mp metric.MeterProvider, name string) *metrics {
	if mp == nil {
		mp = otel.GetMeterProvider()
	}

	m := &metrics{
		meter: mp.Meter(ScopeName,
			metric.WithInstrumentationVersion(version()),
			metric.WithSchemaURL(otelsemconv.SchemaURL),
		),
		name:       name,
		dropAttrs:  map[semconv.DropReasonAttr]metric.MeasurementOption{},
		flushAttrs: map[semconv.FlushReasonAttr]metric.MeasurementOption{},
		stageAttrs: map[semconv.StageNameAttr]metric.MeasurementOption{},
	}

	var errs [7]error

	m.enqueued, errs[0] = goflowconv.NewPipeEnqueued(m.meter)
	m.dropped, errs[1] = goflowconv.NewPipeDropped(m.meter)
	m.depth, errs[2] = goflowconv.NewPipeQueueDepth(m.meter)
	m.capacity, errs[3] = goflowconv.NewPipeQueueCapacity(m.meter)
	m.batchSize, errs[4] = goflowconv.NewBatchSize(m.meter)
	m.batchBody, errs[5] = goflowconv.NewBatchBodySize(m.meter)
	m.duration, errs[6] = goflowconv.NewStageDuration(m.meter)

	for _, err := range errs {
		if err != nil {
			otel.Handle(err)
		}
	}

	m.nameAttrs = m.attrs()

	for _, r := range []semconv.DropReasonAttr{semconv.DropReasonFull, semconv.DropReasonClosed, semconv.DropReasonCanceled} {
		m.dropAttrs[r] = m.attrs(semconv.DropReason(r))
	}

	for _, r := range []semconv.FlushReasonAttr{semconv.FlushReasonCount, semconv.FlushReasonBytes, semconv.FlushReasonInterval, semconv.FlushReasonClose} {
		m.flushAttrs[r] = m.attrs(semconv.FlushReason(r))
	}

	for _, s := range []semconv.StageNameAttr{semconv.StageNameForEach, semconv.StageNameProcess} {
		m.stageAttrs[s] = m.attrs(semconv.StageName(s))
	}

	return m
}

// attrs returns the pipeline name attribute plus kv as a prebuilt set.
func (m *metrics) attrs(kv ...attribute.KeyValue) metric.MeasurementOption {
	return metric.WithAttributeSet(attribute.NewSet(append([]attribute.KeyValue{semconv.PipelineName(m.name)}, kv...)...))
}

// observeQueue registers the queue callbacks. The returned func unregisters them.
func (m *metrics) observeQueue(length, capacity func() int) func() {
	reg, err := m.meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(m.depth.Inst(), int64(length()), m.nameAttrs)
		o.ObserveInt64(m.capacity.Inst(), int64(capacity()), m.nameAttrs)

		return nil
	}, m.depth.Inst(), m.capacity.Inst())
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

func (m *metrics) enqueuedItem(ctx context.Context) {
	m.enqueued.Inst().Add(ctx, 1, m.nameAttrs)
}

func (m *metrics) droppedItems(ctx context.Context, reason semconv.DropReasonAttr, n int64) {
	m.dropped.Inst().Add(ctx, n, m.dropAttrs[reason])
}

func (m *metrics) stage(ctx context.Context, stage semconv.StageNameAttr, d time.Duration, err error) {
	if err != nil {
		m.duration.Record(ctx, d.Seconds(), m.name, stage, m.duration.AttrErrorType(err))

		return
	}

	m.duration.Inst().Record(ctx, d.Seconds(), m.stageAttrs[stage])
}

func (m *metrics) flushedBatch(ctx context.Context, size int, reason semconv.FlushReasonAttr) {
	m.batchSize.Inst().Record(ctx, int64(size), m.flushAttrs[reason])
}

func (m *metrics) flushedBody(ctx context.Context, bytes int, reason semconv.FlushReasonAttr) {
	m.batchBody.Inst().Record(ctx, int64(bytes), m.flushAttrs[reason])
}
