package goflow_test

import (
	"context"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestOfMap(t *testing.T) {
	ctx := t.Context()
	m := map[string]int{"a": 1, "b": 2}
	got := stream.OfMap(ctx, m).Collect()
	assert.Len(t, got, 2)
	assert.Contains(t, got, stream.KeyValue[string, int]{Key: "a", Value: 1})
	assert.Contains(t, got, stream.KeyValue[string, int]{Key: "b", Value: 2})
}

func TestOfMapEmpty(t *testing.T) {
	ctx := t.Context()
	got := stream.OfMap(ctx, map[string]int{}).Collect()
	assert.Nil(t, got)
}

func TestOfMapCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	got := stream.OfMap(ctx, map[string]int{"a": 1}).Collect()
	assert.Nil(t, got)
}

func TestOfMapToMap(t *testing.T) {
	ctx := t.Context()
	m := map[string]int{"x": 10, "y": 20}
	got := stream.ToMap(
		stream.OfMap(ctx, m),
		func(kv stream.KeyValue[string, int]) string { return kv.Key },
		func(kv stream.KeyValue[string, int]) int { return kv.Value },
	)
	assert.Equal(t, m, got)
}
