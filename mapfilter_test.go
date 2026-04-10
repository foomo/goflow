package goflow_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/foomo/gofuncy"
	"github.com/stretchr/testify/assert"
)

func TestMapFilter(t *testing.T) {
	ctx := t.Context()
	got := stream.MapFilter(stream.Of(ctx, 1, 2, 3, 4, 5), func(_ context.Context, n int) (string, bool, error) {
		if n%2 == 0 {
			return "", false, nil // skip evens
		}

		return strconv.Itoa(n), true, nil
	}).Collect()
	assert.Equal(t, []string{"1", "3", "5"}, got)
}

func TestMapFilterDeadLetter(t *testing.T) {
	ctx := t.Context()

	var (
		deadLettered []int
		mu           sync.Mutex
	)

	got := stream.MapFilter(stream.Of(ctx, 1, 2, 3, 4, 5), func(_ context.Context, n int) (int, bool, error) {
		if n == 3 {
			mu.Lock()

			deadLettered = append(deadLettered, n)
			mu.Unlock()

			return 0, false, nil // dead letter, skip
		}

		return n * 10, true, nil
	}).Collect()
	assert.Equal(t, []int{10, 20, 40, 50}, got)
	assert.Equal(t, []int{3}, deadLettered)
}

func TestMapFilterFatalError(t *testing.T) {
	ctx := t.Context()
	got := stream.MapFilter(stream.Of(ctx, 1, 2, 3, 4, 5).WithOptions(gofuncy.WithErrorHandler(func(_ context.Context, _ error) {})), func(_ context.Context, n int) (int, bool, error) {
		if n == 3 {
			return 0, false, errors.New("fatal")
		}

		return n, true, nil
	}).Collect()
	assert.Less(t, len(got), 5)
}

func ExampleMapFilter() {
	ctx := context.Background()
	got := stream.MapFilter(stream.Of(ctx, 1, 2, 3, 4, 5), func(_ context.Context, n int) (string, bool, error) {
		if n%2 == 0 {
			return "", false, nil
		}

		return strconv.Itoa(n), true, nil
	}).Collect()
	fmt.Println(got)
	// Output: [1 3 5]
}
