package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestPeek(t *testing.T) {
	ctx := t.Context()

	var peeked []int

	got := stream.Of(ctx, 1, 2, 3).Peek(func(_ context.Context, n int) {
		peeked = append(peeked, n)
	}).Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
	assert.Equal(t, []int{1, 2, 3}, peeked)
}

func ExampleStream_Peek() {
	ctx := context.Background()

	var peeked []int

	got := stream.Of(ctx, 1, 2, 3).Peek(func(_ context.Context, n int) {
		peeked = append(peeked, n)
	}).Collect()
	fmt.Println(got)
	fmt.Println(peeked)
	// Output:
	// [1 2 3]
	// [1 2 3]
}
