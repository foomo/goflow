package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestDistinct(t *testing.T) {
	ctx := t.Context()
	got := stream.Of(ctx, "a", "b", "a", "c", "b").Distinct(func(s string) string {
		return s
	}).Collect()
	assert.Equal(t, []string{"a", "b", "c"}, got)
}

func ExampleStream_Distinct() {
	ctx := context.Background()
	got := stream.Of(ctx, "a", "b", "a", "c", "b").Distinct(func(s string) string {
		return s
	}).Collect()
	fmt.Println(got)
	// Output: [a b c]
}
