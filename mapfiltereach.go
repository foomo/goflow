package goflow

import (
	"context"
)

// MapFilterEach applies MapFilter to each stream in a slice.
func MapFilterEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, bool, error)) []Stream[U] {
	out := make([]Stream[U], len(streams))
	for i, s := range streams {
		out[i] = MapFilter(s, fn)
	}

	return out
}
