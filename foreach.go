package goflow

import "context"

// ForEach consumes the stream, calling fn for each element.
// Returns the first error from fn or ctx, nil when fully consumed.
func (s Stream[T]) ForEach(fn func(context.Context, T) error) error {
	for item := range s.source {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		if err := fn(s.ctx, item); err != nil {
			return err
		}
	}

	return nil
}
