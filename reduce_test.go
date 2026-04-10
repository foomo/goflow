package goflow_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReduce(t *testing.T) {
	ctx := t.Context()
	sum, err := stream.Reduce(stream.Of(ctx, 1, 2, 3, 4, 5), 0, func(_ context.Context, acc, n int) (int, error) {
		return acc + n, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 15, sum)
}

func TestReduceEmpty(t *testing.T) {
	sum, err := stream.Reduce(stream.Empty[int](), 42, func(_ context.Context, acc, n int) (int, error) {
		return acc + n, nil
	})
	require.NoError(t, err)
	assert.Equal(t, 42, sum)
}

func TestReduceError(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")
	_, err := stream.Reduce(stream.Of(ctx, 1, 2, 3), 0, func(_ context.Context, acc, n int) (int, error) {
		if n == 2 {
			return 0, errBoom
		}

		return acc + n, nil
	})
	require.ErrorIs(t, err, errBoom)
}

func ExampleReduce() {
	ctx := context.Background()
	sum, _ := stream.Reduce(stream.Of(ctx, 1, 2, 3, 4, 5), 0, func(_ context.Context, acc, n int) (int, error) {
		return acc + n, nil
	})
	fmt.Println(sum)
	// Output: 15
}
