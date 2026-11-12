package goflow_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	stream "github.com/foomo/goflow"
)

// Example_shutdown_timeout demonstrates using a context timeout
// to automatically stop a pipeline after a deadline.
func Example_shutdown_timeout() {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var i atomic.Int64

	results := stream.Generate(ctx, func() int64 {
		return i.Add(1)
	}).Take(1000).Collect()

	// The pipeline stops when the timeout fires.
	// Collect returns whatever was gathered before cancellation.
	fmt.Println(len(results) > 0 && len(results) <= 1000)
	// Output: true
}

// Example_shutdown_cancel demonstrates manually cancelling a pipeline.
func Example_shutdown_cancel() {
	ctx, cancel := context.WithCancel(context.Background())

	s := stream.FromFunc(ctx, 0, func(ctx context.Context, send func(int) error) error {
		for i := 1; ; i++ {
			if err := send(i); err != nil {
				return err
			}

			if i == 3 {
				cancel()

				return nil
			}
		}
	})

	results := s.Collect()
	fmt.Println(results)
	// Output: [1 2 3]
}
