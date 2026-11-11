package goflow

import "context"

// AllMatch reports whether all elements match fn. It stops at the first
// element that does not match. It reports true for an empty stream and false
// when the stream's context ends first.
func (s Stream[T]) AllMatch(fn func(context.Context, T) bool) bool {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			return false
		}

		if !fn(s.ctx, item) {
			return false
		}
	}

	return true
}

// AnyMatch reports whether any element matches fn. It stops at the first
// element that matches. It reports false for an empty stream and when the
// stream's context ends first.
func (s Stream[T]) AnyMatch(fn func(context.Context, T) bool) bool {
	for item := range s.items() {
		if s.ctx.Err() != nil {
			return false
		}

		if fn(s.ctx, item) {
			return true
		}
	}

	return false
}

// NoneMatch reports whether no element matches fn. It is the negation of
// [Stream.AnyMatch], so it reports true for an empty stream and when the
// stream's context ends first.
func (s Stream[T]) NoneMatch(fn func(context.Context, T) bool) bool {
	return !s.AnyMatch(fn)
}
