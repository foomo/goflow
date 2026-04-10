package goflow_test

import (
	"context"
	"fmt"
	"strconv"

	stream "github.com/foomo/goflow"
)

func ExampleMapFilterEach() {
	ctx := context.Background()
	s1 := stream.Of(ctx, 1, 2, 3)
	s2 := stream.Of(ctx, 4, 5, 6)
	mapped := stream.MapFilterEach([]stream.Stream[int]{s1, s2}, func(_ context.Context, n int) (string, bool, error) {
		if n%2 == 0 {
			return "", false, nil
		}

		return strconv.Itoa(n), true, nil
	})
	fmt.Println(mapped[0].Collect())
	fmt.Println(mapped[1].Collect())
	// Output:
	// [1 3]
	// [5]
}
