package goflow_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipe(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	send, s := stream.Pipe[int](ctx)

	go func() {
		assert.NoError(t, send.Send(ctx, 1))
		assert.NoError(t, send.Send(ctx, 2))
		assert.NoError(t, send.Send(ctx, 3))
		cancel()
	}()

	got := s.Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestPipeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	send, _ := stream.Pipe[int](ctx)
	err := send.Send(t.Context(), 1)
	require.ErrorIs(t, err, stream.ErrClosed)
	require.ErrorIs(t, err, context.Canceled, "closing by ctx keeps the ctx cause")
	require.ErrorIs(t, send.Try(t.Context(), 1), context.Canceled)
}

func TestPipeBuffered(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	send, s := stream.Pipe[int](ctx, stream.WithBuffer(3))

	// These should not block since buffer is 3
	assert.NoError(t, send.Send(ctx, 10))
	assert.NoError(t, send.Send(ctx, 20))
	assert.NoError(t, send.Send(ctx, 30))
	cancel()

	got := s.Collect()
	assert.Equal(t, []int{10, 20, 30}, got)
}

func ExamplePipe() {
	ctx, cancel := context.WithCancel(context.Background())
	send, s := stream.Pipe[int](ctx)

	go func() {
		_ = send.Send(ctx, 1)
		_ = send.Send(ctx, 2)
		_ = send.Send(ctx, 3)

		cancel()
	}()

	fmt.Println(s.Collect())
	// Output: [1 2 3]
}

// A pipe closed by its ctx reports both ErrClosed and the ctx cause.
func TestPipeClosedByCtxKeepsCause(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	send, _ := stream.Pipe[int](ctx) // unbuffered: Try never queues without a reader

	cancel()

	// The ctx closes the entry point asynchronously.
	require.Eventually(t, func() bool {
		return errors.Is(send.Try(t.Context(), 0), stream.ErrClosed)
	}, time.Second, time.Millisecond)

	err := send.Send(t.Context(), 1)
	require.ErrorIs(t, err, stream.ErrClosed)
	require.ErrorIs(t, err, context.Canceled)
}

// A plain Pipe stops its stages with ctx. goleak in TestMain fails the run
// if a stage goroutine leaks.
func TestPipeStopsWithCtx(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	send, s := stream.Pipe[int](ctx, stream.WithBuffer(3))
	for i := range 3 {
		require.NoError(t, send.Send(ctx, i))
	}

	_, found := stream.Map(s, func(_ context.Context, v int) (int, error) { return v, nil }).FindFirst()
	require.True(t, found)

	cancel()
}

// Buffered elements of a closed Pipe are still read by terminals.
func TestPipeCollectAfterClose(t *testing.T) {
	send, s := stream.Pipe[int](t.Context(), stream.WithBuffer(3))
	for i := range 3 {
		require.NoError(t, send.Send(t.Context(), i))
	}

	require.NoError(t, send.Close(t.Context()))
	assert.Equal(t, []int{0, 1, 2}, slices.Collect(s.Iter()))
}

// Chan on a closed pipe stream must close, not hang.
func TestPipeChanAfterClose(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	send, s := stream.Pipe[int](ctx)
	require.NoError(t, send.Close(ctx))
	cancel()

	select {
	case _, ok := <-s.Chan():
		assert.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("Chan did not close")
	}
}

// Stage functions see the parent's deadline as DeadlineExceeded, not Canceled.
func TestStageKeepsDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)

	_ = stream.Map(stream.Of(ctx, 1), func(ctx context.Context, v int) (int, error) {
		<-ctx.Done()

		errCh <- ctx.Err()

		return v, nil
	}).Collect()

	require.ErrorIs(t, <-errCh, context.DeadlineExceeded)
}
