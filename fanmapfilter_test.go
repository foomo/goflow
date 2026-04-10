package goflow_test

import (
	"context"
	"fmt"
	"strconv"

	stream "github.com/foomo/goflow"
)

func ExampleFanMapFilter() {
	ctx := context.Background()
	got := stream.FanMapFilter(stream.Of(ctx, 1, 2, 3, 4, 5, 6), 2, func(_ context.Context, n int) (string, bool, error) {
		if n%2 == 0 {
			return "", false, nil
		}

		return strconv.Itoa(n * 10), true, nil
	}).Collect()
	fmt.Println(len(got))
	// Output: 3
}
