package goflow

import (
	"context"
	"errors"
	"sync"

	"github.com/foomo/gofuncy"

	"github.com/foomo/goflow/semconv"
)

// Process consumes the stream with n concurrent workers.
// Errors are collected and returned via [errors.Join] once the stream ends.
//
// With [WithErrorHandler], errors are passed to the handler instead and not
// retained, which suits long-lived streams. Use [WithGroupOptions] to configure
// the underlying [gofuncy.Group].
func (s Stream[T]) Process(n int, fn func(context.Context, T) error, opts ...Option) error {
	c := newConfig(opts)

	if n < 1 {
		n = 1
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	onErr := func(ctx context.Context, err error) error {
		if c.errorHandler != nil {
			c.errorHandler(ctx, err)

			return nil
		}

		mu.Lock()
		defer mu.Unlock()

		errs = append(errs, err)

		return nil
	}

	g := gofuncy.NewGroup(s.ctx, append(s.p.groupOpts("goflow.process"), c.groupOpts...)...)
	for range n {
		g.Add(func(ctx context.Context) error {
			// Same cancel policy as ForEach; the error is reported once below.
			if err := consume(ctx, s, semconv.StageNameProcess, fn, onErr); err != nil && ctx.Err() == nil {
				return err
			}

			return nil
		}, gofuncy.WithName("goflow.process.worker"))
	}

	errs = append(errs, g.Wait(), s.ctx.Err())

	return errors.Join(errs...)
}
