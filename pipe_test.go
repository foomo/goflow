package goflow_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
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

func TestPipeFunc(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	var (
		got []int
		mu  sync.Mutex
	)

	send := stream.PipeFunc(ctx, func(ctx context.Context, s stream.Stream[int]) error {
		return s.ForEach(func(_ context.Context, v int) error {
			mu.Lock()

			got = append(got, v)
			mu.Unlock()

			return nil
		})
	})

	// unbuffered pipe — each send blocks until consumed
	go func() {
		assert.NoError(t, send.Send(ctx, 1))
		assert.NoError(t, send.Send(ctx, 2))
		assert.NoError(t, send.Send(ctx, 3))
		cancel() // close pipe after all items sent and consumed
	}()

	// wait for pipe to close
	<-ctx.Done()
	time.Sleep(10 * time.Millisecond) // let ForEach finish

	mu.Lock()
	defer mu.Unlock()

	assert.Equal(t, []int{1, 2, 3}, got)
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

func TestSenderTryFull(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	send, s := stream.Pipe[int](ctx, stream.WithBuffer(1))

	require.NoError(t, send.Try(ctx, 1))
	require.ErrorIs(t, send.Try(ctx, 2), stream.ErrFull)

	cancel()
	assert.Equal(t, []int{1}, s.Collect())
	require.ErrorIs(t, send.Try(ctx, 3), stream.ErrClosed)
}

// A sender blocked on a full buffer must not block other senders.
func TestSenderBlockedSendDoesNotSerialize(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	send, s := stream.Pipe[int](ctx, stream.WithBuffer(1))
	require.NoError(t, send.Send(ctx, 1))

	blocked := make(chan error, 1)

	go func() { blocked <- send.Send(t.Context(), 2) }()

	time.Sleep(20 * time.Millisecond) // let the goroutine block on the full buffer

	done := make(chan error, 1)

	go func() { done <- send.Try(ctx, 3) }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, stream.ErrFull)
	case <-time.After(time.Second):
		t.Fatal("Try blocked behind a blocked Send")
	}

	cancel()
	require.ErrorIs(t, <-blocked, stream.ErrClosed)
	assert.Equal(t, []int{1}, s.Collect())
}

// Cancelling the parent ctx closes the entry point, but fn keeps consuming
// the queued elements, and Close waits for it.
func TestPipeFuncDrainsAfterParentCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	var processed atomic.Int64

	send := stream.PipeFunc(ctx, func(_ context.Context, s stream.Stream[int]) error {
		return s.ForEach(func(ctx context.Context, _ int) error {
			time.Sleep(time.Millisecond)

			if ctx.Err() != nil {
				return ctx.Err()
			}

			processed.Add(1)

			return nil
		})
	}, stream.WithBuffer(10))

	for i := range 10 {
		require.NoError(t, send.Send(ctx, i))
	}

	cancel()

	dctx, dcancel := context.WithTimeout(t.Context(), time.Second)
	defer dcancel()

	require.NoError(t, send.Close(dctx))
	assert.Equal(t, int64(10), processed.Load())
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

// When Close's ctx ends first, fn's ctx is cancelled and Close returns ctx.Err().
func TestPipeFuncCloseTimeout(t *testing.T) {
	fnDone := make(chan error, 1)

	send := stream.PipeFunc(t.Context(), func(ctx context.Context, s stream.Stream[int]) error {
		err := s.ForEach(func(ctx context.Context, _ int) error {
			<-ctx.Done()

			return ctx.Err()
		})
		fnDone <- err

		return err
	}, stream.WithBuffer(1))

	require.NoError(t, send.Send(t.Context(), 1))

	dctx, dcancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer dcancel()

	require.ErrorIs(t, send.Close(dctx), context.DeadlineExceeded)
	require.ErrorIs(t, <-fnDone, context.Canceled)
}

func TestPipeFuncCloseReturnsFnError(t *testing.T) {
	errBoom := errors.New("boom")

	send := stream.PipeFunc(t.Context(), func(context.Context, stream.Stream[int]) error {
		return errBoom
	})

	require.ErrorIs(t, send.Close(t.Context()), errBoom)
}

// PipeFunc on a cancelled ctx still runs fn, so Close does not hang.
func TestPipeFuncCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	send := stream.PipeFunc(ctx, func(_ context.Context, s stream.Stream[int]) error {
		return s.ForEach(func(context.Context, int) error { return nil })
	})

	dctx, dcancel := context.WithTimeout(t.Context(), time.Second)
	defer dcancel()

	require.NoError(t, send.Close(dctx))
}

// Stages that call a terminal internally (Sort, Reverse) keep all elements.
func TestPipeFuncSortKeepsElements(t *testing.T) {
	var got []int

	send := stream.PipeFunc(t.Context(), func(_ context.Context, s stream.Stream[int]) error {
		got = s.Sort(func(a, b int) int { return a - b }).Collect()

		return nil
	}, stream.WithBuffer(5))

	for _, v := range []int{3, 1, 4, 5, 2} {
		require.NoError(t, send.Send(t.Context(), v))
	}

	require.NoError(t, send.Close(t.Context()))
	assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
}

// When fn returns early, its stages are released. goleak in TestMain fails
// the run if a stage goroutine leaks.
func TestPipeFuncReleasesStages(t *testing.T) {
	send := stream.PipeFunc(t.Context(), func(_ context.Context, s stream.Stream[int]) error {
		_, _ = stream.Map(s, func(_ context.Context, v int) (int, error) { return v, nil }).FindFirst()

		return nil
	}, stream.WithBuffer(3))

	for i := range 3 {
		require.NoError(t, send.Send(t.Context(), i))
	}

	require.NoError(t, send.Close(t.Context()))
}

// Process inside PipeFunc handles every buffered element after the parent ctx ends.
func TestPipeFuncProcessDrainsBuffer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	var n atomic.Int64

	send := stream.PipeFunc(ctx, func(_ context.Context, s stream.Stream[int]) error {
		return s.Process(4, func(context.Context, int) error {
			n.Add(1)

			return nil
		})
	}, stream.WithBuffer(50))

	for i := range 50 {
		require.NoError(t, send.Send(ctx, i))
	}

	cancel()
	require.NoError(t, send.Close(t.Context()))
	assert.Equal(t, int64(50), n.Load())
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

// When fn returns early, the entry point closes: sends fail instead of
// filling the buffer and then blocking forever.
func TestPipeFuncEarlyReturnClosesEntry(t *testing.T) {
	errBoom := errors.New("boom")

	send := stream.PipeFunc(t.Context(), func(context.Context, stream.Stream[int]) error {
		return errBoom
	}, stream.WithBuffer(2))

	require.ErrorIs(t, send.Close(t.Context()), errBoom)

	for range 3 {
		err := send.Send(context.Background(), 1)
		require.ErrorIs(t, err, stream.ErrClosed)
	}
}

// Same without Close: fn returning alone must close the entry point.
func TestPipeFuncEarlyReturnWithoutClose(t *testing.T) {
	send := stream.PipeFunc(t.Context(), func(context.Context, stream.Stream[int]) error {
		return nil
	}, stream.WithBuffer(2))

	require.Eventually(t, func() bool {
		return errors.Is(send.Try(t.Context(), 1), stream.ErrClosed)
	}, time.Second, time.Millisecond)

	done := make(chan error, 1)

	go func() { done <- send.Send(context.Background(), 1) }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, stream.ErrClosed)
	case <-time.After(time.Second):
		t.Fatal("Send blocked after fn returned")
	}
}
