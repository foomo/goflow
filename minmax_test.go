package goflow_test

import (
	"cmp"
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Min(cmp.Compare[int])
	assert.True(t, ok)
	assert.Equal(t, 1, v)
}

func TestMinEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().Min(cmp.Compare[int])
	assert.False(t, ok)
	assert.Zero(t, v)
}

func TestMax(t *testing.T) {
	ctx := t.Context()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Max(cmp.Compare[int])
	assert.True(t, ok)
	assert.Equal(t, 5, v)
}

func TestMaxEmpty(t *testing.T) {
	v, ok := stream.Empty[int]().Max(cmp.Compare[int])
	assert.False(t, ok)
	assert.Zero(t, v)
}

func ExampleStream_Min() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Min(cmp.Compare[int])
	fmt.Println(v, ok)
	// Output: 1 true
}

func ExampleStream_Max() {
	ctx := context.Background()
	v, ok := stream.Of(ctx, 3, 1, 4, 1, 5).Max(cmp.Compare[int])
	fmt.Println(v, ok)
	// Output: 5 true
}
