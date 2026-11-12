package goflowconv

import (
	"context"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	otelsemconv "go.opentelemetry.io/otel/semconv/v1.40.0"

	"github.com/foomo/goflow/semconv"
)

// withRequired appends required attributes without writing into the spare
// capacity of the caller's slice.
func withRequired(attrs []attribute.KeyValue, required ...attribute.KeyValue) metric.MeasurementOption {
	if len(attrs) == 0 {
		return metric.WithAttributes(required...)
	}

	return metric.WithAttributes(append(slices.Clip(attrs), required...)...)
}

// ------------------------------------------------------------------------------------------------
// ~ goflow.pipe.enqueued
// ------------------------------------------------------------------------------------------------

// PipeEnqueued is an instrument used to record metric values conforming to
// the "goflow.pipe.enqueued" semantic conventions. It represents the number
// of elements queued by a pipe.
type PipeEnqueued struct {
	metric.Int64Counter
}

var newPipeEnqueuedOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of elements queued by the pipe."),
	metric.WithUnit("{item}"),
}

// NewPipeEnqueued returns a new PipeEnqueued instrument. Options in opt are
// applied after the defaults. A nil m or a creation error yields a no-op
// instrument.
func NewPipeEnqueued(m metric.Meter, opt ...metric.Int64CounterOption) (PipeEnqueued, error) {
	if m == nil {
		return PipeEnqueued{noop.Int64Counter{}}, nil
	}

	i, err := m.Int64Counter("goflow.pipe.enqueued", append(slices.Clip(newPipeEnqueuedOpts), opt...)...)
	if err != nil {
		return PipeEnqueued{noop.Int64Counter{}}, err
	}

	return PipeEnqueued{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PipeEnqueued) Inst() metric.Int64Counter { return m.Int64Counter }

// Name returns the semantic convention name of the instrument.
func (PipeEnqueued) Name() string { return "goflow.pipe.enqueued" }

// Unit returns the semantic convention unit of the instrument.
func (PipeEnqueued) Unit() string { return "{item}" }

// Description returns the semantic convention description of the instrument.
func (PipeEnqueued) Description() string { return "Number of elements queued by the pipe." }

// Add adds incr for the pipeline named pipelineName.
func (m PipeEnqueued) Add(ctx context.Context, incr int64, pipelineName string, attrs ...attribute.KeyValue) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Counter.Add(ctx, incr, withRequired(attrs, semconv.PipelineName(pipelineName)))
}

// AddSet adds incr with a precomputed attribute set. The set must hold the
// attributes Add requires.
func (m PipeEnqueued) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Counter.Add(ctx, incr, metric.WithAttributeSet(set))
}

// ------------------------------------------------------------------------------------------------
// ~ goflow.pipe.dropped
// ------------------------------------------------------------------------------------------------

// PipeDropped is an instrument used to record metric values conforming to
// the "goflow.pipe.dropped" semantic conventions. It represents the number
// of elements a pipe rejected.
type PipeDropped struct {
	metric.Int64Counter
}

var newPipeDroppedOpts = []metric.Int64CounterOption{
	metric.WithDescription("Number of elements the pipe rejected."),
	metric.WithUnit("{item}"),
}

// NewPipeDropped returns a new PipeDropped instrument. See [NewPipeEnqueued]
// for option and nil handling.
func NewPipeDropped(m metric.Meter, opt ...metric.Int64CounterOption) (PipeDropped, error) {
	if m == nil {
		return PipeDropped{noop.Int64Counter{}}, nil
	}

	i, err := m.Int64Counter("goflow.pipe.dropped", append(slices.Clip(newPipeDroppedOpts), opt...)...)
	if err != nil {
		return PipeDropped{noop.Int64Counter{}}, err
	}

	return PipeDropped{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PipeDropped) Inst() metric.Int64Counter { return m.Int64Counter }

// Name returns the semantic convention name of the instrument.
func (PipeDropped) Name() string { return "goflow.pipe.dropped" }

// Unit returns the semantic convention unit of the instrument.
func (PipeDropped) Unit() string { return "{item}" }

// Description returns the semantic convention description of the instrument.
func (PipeDropped) Description() string { return "Number of elements the pipe rejected." }

// Add adds incr for the pipeline named pipelineName and the given drop reason.
func (m PipeDropped) Add(ctx context.Context, incr int64, pipelineName string, reason semconv.DropReasonAttr, attrs ...attribute.KeyValue) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Counter.Add(ctx, incr, withRequired(attrs, semconv.PipelineName(pipelineName), semconv.DropReason(reason)))
}

// AddSet adds incr with a precomputed attribute set. The set must hold the
// attributes Add requires.
func (m PipeDropped) AddSet(ctx context.Context, incr int64, set attribute.Set) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Counter.Add(ctx, incr, metric.WithAttributeSet(set))
}

// ------------------------------------------------------------------------------------------------
// ~ goflow.pipe.queue.depth
// ------------------------------------------------------------------------------------------------

// PipeQueueDepth is an instrument used to record metric values conforming
// to the "goflow.pipe.queue.depth" semantic conventions. It represents the
// number of elements waiting in a pipe buffer. Observe it with
// metric.Meter.RegisterCallback.
type PipeQueueDepth struct {
	metric.Int64ObservableUpDownCounter
}

var newPipeQueueDepthOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Number of elements waiting in the pipe buffer."),
	metric.WithUnit("{item}"),
}

// NewPipeQueueDepth returns a new PipeQueueDepth instrument. See
// [NewPipeEnqueued] for option and nil handling.
func NewPipeQueueDepth(m metric.Meter, opt ...metric.Int64ObservableUpDownCounterOption) (PipeQueueDepth, error) {
	if m == nil {
		return PipeQueueDepth{noop.Int64ObservableUpDownCounter{}}, nil
	}

	i, err := m.Int64ObservableUpDownCounter("goflow.pipe.queue.depth", append(slices.Clip(newPipeQueueDepthOpts), opt...)...)
	if err != nil {
		return PipeQueueDepth{noop.Int64ObservableUpDownCounter{}}, err
	}

	return PipeQueueDepth{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PipeQueueDepth) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (PipeQueueDepth) Name() string { return "goflow.pipe.queue.depth" }

// Unit returns the semantic convention unit of the instrument.
func (PipeQueueDepth) Unit() string { return "{item}" }

// Description returns the semantic convention description of the instrument.
func (PipeQueueDepth) Description() string { return "Number of elements waiting in the pipe buffer." }

// ------------------------------------------------------------------------------------------------
// ~ goflow.pipe.queue.capacity
// ------------------------------------------------------------------------------------------------

// PipeQueueCapacity is an instrument used to record metric values conforming
// to the "goflow.pipe.queue.capacity" semantic conventions. It represents the
// capacity of a pipe buffer. Observe it with metric.Meter.RegisterCallback.
type PipeQueueCapacity struct {
	metric.Int64ObservableUpDownCounter
}

var newPipeQueueCapacityOpts = []metric.Int64ObservableUpDownCounterOption{
	metric.WithDescription("Capacity of the pipe buffer."),
	metric.WithUnit("{item}"),
}

// NewPipeQueueCapacity returns a new PipeQueueCapacity instrument. See
// [NewPipeEnqueued] for option and nil handling.
func NewPipeQueueCapacity(m metric.Meter, opt ...metric.Int64ObservableUpDownCounterOption) (PipeQueueCapacity, error) {
	if m == nil {
		return PipeQueueCapacity{noop.Int64ObservableUpDownCounter{}}, nil
	}

	i, err := m.Int64ObservableUpDownCounter("goflow.pipe.queue.capacity", append(slices.Clip(newPipeQueueCapacityOpts), opt...)...)
	if err != nil {
		return PipeQueueCapacity{noop.Int64ObservableUpDownCounter{}}, err
	}

	return PipeQueueCapacity{i}, nil
}

// Inst returns the underlying metric instrument.
func (m PipeQueueCapacity) Inst() metric.Int64ObservableUpDownCounter {
	return m.Int64ObservableUpDownCounter
}

// Name returns the semantic convention name of the instrument.
func (PipeQueueCapacity) Name() string { return "goflow.pipe.queue.capacity" }

// Unit returns the semantic convention unit of the instrument.
func (PipeQueueCapacity) Unit() string { return "{item}" }

// Description returns the semantic convention description of the instrument.
func (PipeQueueCapacity) Description() string { return "Capacity of the pipe buffer." }

// ------------------------------------------------------------------------------------------------
// ~ goflow.batch.size
// ------------------------------------------------------------------------------------------------

// BatchSize is an instrument used to record metric values conforming to the
// "goflow.batch.size" semantic conventions. It represents the number of
// elements per flushed batch.
type BatchSize struct {
	metric.Int64Histogram
}

var newBatchSizeOpts = []metric.Int64HistogramOption{
	metric.WithDescription("Number of elements per flushed batch."),
	metric.WithUnit("{item}"),
	metric.WithExplicitBucketBoundaries(1, 2, 5, 10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000),
}

// NewBatchSize returns a new BatchSize instrument. See [NewPipeEnqueued]
// for option and nil handling.
func NewBatchSize(m metric.Meter, opt ...metric.Int64HistogramOption) (BatchSize, error) {
	if m == nil {
		return BatchSize{noop.Int64Histogram{}}, nil
	}

	i, err := m.Int64Histogram("goflow.batch.size", append(slices.Clip(newBatchSizeOpts), opt...)...)
	if err != nil {
		return BatchSize{noop.Int64Histogram{}}, err
	}

	return BatchSize{i}, nil
}

// Inst returns the underlying metric instrument.
func (m BatchSize) Inst() metric.Int64Histogram { return m.Int64Histogram }

// Name returns the semantic convention name of the instrument.
func (BatchSize) Name() string { return "goflow.batch.size" }

// Unit returns the semantic convention unit of the instrument.
func (BatchSize) Unit() string { return "{item}" }

// Description returns the semantic convention description of the instrument.
func (BatchSize) Description() string { return "Number of elements per flushed batch." }

// Record records val for the pipeline named pipelineName and the given flush reason.
func (m BatchSize) Record(ctx context.Context, val int64, pipelineName string, reason semconv.FlushReasonAttr, attrs ...attribute.KeyValue) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Histogram.Record(ctx, val, withRequired(attrs, semconv.PipelineName(pipelineName), semconv.FlushReason(reason)))
}

// RecordSet records val with a precomputed attribute set. The set must hold
// the attributes Record requires.
func (m BatchSize) RecordSet(ctx context.Context, val int64, set attribute.Set) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Histogram.Record(ctx, val, metric.WithAttributeSet(set))
}

// ------------------------------------------------------------------------------------------------
// ~ goflow.batch.body.size
// ------------------------------------------------------------------------------------------------

// BatchBodySize is an instrument used to record metric values conforming to
// the "goflow.batch.body.size" semantic conventions. It represents the summed
// element size per flushed batch, as reported by the goflow.MaxBytes size
// function.
type BatchBodySize struct {
	metric.Int64Histogram
}

var newBatchBodySizeOpts = []metric.Int64HistogramOption{
	metric.WithDescription("Summed element size per flushed batch."),
	metric.WithUnit("By"),
	metric.WithExplicitBucketBoundaries(1<<10, 4<<10, 16<<10, 64<<10, 256<<10, 1<<20, 4<<20, 16<<20),
}

// NewBatchBodySize returns a new BatchBodySize instrument. See
// [NewPipeEnqueued] for option and nil handling.
func NewBatchBodySize(m metric.Meter, opt ...metric.Int64HistogramOption) (BatchBodySize, error) {
	if m == nil {
		return BatchBodySize{noop.Int64Histogram{}}, nil
	}

	i, err := m.Int64Histogram("goflow.batch.body.size", append(slices.Clip(newBatchBodySizeOpts), opt...)...)
	if err != nil {
		return BatchBodySize{noop.Int64Histogram{}}, err
	}

	return BatchBodySize{i}, nil
}

// Inst returns the underlying metric instrument.
func (m BatchBodySize) Inst() metric.Int64Histogram { return m.Int64Histogram }

// Name returns the semantic convention name of the instrument.
func (BatchBodySize) Name() string { return "goflow.batch.body.size" }

// Unit returns the semantic convention unit of the instrument.
func (BatchBodySize) Unit() string { return "By" }

// Description returns the semantic convention description of the instrument.
func (BatchBodySize) Description() string { return "Summed element size per flushed batch." }

// Record records val for the pipeline named pipelineName and the given flush reason.
func (m BatchBodySize) Record(ctx context.Context, val int64, pipelineName string, reason semconv.FlushReasonAttr, attrs ...attribute.KeyValue) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Histogram.Record(ctx, val, withRequired(attrs, semconv.PipelineName(pipelineName), semconv.FlushReason(reason)))
}

// RecordSet records val with a precomputed attribute set. The set must hold
// the attributes Record requires.
func (m BatchBodySize) RecordSet(ctx context.Context, val int64, set attribute.Set) {
	if !m.Enabled(ctx) {
		return
	}

	m.Int64Histogram.Record(ctx, val, metric.WithAttributeSet(set))
}

// ------------------------------------------------------------------------------------------------
// ~ goflow.stage.duration
// ------------------------------------------------------------------------------------------------

// StageDuration is an instrument used to record metric values conforming to
// the "goflow.stage.duration" semantic conventions. It represents the
// duration of handling one element in a consumer stage. Its count is the
// stage throughput; with error.type it also yields the error rate.
type StageDuration struct {
	metric.Float64Histogram
}

var newStageDurationOpts = []metric.Float64HistogramOption{
	metric.WithDescription("Duration of handling one element in a consumer stage."),
	metric.WithUnit("s"),
	metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60),
}

// NewStageDuration returns a new StageDuration instrument. See
// [NewPipeEnqueued] for option and nil handling.
func NewStageDuration(m metric.Meter, opt ...metric.Float64HistogramOption) (StageDuration, error) {
	if m == nil {
		return StageDuration{noop.Float64Histogram{}}, nil
	}

	i, err := m.Float64Histogram("goflow.stage.duration", append(slices.Clip(newStageDurationOpts), opt...)...)
	if err != nil {
		return StageDuration{noop.Float64Histogram{}}, err
	}

	return StageDuration{i}, nil
}

// Inst returns the underlying metric instrument.
func (m StageDuration) Inst() metric.Float64Histogram { return m.Float64Histogram }

// Name returns the semantic convention name of the instrument.
func (StageDuration) Name() string { return "goflow.stage.duration" }

// Unit returns the semantic convention unit of the instrument.
func (StageDuration) Unit() string { return "s" }

// Description returns the semantic convention description of the instrument.
func (StageDuration) Description() string {
	return "Duration of handling one element in a consumer stage."
}

// Record records val in seconds for the pipeline named pipelineName and the given stage.
func (m StageDuration) Record(ctx context.Context, val float64, pipelineName string, stage semconv.StageNameAttr, attrs ...attribute.KeyValue) {
	if !m.Enabled(ctx) {
		return
	}

	m.Float64Histogram.Record(ctx, val, withRequired(attrs, semconv.PipelineName(pipelineName), semconv.StageName(stage)))
}

// RecordSet records val with a precomputed attribute set. The set must hold
// the attributes Record requires.
func (m StageDuration) RecordSet(ctx context.Context, val float64, set attribute.Set) {
	if !m.Enabled(ctx) {
		return
	}

	m.Float64Histogram.Record(ctx, val, metric.WithAttributeSet(set))
}

// AttrErrorType returns the conditionally required "error.type" attribute,
// set only when handling the element failed.
func (StageDuration) AttrErrorType(err error) attribute.KeyValue {
	return otelsemconv.ErrorType(err)
}
