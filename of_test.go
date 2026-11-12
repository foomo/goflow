package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestOf(t *testing.T) {
	data := []int{1, 2, 3, 4, 4, 22, 2, 1, 4}
	assert.Equal(t, data, stream.Of(t.Context(), data...).Collect())
}

func TestOfEmpty(t *testing.T) {
	s := stream.Of[int](t.Context())
	assert.Equal(t, 0, s.Count())
}

func TestOfCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s := stream.Of(ctx, 1, 2, 3, 4, 5)
	assert.Equal(t, 0, s.Count())
}

func ExampleOf() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
