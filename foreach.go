package goflow

import (
	"context"

	"github.com/foomo/goflow/semconv"
)

// ForEach consumes the stream, calling fn for each element.
// Returns the first error from fn or ctx, nil when fully consumed.
//
// With [WithErrorHandler], errors from fn are passed to the handler and
// consumption continues; ForEach then only returns ctx errors or nil.
func (s Stream[T]) ForEach(fn func(context.Context, T) error, opts ...Option) error {
	c := newConfig(opts)

	return consume(s.ctx, s, semconv.StageNameForEach, fn, func(ctx context.Context, err error) error {
		if c.errorHandler == nil {
			return err
		}

		c.errorHandler(ctx, err)

		return nil
	})
}
