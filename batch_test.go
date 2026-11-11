package goflow_test

import (
	"context"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchMaxItems(t *testing.T) {
	got := stream.Batch(stream.Of(t.Context(), 1, 2, 3, 4, 5), stream.MaxItems(2)).Collect()
	assert.Equal(t, [][]int{{1, 2}, {3, 4}, {5}}, got)
}

func TestBatchMaxBytes(t *testing.T) {
	size := func(s string) int { return len(s) }
	got := stream.Batch(stream.Of(t.Context(), "aa", "bb", "c", "dddddd", "e"), stream.MaxBytes(4, size)).Collect()
	// "aa"+"bb" reaches 4; "c" then "dddddd" would exceed, so "c" flushes;
	// "dddddd" is larger than the limit and forms its own batch.
	assert.Equal(t, [][]string{{"aa", "bb"}, {"c"}, {"dddddd"}, {"e"}}, got)
}

func TestBatchMaxBytesTypeMismatch(t *testing.T) {
	assert.Panics(t, func() {
		stream.Batch(stream.Of(t.Context(), 1), stream.MaxBytes(4, func(string) int { return 1 }))
	})
}

func TestBatchMaxWait(t *testing.T) {
	send, s := stream.Pipe[int](t.Context(), stream.WithBuffer(10))
	batches := make(chan []int, 4)
	done := make(chan error, 1)

	go func() {
		done <- stream.Batch(s, stream.MaxItems(100), stream.MaxWait(20*time.Millisecond)).
			ForEach(func(_ context.Context, b []int) error {
				batches <- b

				return nil
			})
	}()

	require.NoError(t, send.Send(t.Context(), 1))
	require.NoError(t, send.Send(t.Context(), 2))

	select {
	case b := <-batches:
		assert.Equal(t, []int{1, 2}, b)
	case <-time.After(time.Second):
		t.Fatal("MaxWait did not flush")
	}

	require.NoError(t, send.Send(t.Context(), 3))
	require.NoError(t, send.Close(t.Context()))
	require.NoError(t, <-done)

	assert.Equal(t, []int{3}, <-batches) // partial batch flushed on close
}

func TestBatchDefault(t *testing.T) {
	got := stream.Batch(stream.Of(t.Context(), 1, 2)).Collect()
	assert.Equal(t, [][]int{{1}, {2}}, got)
}
