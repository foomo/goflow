package goflow_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestShutdownMultiStagePipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var produced atomic.Int64

	s := stream.Generate(ctx, func() int64 {
		return produced.Add(1)
	})

	doubled := stream.Map(s, func(_ context.Context, v int64) (int64, error) {
		return v * 2, nil
	})

	filtered := doubled.Filter(func(_ context.Context, v int64) bool {
		return v > 0
	})

	// Cancel after a short delay to verify partial results and clean shutdown.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	got := filtered.Collect()
	assert.NotEmpty(t, got, "should have collected some items before cancellation")
	assert.Positive(t, produced.Load(), "producer should have run")
}

func TestShutdownProcessCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var processed atomic.Int64

	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := stream.Generate(ctx, func() int {
		return 1
	}).Process(3, func(ctx context.Context, _ int) error {
		processed.Add(1)
		// Simulate work
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
			return nil
		}
	})

	// Process may or may not return an error depending on timing, but it must terminate.
	_ = err

	assert.Positive(t, processed.Load(), "some items should have been processed")
}

func TestShutdownFromFuncCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	producerDone := make(chan struct{})

	s := stream.FromFunc(ctx, 0, func(ctx context.Context, send func(int) error) error {
		defer close(producerDone)

		for i := 0; ; i++ {
			if err := send(i); err != nil {
				return err
			}
		}
	})

	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	got := s.Collect()
	assert.NotNil(t, got)

	// Verify the producer function was notified of cancellation.
	select {
	case <-producerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("producer did not terminate after context cancellation")
	}
}

func TestShutdownPreCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	// A multi-stage pipeline with a pre-cancelled context should return immediately.
	got := stream.Map(
		stream.Of(ctx, 1, 2, 3, 4, 5),
		func(_ context.Context, v int) (string, error) {
			return "x", nil
		},
	).Filter(func(_ context.Context, v string) bool {
		return true
	}).Collect()

	assert.Empty(t, got)
}

func TestShutdownTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	var count atomic.Int64

	got := stream.Generate(ctx, func() int64 {
		return count.Add(1)
	}).Collect()

	assert.NotEmpty(t, got, "should have collected items before timeout")
}
