package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestAllMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 2, 4, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	assert.False(t, stream.Of(ctx, 2, 3, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
}

func TestAllMatchEmpty(t *testing.T) {
	assert.True(t, stream.Empty[int]().AllMatch(func(_ context.Context, _ int) bool { return false }))
}

func TestAnyMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 1, 2, 3).AnyMatch(func(_ context.Context, n int) bool { return n == 2 }))
	assert.False(t, stream.Of(ctx, 1, 3, 5).AnyMatch(func(_ context.Context, n int) bool { return n == 2 }))
}

func TestAnyMatchEmpty(t *testing.T) {
	assert.False(t, stream.Empty[int]().AnyMatch(func(_ context.Context, _ int) bool { return true }))
}

func TestNoneMatch(t *testing.T) {
	ctx := t.Context()
	assert.True(t, stream.Of(ctx, 1, 3, 5).NoneMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	assert.False(t, stream.Of(ctx, 1, 2, 5).NoneMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
}

func TestNoneMatchEmpty(t *testing.T) {
	assert.True(t, stream.Empty[int]().NoneMatch(func(_ context.Context, _ int) bool { return true }))
}

func ExampleStream_AllMatch() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, 2, 4, 6).AllMatch(func(_ context.Context, n int) bool { return n%2 == 0 }))
	// Output: true
}

func ExampleStream_AnyMatch() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, 1, 2, 3).AnyMatch(func(_ context.Context, n int) bool { return n > 2 }))
	// Output: true
}
