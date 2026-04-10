package goflow

import (
	"context"
)

// FanMap fans out a stream into n partitions, maps each concurrently, and fans in the results.
// Output order is non-deterministic.
func FanMap[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, error)) Stream[U] {
	return FanIn(MapEach(s.FanOut(n), fn))
}
