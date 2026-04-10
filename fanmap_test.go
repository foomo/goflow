package goflow_test

import (
	"context"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/foomo/gofuncy"
	"github.com/foomo/opentelemetry-go/exporters/glossy/glossytrace"
	oteltesting "github.com/foomo/opentelemetry-go/testing"
	"github.com/stretchr/testify/assert"
)

func TestFanMap(t *testing.T) {
	ctx := t.Context()
	got := stream.FanMap(stream.Of(ctx, 1, 2, 3, 4, 5, 6), 3, func(_ context.Context, n int) (int, error) {
		return n * 10, nil
	}).Collect()
	assert.Len(t, got, 6)
	assert.ElementsMatch(t, []int{10, 20, 30, 40, 50, 60}, got)
}

func TestFanMap_withTracing(t *testing.T) {
	ctx := t.Context()

	tp := oteltesting.ReportTraces(t, glossytrace.NewTest(t, glossytrace.WithSpanAttributes()))

	ctx, span := tp.Tracer("test").Start(ctx, "my-pipeline")
	defer span.End()

	got := stream.FanMap(stream.Of(ctx, 1, 2, 3, 4, 5, 6).WithOptions(gofuncy.WithTracerProvider(tp)), 3,
		func(_ context.Context, n int) (int, error) {
			return n * 10, nil
		},
	).Collect()

	assert.Len(t, got, 6)
	assert.ElementsMatch(t, []int{10, 20, 30, 40, 50, 60}, got)
}
