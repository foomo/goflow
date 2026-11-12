package goflow_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFromIter(t *testing.T) {
	ctx := t.Context()
	seq := slices.Values([]int{10, 20, 30})
	got := stream.FromIter(ctx, seq).Collect()
	assert.Equal(t, []int{10, 20, 30}, got)
}

func TestFromIterCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	seq := slices.Values([]int{1, 2, 3})
	got := stream.FromIter(ctx, seq).Collect()
	assert.Nil(t, got)
}

func ExampleFromIter() {
	ctx := context.Background()
	seq := slices.Values([]int{10, 20, 30})
	fmt.Println(stream.FromIter(ctx, seq).Collect())
	// Output: [10 20 30]
}
