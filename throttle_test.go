package goflow_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestThrottle(t *testing.T) {
	ctx := t.Context()
	start := time.Now()
	got := stream.Of(ctx, 1, 2, 3).Throttle(20 * time.Millisecond).Collect()
	elapsed := time.Since(start)

	assert.Equal(t, []int{1, 2, 3}, got)
	// 3 items with 20ms throttle: first immediate, then 2 waits = ~40ms minimum
	assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
}

func ExampleStream_Throttle() {
	ctx := context.Background()
	got := stream.Of(ctx, 1, 2, 3).Throttle(1 * time.Millisecond).Collect()
	fmt.Println(got)
	// Output: [1 2 3]
}
