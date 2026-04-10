package goflow

import "context"

// Reduce folds all elements into a single value using fn.
// Returns the accumulated result or the first error from fn.
func Reduce[T, U any](s Stream[T], initial U, fn func(context.Context, U, T) (U, error)) (U, error) {
	acc := initial

	for item := range s.source {
		if s.ctx.Err() != nil {
			return acc, s.ctx.Err()
		}

		var err error

		acc, err = fn(s.ctx, acc, item)
		if err != nil {
			return acc, err
		}
	}

	return acc, nil
}
