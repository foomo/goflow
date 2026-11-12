package goflow

import (
	"context"
)

// Empty returns a closed stream with no elements, bound to
// [context.Background].
func Empty[T any]() Stream[T] {
	return Stream[T]{ctx: context.Background(), source: closedChan[T]()}
}
