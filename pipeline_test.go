package goflow_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
)

func TestForEachErrorHandler(t *testing.T) {
	errBoom := errors.New("boom")

	var handled atomic.Int64

	err := stream.Of(t.Context(), 1, 2, 3).ForEach(func(_ context.Context, v int) error {
		if v != 2 {
			return errBoom
		}

		return nil
	}, stream.WithErrorHandler(func(_ context.Context, err error) {
		require.ErrorIs(t, err, errBoom)
		handled.Add(1)
	}))
	require.NoError(t, err)
	assert.Equal(t, int64(2), handled.Load())
}

func TestProcessErrorHandler(t *testing.T) {
	errBoom := errors.New("boom")

	var handled atomic.Int64

	err := stream.Of(t.Context(), 1, 2, 3, 4).Process(2, func(context.Context, int) error {
		return errBoom
	}, stream.WithErrorHandler(func(context.Context, error) { handled.Add(1) }))
	require.NoError(t, err)
	assert.Equal(t, int64(4), handled.Load())
}

// TestBrevoShape runs the target long-lived pipeline shape end to end:
// many request-scoped senders, batching, a draining shutdown, metrics and span links.
func TestBrevoShape(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	ctx, cancel := context.WithCancel(t.Context())

	var (
		mu     sync.Mutex
		posted []int
		sizes  []int
	)

	send := stream.PipeFunc(ctx, func(_ context.Context, events stream.Stream[int]) error {
		return stream.Batch(events,
			stream.MaxItems(20),
			stream.MaxBytes(150, func(int) int { return 10 }),
			stream.MaxWait(50*time.Millisecond),
		).ForEach(func(_ context.Context, b []int) error {
			mu.Lock()
			defer mu.Unlock()

			posted = append(posted, b...)
			sizes = append(sizes, len(b))

			return nil
		}, stream.WithErrorHandler(func(context.Context, error) {}))
	},
		stream.WithBuffer(1000),
		stream.WithName("brevo"),
		stream.WithMeterProvider(mp),
		stream.WithTracerProvider(tp),
	)

	const senders, perSender = 50, 10

	var wg sync.WaitGroup
	for range senders {
		wg.Go(func() {
			rctx, span := tp.Tracer("test").Start(ctx, "request")
			defer span.End()

			for i := range perSender {
				assert.NoError(t, send.Try(rctx, i))
			}
		})
	}

	wg.Wait()
	cancel() // service ctx ends; queued items must still be posted

	dctx, dcancel := context.WithTimeout(t.Context(), time.Second)
	defer dcancel()

	require.NoError(t, send.Close(dctx))

	assert.Len(t, posted, senders*perSender)

	for _, n := range sizes {
		assert.LessOrEqual(t, n, 15, "MaxBytes 150 / 10 per item caps a batch at 15")
	}

	var batchSpans int

	for _, s := range recorder.Ended() {
		if s.Name() == "process brevo" {
			batchSpans++

			assert.Equal(t, trace.SpanKindConsumer, s.SpanKind())
			assert.False(t, s.Parent().IsValid(), "batch span must be a new root")
			assert.NotEmpty(t, s.Links())
			assert.LessOrEqual(t, len(s.Links()), 128)
			assert.Contains(t, s.Attributes(), semconv.PipelineName("brevo"))
			assert.Contains(t, s.Attributes(), semconv.StageName(semconv.StageNameForEach))
		}
	}

	assert.Equal(t, len(sizes), batchSpans, "one span per batch, none per item")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	counts := map[string]int64{}

	for _, sm := range rm.ScopeMetrics {
		assert.Equal(t, stream.ScopeName, sm.Scope.Name)

		for _, m := range sm.Metrics {
			switch d := m.Data.(type) {
			case metricdata.Sum[int64]:
				for _, dp := range d.DataPoints {
					v, _ := dp.Attributes.Value(semconv.PipelineNameKey)
					assert.Equal(t, "brevo", v.AsString())

					counts[m.Name] += dp.Value
				}
			case metricdata.Gauge[int64]:
				t.Fatalf("%s must be an up-down counter, not a gauge", m.Name)
			case metricdata.Histogram[float64]:
				for _, dp := range d.DataPoints {
					counts[m.Name] += int64(dp.Count) //nolint:gosec // test counts are small
				}
			case metricdata.Histogram[int64]:
				for _, dp := range d.DataPoints {
					counts[m.Name] += int64(dp.Count) //nolint:gosec // test counts are small
				}
			}
		}
	}

	assert.Equal(t, int64(senders*perSender), counts["goflow.pipe.enqueued"])
	assert.Equal(t, int64(len(sizes)), counts["goflow.stage.duration"])
	assert.Equal(t, int64(len(sizes)), counts["goflow.batch.size"])
	assert.Equal(t, int64(len(sizes)), counts["goflow.batch.body.size"])
}

// Stage goroutines of a Pipe pipeline run for its whole life, so their
// gofuncy spans are off unless WithStageTracing is set.
func TestPipeStageTracing(t *testing.T) {
	for name, tc := range map[string]struct {
		opts      []stream.Option
		wantStage bool
	}{
		"default":            {wantStage: false},
		"with stage tracing": {opts: []stream.Option{stream.WithStageTracing()}, wantStage: true},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

			send := stream.PipeFunc(t.Context(), func(_ context.Context, s stream.Stream[int]) error {
				s = s.WithOptions(gofuncy.WithTracerProvider(tp))

				return stream.Batch(s, stream.MaxItems(1)).Process(1, func(context.Context, []int) error { return nil })
			}, append(tc.opts, stream.WithBuffer(2), stream.WithTracerProvider(tp))...)

			pctx, pspan := tp.Tracer("test").Start(t.Context(), "producer")
			require.NoError(t, send.Send(pctx, 1))
			pspan.End()
			require.NoError(t, send.Close(t.Context()))

			count := func() (int, int) {
				var stage, consumer int

				for _, span := range recorder.Ended() {
					switch span.Name() {
					case "process goflow.pipe":
						consumer++
					case "producer":
					default:
						stage++
					}
				}

				return stage, consumer
			}

			// Stage spans end when their goroutines exit, just after fn returns.
			if tc.wantStage {
				require.Eventually(t, func() bool { stage, _ := count(); return stage > 0 }, time.Second, time.Millisecond)
			} else {
				time.Sleep(20 * time.Millisecond)
			}

			stage, consumer := count()
			assert.Equal(t, 1, consumer, "one consumer span per traced element")
			assert.Equal(t, tc.wantStage, stage > 0)
		})
	}
}

func TestPipeQueueGauges(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	send, _ := stream.Pipe[int](t.Context(), stream.WithBuffer(5), stream.WithName("q"), stream.WithMeterProvider(mp))
	require.NoError(t, send.Send(t.Context(), 1))
	require.NoError(t, send.Send(t.Context(), 2))

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	got := map[string]int64{}

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if d, ok := m.Data.(metricdata.Sum[int64]); ok && !d.IsMonotonic {
				for _, dp := range d.DataPoints {
					got[m.Name] = dp.Value
				}
			}
		}
	}

	assert.Equal(t, map[string]int64{"goflow.pipe.queue.depth": 2, "goflow.pipe.queue.capacity": 5}, got)
	require.NoError(t, send.Close(t.Context()))

	// Gauges are removed once the entry point closed.
	rm = metricdata.ResourceMetrics{}
	require.NoError(t, reader.Collect(t.Context(), &rm))

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if d, ok := m.Data.(metricdata.Sum[int64]); ok && !d.IsMonotonic {
				assert.Empty(t, d.DataPoints, m.Name)
			}
		}
	}
}

// A consumer span exists to link the origin spans of its elements, so an
// element sent from a ctx without a span gets no consumer span.
func TestConsumerSpanNeedsOrigin(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	var seen atomic.Int32

	send := stream.PipeFunc(t.Context(), func(_ context.Context, s stream.Stream[int]) error {
		return s.ForEach(func(context.Context, int) error {
			seen.Add(1)

			return nil
		})
	}, stream.WithTracerProvider(tp))

	require.NoError(t, send.Send(t.Context(), 1))

	pctx, pspan := tp.Tracer("test").Start(t.Context(), "producer")
	require.NoError(t, send.Send(pctx, 2))
	pspan.End()

	require.NoError(t, send.Close(t.Context()))
	assert.Equal(t, int32(2), seen.Load())

	var consumer int

	for _, span := range recorder.Ended() {
		if span.Name() == "process goflow.pipe" {
			consumer++

			assert.Len(t, span.Links(), 1)
		}
	}

	assert.Equal(t, 1, consumer, "only the traced element gets a consumer span")
}

// A failed stage records its duration with the error.type attribute.
func TestStageDurationErrorType(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	send, s := stream.Pipe[int](t.Context(), stream.WithBuffer(1), stream.WithName("q"), stream.WithMeterProvider(mp))
	require.NoError(t, send.Send(t.Context(), 1))
	require.NoError(t, send.Close(t.Context()))

	err := s.ForEach(func(context.Context, int) error { return errors.New("boom") },
		stream.WithErrorHandler(func(context.Context, error) {}))
	require.NoError(t, err)

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	var found bool

	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != "goflow.stage.duration" {
				continue
			}

			for _, dp := range m.Data.(metricdata.Histogram[float64]).DataPoints {
				if _, ok := dp.Attributes.Value("error.type"); ok {
					found = true
				}
			}
		}
	}

	assert.True(t, found, "no goflow.stage.duration point with error.type")
}
