package goflow

import (
	"context"
)

// MapEach applies Map to each stream in a slice, returning a slice of transformed streams.
func MapEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, error)) []Stream[U] {
	out := make([]Stream[U], len(streams))
	for i, s := range streams {
		out[i] = Map(s, fn)
	}

	return out
}
