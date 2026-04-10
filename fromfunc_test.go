package goflow_test

import (
	"context"
	"fmt"

	stream "github.com/foomo/goflow"
)

// ExampleFromFunc demonstrates creating a stream from a blocking producer function.
func ExampleFromFunc() {
	ctx, cancel := context.WithCancel(context.Background())

	s := stream.FromFunc(ctx, 0, func(ctx context.Context, send func(string) error) error {
		for _, v := range []string{"alpha", "beta", "gamma"} {
			if err := send(v); err != nil {
				return err
			}
		}

		cancel()

		return nil
	})

	for _, v := range s.Collect() {
		fmt.Println(v)
	}
	// Output:
	// alpha
	// beta
	// gamma
}
