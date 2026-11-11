package goflow_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMap(t *testing.T) {
	ctx := t.Context()
	got := stream.Map(stream.Of(ctx, 1, 2, 3), func(_ context.Context, n int) (string, error) {
		return strconv.Itoa(n), nil
	}).Collect()
	assert.Equal(t, []string{"1", "2", "3"}, got)
}

func TestMapSameType(t *testing.T) {
	ctx := t.Context()
	got := stream.Map(stream.Of(ctx, 1, 2, 3), func(_ context.Context, n int) (int, error) { return n * 2, nil }).Collect()
	assert.Equal(t, []int{2, 4, 6}, got)
}

func TestMapChain(t *testing.T) {
	s := stream.Of(t.Context(), 1, 2, 3)
	doubled := stream.Map(s, func(_ context.Context, n int) (int, error) { return n * 2, nil })
	strs := stream.Map(doubled, func(_ context.Context, n int) (string, error) { return strconv.Itoa(n), nil })
	assert.Equal(t, []string{"2", "4", "6"}, strs.Collect())
}

func TestMapError(t *testing.T) {
	ctx := t.Context()

	var gotErr atomic.Value

	errBoom := errors.New("boom")
	got := stream.Map(stream.Of(ctx, 1, 2, 3, 4, 5).WithOptions(gofuncy.WithErrorHandler(func(_ context.Context, err error) {
		gotErr.Store(err)
	})), func(_ context.Context, n int) (int, error) {
		if n == 3 {
			return 0, errBoom
		}

		return n * 10, nil
	}).Collect()
	// Stream closes on error — we get elements before the error
	assert.Less(t, len(got), 5)
	// gofuncy calls the error handler after the stage closed its output.
	require.Eventually(t, func() bool { return gotErr.Load() != nil }, time.Second, time.Millisecond)
	assert.ErrorIs(t, gotErr.Load().(error), errBoom)
}

func TestMapCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s := stream.Map(stream.Of(ctx, 1, 2, 3), func(_ context.Context, n int) (string, error) { return strconv.Itoa(n), nil })
	assert.Equal(t, 0, s.Count())
}

func ExampleMap() {
	ctx := context.Background()
	got := stream.Map(stream.Of(ctx, 1, 2, 3), func(_ context.Context, n int) (string, error) {
		return strconv.Itoa(n), nil
	}).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
