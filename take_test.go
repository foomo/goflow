package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestTake(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Take(3).Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestTakeMoreThanSource(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2).Take(10).Collect()
	assert.Equal(t, []int{1, 2}, got)
}

func ExampleStream_Take() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Take(3).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
