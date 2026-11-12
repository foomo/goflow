package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFrom(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 10

	ch <- 20

	ch <- 30

	close(ch)

	assert.Equal(t, []int{10, 20, 30}, stream.From(t.Context(), ch).Collect())
}

func ExampleFrom() {
	ctx := context.Background()

	ch := make(chan int, 3)
	ch <- 10

	ch <- 20

	ch <- 30

	close(ch)

	fmt.Println(stream.From(ctx, ch).Collect())
	// Output: [10 20 30]
}
