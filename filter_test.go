package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5, 6).Filter(func(_ context.Context, n int) bool {
		return n%2 == 0
	}).Collect()
	assert.Equal(t, []int{2, 4, 6}, got)
}

func TestFilterEmpty(t *testing.T) {
	got := stream.Empty[int]().Filter(func(_ context.Context, _ int) bool {
		return true
	}).Collect()
	assert.Nil(t, got)
}

func ExampleStream_Filter() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5, 6).Filter(func(_ context.Context, n int) bool {
		return n%2 == 0
	}).Collect()
	fmt.Println(got)
	// Output: [2 4 6]
}
