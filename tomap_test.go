package goflow_test

import (
	"context"
	"fmt"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestToMap(t *testing.T) {
	ctx := t.Context()
	got := stream.ToMap(
		stream.Of(ctx, "a", "bb", "ccc"),
		func(s string) string { return s },
		func(s string) int { return len(s) },
	)
	assert.Equal(t, map[string]int{"a": 1, "bb": 2, "ccc": 3}, got)
}

func TestToMapEmpty(t *testing.T) {
	got := stream.ToMap(
		stream.Empty[string](),
		func(s string) string { return s },
		func(s string) int { return len(s) },
	)
	assert.Empty(t, got)
}

func TestToMapDuplicateKeys(t *testing.T) {
	ctx := t.Context()
	got := stream.ToMap(
		stream.Of(ctx, "a", "ab", "ac"),
		func(s string) byte { return s[0] },
		func(s string) string { return s },
	)
	// Last value wins for key 'a'
	assert.Equal(t, "ac", got['a'])
}

func ExampleToMap() {
	ctx := context.Background()
	got := stream.ToMap(
		stream.Of(ctx, "a", "bb"),
		func(s string) string { return s },
		func(s string) int { return len(s) },
	)
	fmt.Println(got["a"], got["bb"])
	// Output: 1 2
}
