package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	ctx := t.Context()
	got := stream.Split(stream.Of(ctx, 1, 2, 3, 4, 5), 2).Collect()
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, got)
}

func TestSplitSingleBatch(t *testing.T) {
	ctx := t.Context()
	got := stream.Split(stream.Of(ctx, 1, 2), 5).Collect()
	assert.Equal(t, [][]int{{1, 2}}, got)
}

func ExampleSplit() {
	ctx := context.Background()
	got := stream.Split(stream.Of(ctx, 1, 2, 3, 4, 5), 2).Collect()
	fmt.Println(got)
	// Output: [[1 2] [3 4] [5]]
}
