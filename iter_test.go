package goflow_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestIter(t *testing.T) {
	ctx := t.Context()

	var got []int
	for v := range stream.Of(ctx, 1, 2, 3).Iter() {
		got = append(got, v)
	}

	assert.Equal(t, []int{1, 2, 3}, got)
}

func TestIterEmpty(t *testing.T) {
	var got []int
	for v := range stream.Empty[int]().Iter() {
		got = append(got, v)
	}

	assert.Nil(t, got)
}

func TestIterBreak(t *testing.T) {
	ctx := t.Context()

	var got []int

	for v := range stream.Of(ctx, 1, 2, 3, 4, 5).Iter() {
		if v > 2 {
			break
		}

		got = append(got, v)
	}

	assert.Equal(t, []int{1, 2}, got)
}

func TestIter2(t *testing.T) {
	ctx := t.Context()

	var (
		keys []int
		vals []string
	)

	for i, v := range stream.Of(ctx, "a", "b", "c").Iter2() {
		keys = append(keys, i)
		vals = append(vals, v)
	}

	assert.Equal(t, []int{0, 1, 2}, keys)
	assert.Equal(t, []string{"a", "b", "c"}, vals)
}

func TestFromIter(t *testing.T) {
	ctx := t.Context()
	seq := slices.Values([]int{10, 20, 30})
	got := stream.FromIter(ctx, seq).Collect()
	assert.Equal(t, []int{10, 20, 30}, got)
}

func TestFromIterCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	seq := slices.Values([]int{1, 2, 3})
	got := stream.FromIter(ctx, seq).Collect()
	assert.Nil(t, got)
}

func ExampleStream_Iter() {
	ctx := context.Background()
	for v := range stream.Of(ctx, 1, 2, 3).Iter() {
		fmt.Println(v)
	}
	// Output:
	// 1
	// 2
	// 3
}

func ExampleFromIter() {
	ctx := context.Background()
	seq := slices.Values([]int{10, 20, 30})
	fmt.Println(stream.FromIter(ctx, seq).Collect())
	// Output: [10 20 30]
}
