package goflowconv_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"

	"github.com/foomo/goflow/semconv"
	"github.com/foomo/goflow/semconv/goflowconv"
)

func TestStageDuration(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	inst, err := goflowconv.NewStageDuration(mp.Meter("test"))
	require.NoError(t, err)

	// A caller slice with spare capacity must not be mutated.
	attrs := make([]attribute.KeyValue, 0, 4)
	attrs = append(attrs, inst.AttrErrorType(errors.New("boom")))
	inst.Record(t.Context(), 0.2, "brevo", semconv.StageNameForEach, attrs...)
	require.False(t, attrs[:cap(attrs)][len(attrs)].Valid(), "caller slice mutated")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(t.Context(), &rm))

	want := metricdata.Metrics{
		Name:        inst.Name(),
		Description: inst.Description(),
		Unit:        inst.Unit(),
		Data: metricdata.Histogram[float64]{
			Temporality: metricdata.CumulativeTemporality,
			DataPoints: []metricdata.HistogramDataPoint[float64]{{
				Attributes: attribute.NewSet(
					semconv.PipelineName("brevo"),
					semconv.StageName(semconv.StageNameForEach),
					attribute.String("error.type", "*errors.errorString"),
				),
			}},
		},
	}
	metricdatatest.AssertEqual(t, want, rm.ScopeMetrics[0].Metrics[0],
		metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue())
}

func TestMetadataMatchesCollected(t *testing.T) {
	t.Parallel()

	reader := sdkmetric.NewManualReader()
	m := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader)).Meter("test")
	ctx := t.Context()

	enqueued, err := goflowconv.NewPipeEnqueued(m)
	require.NoError(t, err)
	dropped, err := goflowconv.NewPipeDropped(m)
	require.NoError(t, err)
	batchSize, err := goflowconv.NewBatchSize(m)
	require.NoError(t, err)
	batchBody, err := goflowconv.NewBatchBodySize(m)
	require.NoError(t, err)

	enqueued.Add(ctx, 1, "p")
	dropped.Add(ctx, 1, "p", semconv.DropReasonFull)
	batchSize.Record(ctx, 3, "p", semconv.FlushReasonCount)
	batchBody.Record(ctx, 30, "p", semconv.FlushReasonCount)

	want := map[string][2]string{
		enqueued.Name():  {enqueued.Unit(), enqueued.Description()},
		dropped.Name():   {dropped.Unit(), dropped.Description()},
		batchSize.Name(): {batchSize.Unit(), batchSize.Description()},
		batchBody.Name(): {batchBody.Unit(), batchBody.Description()},
	}

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	got := map[string][2]string{}
	for _, metric := range rm.ScopeMetrics[0].Metrics {
		got[metric.Name] = [2]string{metric.Unit, metric.Description}
	}

	require.Equal(t, want, got)
}

func TestNilMeter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	enqueued, err := goflowconv.NewPipeEnqueued(nil)
	require.NoError(t, err)
	enqueued.Add(ctx, 1, "p")

	duration, err := goflowconv.NewStageDuration(nil)
	require.NoError(t, err)
	duration.Record(ctx, 1, "p", semconv.StageNameProcess)

	depth, err := goflowconv.NewPipeQueueDepth(nil)
	require.NoError(t, err)
	require.NotNil(t, depth.Inst())
}
