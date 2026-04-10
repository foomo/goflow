package goflow_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestMapEach(t *testing.T) {
	ctx := t.Context()
	s1 := stream.Of(ctx, 1, 2, 3)
	s2 := stream.Of(ctx, 4, 5, 6)
	mapped := stream.MapEach([]stream.Stream[int]{s1, s2}, func(_ context.Context, n int) (string, error) {
		return strconv.Itoa(n), nil
	})
	assert.Len(t, mapped, 2)
	assert.Equal(t, []string{"1", "2", "3"}, mapped[0].Collect())
	assert.Equal(t, []string{"4", "5", "6"}, mapped[1].Collect())
}

func ExampleMapEach() {
	ctx := context.Background()
	s1 := stream.Of(ctx, 1, 2, 3)
	s2 := stream.Of(ctx, 4, 5, 6)
	mapped := stream.MapEach([]stream.Stream[int]{s1, s2}, func(_ context.Context, n int) (string, error) {
		return strconv.Itoa(n), nil
	})
	fmt.Println(mapped[0].Collect())
	fmt.Println(mapped[1].Collect())
	// Output:
	// [1 2 3]
	// [4 5 6]
}
