package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestEmpty(t *testing.T) {
	s := stream.Empty[int]()
	assert.Equal(t, 0, s.Count())
}

func TestEmptySource(t *testing.T) {
	s := stream.Empty[string]()
	assert.Empty(t, s.Collect())
}

func TestOf(t *testing.T) {
	data := []int{1, 2, 3, 4, 4, 22, 2, 1, 4}
	assert.Equal(t, data, stream.Of(t.Context(), data...).Collect())
}

func TestOfEmpty(t *testing.T) {
	s := stream.Of[int](t.Context())
	assert.Equal(t, 0, s.Count())
}

func TestOfCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	s := stream.Of(ctx, 1, 2, 3, 4, 5)
	assert.Equal(t, 0, s.Count())
}

func TestFrom(t *testing.T) {
	ch := make(chan int, 3)
	ch <- 10

	ch <- 20

	ch <- 30

	close(ch)

	assert.Equal(t, []int{10, 20, 30}, stream.From(t.Context(), ch).Collect())
}

func TestCollect(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, stream.Of(t.Context(), "a", "b").Collect())
	assert.Nil(t, stream.Empty[int]().Collect())
}

func ExampleOf() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
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

func ExampleEmpty() {
	fmt.Println(stream.Empty[int]().Count())
	// Output: 0
}

func ExampleStream_Collect() {
	ctx := context.Background()
	fmt.Println(stream.Of(ctx, "a", "b", "c").Collect())
	// Output: [a b c]
}
