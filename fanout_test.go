package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFanOut(t *testing.T) {
	ctx := t.Context()
	parts := stream.Of(ctx, 1, 2, 3, 4, 5).FanOut(2)
	assert.Len(t, parts, 2)

	// consume both partitions concurrently to avoid deadlock
	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = parts[1].Collect()

		close(done)
	}()

	results[0] = parts[0].Collect()

	<-done

	assert.Equal(t, []int{1, 3, 5}, results[0])
	assert.Equal(t, []int{2, 4}, results[1])
}

func TestFanOutCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	parts := stream.Of(ctx, 1, 2, 3).FanOut(2)
	assert.Nil(t, parts)
}

func ExampleStream_FanOut() {
	ctx := context.Background()
	parts := stream.Of(ctx, 1, 2, 3, 4, 5).FanOut(2)

	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = parts[1].Collect()

		close(done)
	}()

	results[0] = parts[0].Collect()

	<-done

	fmt.Println(results[0])
	fmt.Println(results[1])
	// Output:
	// [1 3 5]
	// [2 4]
}
