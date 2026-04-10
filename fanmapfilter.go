package goflow

import (
	"context"
)

// FanMapFilter fans out, applies MapFilter concurrently, and fans in the results.
func FanMapFilter[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, bool, error)) Stream[U] {
	return FanIn(MapFilterEach(s.FanOut(n), fn))
}
