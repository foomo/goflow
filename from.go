package goflow

import (
	"context"
)

// From wraps an existing channel as a Stream with the given context.
func From[T any](ctx context.Context, source <-chan T) Stream[T] {
	return Stream[T]{ctx: ctx, source: source}
}
