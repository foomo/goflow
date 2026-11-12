package goflow_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
