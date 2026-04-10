package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestReverse(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 1, 2, 3).Reverse().Collect()
	assert.Equal(t, []int{3, 2, 1}, got)
}

func TestReverseEmpty(t *testing.T) {
	got := stream.Empty[int]().Reverse().Collect()
	assert.Nil(t, got)
}

func ExampleStream_Reverse() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Reverse().Collect()
	fmt.Println(got)
	// Output: [3 2 1]
}
