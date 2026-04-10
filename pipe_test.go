package goflow_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestPipe(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	send, s := stream.Pipe[int](ctx)

	go func() {
		assert.NoError(t, send(ctx, 1))
		assert.NoError(t, send(ctx, 2))
		assert.NoError(t, send(ctx, 3))
		cancel()
	}()

	got := s.Collect()
	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestPipeCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	send, _ := stream.Pipe[int](ctx)
	assert.ErrorIs(t, send(ctx, 1), context.Canceled)
}

func TestPipeBuffered(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	send, s := stream.Pipe[int](ctx, 3)

	// These should not block since buffer is 3
	assert.NoError(t, send(ctx, 10))
	assert.NoError(t, send(ctx, 20))
	assert.NoError(t, send(ctx, 30))
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
		assert.NoError(t, send(ctx, 1))
		assert.NoError(t, send(ctx, 2))
		assert.NoError(t, send(ctx, 3))
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
		_ = send(ctx, 1)
		_ = send(ctx, 2)
		_ = send(ctx, 3)

		cancel()
	}()

	fmt.Println(s.Collect())
	// Output: [1 2 3]
}
