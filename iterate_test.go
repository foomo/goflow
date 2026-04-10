package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestIterate(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	got := stream.Iterate(ctx, 1, func(n int) int { return n * 2 }).Take(5).Collect()
	assert.Equal(t, []int{1, 2, 4, 8, 16}, got)
}

func TestIterateCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got := stream.Iterate(ctx, 1, func(n int) int { return n + 1 }).Collect()
	assert.Nil(t, got)
}

func ExampleIterate() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := stream.Iterate(ctx, 1, func(n int) int { return n * 2 }).Take(5).Collect()
	fmt.Println(got)
	// Output: [1 2 4 8 16]
}
