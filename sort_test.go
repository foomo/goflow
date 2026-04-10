package goflow_test

import (
	"cmp"
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, 3, 1, 4, 1, 5).Sort(cmp.Compare[int]).Collect()
	assert.Equal(t, []int{1, 1, 3, 4, 5}, got)
}

func ExampleStream_Sort() {
	ctx := context.Background()
	got := stream.Of(ctx, 3, 1, 4, 1, 5).Sort(cmp.Compare[int]).Collect()
	fmt.Println(got)
	// Output: [1 1 3 4 5]
}
