package goflow_test

import (
	"testing"

	stream "github.com/foomo/goflow"
	"github.com/stretchr/testify/assert"
)

func TestFanIn(t *testing.T) {
	ctx := t.Context()
	s1 := stream.Of(ctx, 1, 2, 3)
	s2 := stream.Of(ctx, 4, 5, 6)
	got := stream.FanIn([]stream.Stream[int]{s1, s2}).Collect()
	assert.Len(t, got, 6)
	assert.ElementsMatch(t, []int{1, 2, 3, 4, 5, 6}, got)
}

func TestFanInEmpty(t *testing.T) {
	got := stream.FanIn[int](nil).Collect()
	assert.Nil(t, got)
}
