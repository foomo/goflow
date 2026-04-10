package goflow

import "context"

// AllMatch returns true if all elements match the predicate.
// Short-circuits on the first non-matching element.
// Returns true for an empty stream.
func (s Stream[T]) AllMatch(fn func(context.Context, T) bool) bool {
	for item := range s.source {
		if s.ctx.Err() != nil {
			return false
		}

		if !fn(s.ctx, item) {
			return false
		}
	}

	return true
}

// AnyMatch returns true if any element matches the predicate.
// Short-circuits on the first matching element.
// Returns false for an empty stream.
func (s Stream[T]) AnyMatch(fn func(context.Context, T) bool) bool {
	for item := range s.source {
		if s.ctx.Err() != nil {
			return false
		}

		if fn(s.ctx, item) {
			return true
		}
	}

	return false
}

// NoneMatch returns true if no elements match the predicate.
// Short-circuits on the first matching element.
// Returns true for an empty stream.
func (s Stream[T]) NoneMatch(fn func(context.Context, T) bool) bool {
	return !s.AnyMatch(fn)
}
