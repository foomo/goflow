package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFlatten(t *testing.T) {
	ctx := t.Context()
	chunked := stream.Split(stream.Of(ctx, 1, 2, 3, 4, 5), 2)
	got := stream.Flatten(chunked).Collect()
	assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
}

func TestFlattenEmpty(t *testing.T) {
	got := stream.Flatten(stream.Empty[[]int]()).Collect()
	assert.Nil(t, got)
}

func ExampleFlatten() {
	ctx := context.Background()
	chunked := stream.Split(stream.Of(ctx, 1, 2, 3, 4, 5), 2)
	got := stream.Flatten(chunked).Collect()
	fmt.Println(got)
	// Output: [1 2 3 4 5]
}
