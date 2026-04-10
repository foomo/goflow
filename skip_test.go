package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestSkip(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Skip(2).Collect()
	assert.Equal(t, []int{3, 4, 5}, got)
}

func TestSkipAll(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3).Skip(10).Collect()
	assert.Nil(t, got)
}

func ExampleStream_Skip() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3, 4, 5).Skip(2).Collect()
	fmt.Println(got)
	// Output: [3 4 5]
}
