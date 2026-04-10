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

func TestForEach(t *testing.T) {
	ctx := t.Context()

	var got []int

	err := stream.Of(ctx, 1, 2, 3).ForEach(func(_ context.Context, v int) error {
		got = append(got, v)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestForEachError(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")

	var count int

	err := stream.Of(ctx, 1, 2, 3, 4, 5).ForEach(func(_ context.Context, _ int) error {
		count++
		if count == 3 {
			return errBoom
		}

		return nil
	})
	require.ErrorIs(t, err, errBoom)
	assert.Equal(t, 3, count)
}

func TestForEachEmpty(t *testing.T) {
	err := stream.Empty[int]().ForEach(func(_ context.Context, _ int) error {
		t.Fatal("should not be called")
		return nil
	})
	require.NoError(t, err)
}

func ExampleStream_ForEach() {
	ctx := context.Background()
	_ = stream.Of(ctx, 1, 2, 3).ForEach(func(_ context.Context, v int) error {
		fmt.Println(v)
		return nil
	})
	// Output:
	// 1
	// 2
	// 3
}
