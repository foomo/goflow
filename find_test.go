package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFindFirst(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 10, 20, 30).FindFirst()
	assert.True(t, ok)
	assert.Equal(t, 10, v)
}

func TestFindFirstEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().FindFirst()
	assert.False(t, ok)
	assert.Zero(t, v)
}

func TestFindFirstMatch(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 1, 2, 3, 4).FindFirstMatch(func(_ context.Context, n int) bool { return n > 2 })
	assert.True(t, ok)
	assert.Equal(t, 3, v)
}

func TestFindFirstMatchNone(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 1, 2, 3).FindFirstMatch(func(_ context.Context, n int) bool { return n > 10 })
	assert.False(t, ok)
	assert.Zero(t, v)
}

func TestFindFirstMatchEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().FindFirstMatch(func(_ context.Context, _ int) bool { return true })
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_FindFirst() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 10, 20, 30).FindFirst()
	fmt.Println(v, ok)
	// Output: 10 true
}

func ExampleStream_FindFirstMatch() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 1, 2, 3, 4).FindFirstMatch(func(_ context.Context, n int) bool { return n > 2 })
	fmt.Println(v, ok)
	// Output: 3 true
}
