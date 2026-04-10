package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestWindow(t *testing.T) {
	ctx := t.Context()
	got := stream.Window(stream.Of(ctx, 1, 2, 3, 4, 5), 3).Collect()
	assert.Equal(t, [][]int{{1, 2, 3}, {2, 3, 4}, {3, 4, 5}}, got)
}

func TestWindowLargerThanSource(t *testing.T) {
	ctx := t.Context()
	got := stream.Window(stream.Of(ctx, 1, 2), 5).Collect()
	assert.Nil(t, got)
}

func ExampleWindow() {
	ctx := context.Background()
	got := stream.Window(stream.Of(ctx, 1, 2, 3, 4, 5), 3).Collect()
	fmt.Println(got)
	// Output: [[1 2 3] [2 3 4] [3 4 5]]
}
