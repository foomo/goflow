package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFlatMap(t *testing.T) {
	ctx := t.Context()
	got := stream.FlatMap(stream.Of(ctx, 1, 2, 3), func(ctx context.Context, n int) stream.Stream[int] {
		return stream.Of(ctx, n, n*10)
	}).Collect()
	assert.Equal(t, []int{1, 10, 2, 20, 3, 30}, got)
}

func TestFlatMapEmpty(t *testing.T) {
	got := stream.FlatMap(stream.Empty[int](), func(ctx context.Context, n int) stream.Stream[int] {
		return stream.Of(ctx, n)
	}).Collect()
	assert.Nil(t, got)
}

func TestFlatMapEmptySubStreams(t *testing.T) {
	ctx := t.Context()
	got := stream.FlatMap(stream.Of(ctx, 1, 2, 3), func(_ context.Context, _ int) stream.Stream[int] {
		return stream.Empty[int]()
	}).Collect()
	assert.Nil(t, got)
}

func ExampleFlatMap() {
	ctx := context.Background()
	got := stream.FlatMap(stream.Of(ctx, 1, 2, 3), func(ctx context.Context, n int) stream.Stream[int] {
		return stream.Of(ctx, n, n*10)
	}).Collect()
	fmt.Println(got)
	// Output: [1 10 2 20 3 30]
}
