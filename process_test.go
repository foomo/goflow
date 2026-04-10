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

func TestProcess(t *testing.T) {
	ctx := t.Context()

	var (
		mu  sync.Mutex
		got []int
	)

	err := stream.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Process(3, func(_ context.Context, v int) error {
		mu.Lock()

		got = append(got, v)
		mu.Unlock()

		return nil
	})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, got)
}

func TestProcessError(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")
	err := stream.Of(ctx, 1, 2, 3).Process(2, func(_ context.Context, v int) error {
		if v == 2 {
			return errBoom
		}

		return nil
	})
	require.ErrorIs(t, err, errBoom)
}

func TestProcessLimit(t *testing.T) {
	ctx := t.Context()

	var (
		active atomic.Int32
		peak   atomic.Int32
	)

	err := stream.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Process(3, func(_ context.Context, _ int) error {
		cur := active.Add(1)

		for {
			old := peak.Load()
			if cur <= old || peak.CompareAndSwap(old, cur) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)
		active.Add(-1)

		return nil
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, peak.Load(), int32(3))
}
