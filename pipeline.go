package goflow

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/foomo/gofuncy"
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

// groupOpts returns the gofuncy options for a stage group named name (nil-safe).
func (p *pipeline) groupOpts(name string) []gofuncy.GroupOption {
	opts := []gofuncy.GroupOption{gofuncy.WithName(name)}
	if p != nil && !p.stageTracing {
		opts = append(opts, gofuncy.WithoutTracing())
	}

	return opts
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
