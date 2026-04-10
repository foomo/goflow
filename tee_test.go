package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestTee(t *testing.T) {
	ctx := t.Context()
	streams := stream.Of(ctx, 1, 2, 3).Tee(2)
	assert.Len(t, streams, 2)

	// consume both concurrently to avoid deadlock
	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = streams[1].Collect()

		close(done)
	}()

	results[0] = streams[0].Collect()

	<-done

	assert.Equal(t, []int{1, 2, 3}, results[0])
	assert.Equal(t, []int{1, 2, 3}, results[1])
}

func ExampleStream_Tee() {
	ctx := context.Background()
	streams := stream.Of(ctx, 1, 2, 3).Tee(2)

	results := make([][]int, 2)
	done := make(chan struct{})

	go func() {
		results[1] = streams[1].Collect()

		close(done)
	}()

	results[0] = streams[0].Collect()

	<-done

	fmt.Println(results[0])
	fmt.Println(results[1])
	// Output:
	// [1 2 3]
	// [1 2 3]
}
