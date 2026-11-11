package goflow

import "context"

// FindFirst returns the first element of the stream and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) FindFirst() (T, bool) {
	for item := range s.items() {
		return item, true
	}

	var zero T

	return zero, false
}

// FindFirstMatch returns the first element matching the predicate and true,
// or the zero value and false if no element matches.
func (s Stream[T]) FindFirstMatch(fn func(context.Context, T) bool) (T, bool) {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			break
		}

		if fn(s.ctx, item) {
			return item, true
		}
	}

	var zero T

	return zero, false
}
