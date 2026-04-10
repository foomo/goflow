package goflow_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var counter atomic.Int64

	got := stream.Generate(ctx, func() int64 { return counter.Add(1) }).Take(4).Collect()
	assert.Equal(t, []int64{1, 2, 3, 4}, got)
}

func TestGenerateCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got := stream.Generate(ctx, func() int { return 1 }).Collect()
	assert.Nil(t, got)
}

func ExampleGenerate() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var i int

	got := stream.Generate(ctx, func() int { i++; return i }).Take(3).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
