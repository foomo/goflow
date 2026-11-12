package goflow

import (
	"github.com/foomo/goflow/semconv"
	"go.opentelemetry.io/otel/trace"
)

// batchMeta describes a flushed batch.
type batchMeta struct {
	// links are the aggregated origins of the batch, capped at maxLinks.
	links        []trace.Link
	droppedLinks int
	reason       semconv.FlushReasonAttr
	count        int
}
