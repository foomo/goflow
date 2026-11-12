package goflow_test

import (
	"context"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
