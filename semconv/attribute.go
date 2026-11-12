package semconv

import "go.opentelemetry.io/otel/attribute"

const (
	// PipelineNameKey is the attribute key for the pipeline name set with
	// goflow.WithName. Low cardinality: it must be a static label.
	PipelineNameKey = attribute.Key("goflow.pipeline.name")
	// StageNameKey is the attribute key for the consumer stage that handled
	// an element.
	StageNameKey = attribute.Key("goflow.stage.name")
	// FlushReasonKey is the attribute key for why a batch was flushed.
	FlushReasonKey = attribute.Key("goflow.batch.flush.reason")
	// DropReasonKey is the attribute key for why the pipe rejected an element.
	DropReasonKey = attribute.Key("goflow.pipe.drop.reason")
	// BatchItemCountKey is the span attribute key for the number of elements
	// in a batch. Span only.
	BatchItemCountKey = attribute.Key("goflow.batch.item.count")
	// LinkDroppedCountKey is the span attribute key for the number of origin
	// links that were not added because of the per-span link cap. Span only.
	LinkDroppedCountKey = attribute.Key("goflow.link.dropped_count")
)

// PipelineName returns a [PipelineNameKey] attribute.
func PipelineName(v string) attribute.KeyValue { return PipelineNameKey.String(v) }

// BatchItemCount returns a [BatchItemCountKey] attribute.
func BatchItemCount(v int) attribute.KeyValue { return BatchItemCountKey.Int(v) }

// LinkDroppedCount returns a [LinkDroppedCountKey] attribute.
func LinkDroppedCount(v int) attribute.KeyValue { return LinkDroppedCountKey.Int(v) }

// StageNameAttr is an enum for [StageNameKey] values.
type StageNameAttr string

const (
	// StageNameForEach is the Stream.ForEach consumer.
	StageNameForEach StageNameAttr = "foreach"
	// StageNameProcess is the Stream.Process consumer.
	StageNameProcess StageNameAttr = "process"
)

// StageName returns a [StageNameKey] attribute.
func StageName(v StageNameAttr) attribute.KeyValue { return StageNameKey.String(string(v)) }

// FlushReasonAttr is an enum for [FlushReasonKey] values.
type FlushReasonAttr string

const (
	// FlushReasonCount means the batch reached goflow.MaxItems.
	FlushReasonCount FlushReasonAttr = "count"
	// FlushReasonBytes means the batch reached goflow.MaxBytes, or the next
	// element would have exceeded it.
	FlushReasonBytes FlushReasonAttr = "bytes"
	// FlushReasonInterval means goflow.MaxWait passed since the first element.
	FlushReasonInterval FlushReasonAttr = "interval"
	// FlushReasonClose means the input closed and the partial batch was flushed.
	FlushReasonClose FlushReasonAttr = "close"
)

// FlushReason returns a [FlushReasonKey] attribute.
func FlushReason(v FlushReasonAttr) attribute.KeyValue { return FlushReasonKey.String(string(v)) }

// DropReasonAttr is an enum for [DropReasonKey] values.
type DropReasonAttr string

const (
	// DropReasonFull means Sender.Try found the buffer full.
	DropReasonFull DropReasonAttr = "full"
	// DropReasonClosed means the pipe was closed.
	DropReasonClosed DropReasonAttr = "closed"
	// DropReasonCanceled means the sender's context ended before Sender.Send
	// could queue the element.
	DropReasonCanceled DropReasonAttr = "canceled"
)

// DropReason returns a [DropReasonKey] attribute.
func DropReason(v DropReasonAttr) attribute.KeyValue { return DropReasonKey.String(string(v)) }
