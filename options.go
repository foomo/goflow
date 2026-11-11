package goflow

import (
	"time"

	"github.com/foomo/gofuncy"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Option configures [Pipe], [PipeFunc], [Batch], [Stream.ForEach] and
// [Stream.Process]. Each function ignores options that do not apply to it.
type Option func(*config)

// ErrorHandler receives per-element errors from [Stream.ForEach] and
// [Stream.Process]. It is the gofuncy handler type, so one value serves both
// libraries.
type ErrorHandler = gofuncy.ErrorHandler

type config struct {
	name           string
	buffer         int
	meterProvider  metric.MeterProvider
	tracerProvider trace.TracerProvider
	maxItems       int
	maxBytes       int
	sizeFn         any // func(T) int, checked by Batch
	maxWait        time.Duration
	errorHandler   ErrorHandler
	groupOpts      []gofuncy.GroupOption
	stageTracing   bool
}

func newConfig(opts []Option) config {
	var c config

	for _, opt := range opts {
		if opt != nil {
			opt(&c)
		}
	}

	return c
}

// WithName sets the pipeline name used as the goflow.pipeline.name attribute
// and in the consumer span name "process <name>"; it must be a static label
// ([Pipe], [PipeFunc]).
func WithName(name string) Option {
	return func(c *config) { c.name = name }
}

// WithBuffer sets the entry point channel capacity ([Pipe], [PipeFunc]).
func WithBuffer(n int) Option {
	return func(c *config) { c.buffer = n }
}

// WithMeterProvider sets the meter provider for pipeline metrics ([Pipe],
// [PipeFunc]). Defaults to the global otel meter provider.
func WithMeterProvider(mp metric.MeterProvider) Option {
	return func(c *config) { c.meterProvider = mp }
}

// WithTracerProvider sets the tracer provider for linked consumer spans
// ([Pipe], [PipeFunc]). Defaults to the global otel tracer provider.
func WithTracerProvider(tp trace.TracerProvider) Option {
	return func(c *config) { c.tracerProvider = tp }
}

// MaxItems flushes a batch when it holds n elements ([Batch]).
func MaxItems(n int) Option {
	return func(c *config) { c.maxItems = n }
}

// MaxBytes flushes a batch before its summed element size would exceed n
// ([Batch]). An element larger than n forms a batch of its own. The type T must
// match the stream element type, otherwise Batch panics.
func MaxBytes[T any](n int, size func(T) int) Option {
	return func(c *config) {
		c.maxBytes = n
		c.sizeFn = size
	}
}

// MaxWait flushes a non-empty batch d after its first element arrived
// ([Batch]).
func MaxWait(d time.Duration) Option {
	return func(c *config) { c.maxWait = d }
}

// WithErrorHandler hands per-element errors to h and keeps consuming
// ([Stream.ForEach], [Stream.Process]). Errors are not retained.
func WithErrorHandler(h ErrorHandler) Option {
	return func(c *config) { c.errorHandler = h }
}

// WithGroupOptions passes options to the [gofuncy.Group] that runs the workers
// ([Stream.Process]).
func WithGroupOptions(opts ...gofuncy.GroupOption) Option {
	return func(c *config) { c.groupOpts = append(c.groupOpts, opts...) }
}

// WithStageTracing keeps the gofuncy goroutine spans of the pipeline stages
// ([Pipe], [PipeFunc]).
//
// By default a Pipe pipeline passes [gofuncy.WithoutTracing] to its stages:
// they run for the life of the pipeline, so their spans would only end at
// shutdown. gofuncy metrics and the goflow consumer spans are not affected.
func WithStageTracing() Option {
	return func(c *config) { c.stageTracing = true }
}
