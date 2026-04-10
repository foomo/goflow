package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestConcat(t *testing.T) {
	ctx := t.Context()
	got := stream.Concat(
		stream.Of(ctx, 1, 2),
		stream.Of(ctx, 3, 4),
		stream.Of(ctx, 5),
	).Collect()
	assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
}

func TestConcatEmpty(t *testing.T) {
	got := stream.Concat[int]().Collect()
	assert.Nil(t, got)
}

func TestConcatSingle(t *testing.T) {
	ctx := t.Context()
	got := stream.Concat(stream.Of(ctx, 1, 2, 3)).Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestConcatWithEmpty(t *testing.T) {
	ctx := t.Context()
	got := stream.Concat(
		stream.Of(ctx, 1),
		stream.Empty[int](),
		stream.Of(ctx, 2),
	).Collect()
	assert.Equal(t, []int{1, 2}, got)
}

func ExampleConcat() {
	ctx := context.Background()
	got := stream.Concat(
		stream.Of(ctx, 1, 2),
		stream.Of(ctx, 3, 4),
	).Collect()
	fmt.Println(got)
	// Output: [1 2 3 4]
}
