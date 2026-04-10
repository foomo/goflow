package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// Distinct deduplicates elements using a key function. First occurrence wins.
func (s Stream[T]) Distinct(key func(T) string) Stream[T] {
	if s.ctx.Err() != nil {
		return closed[T](s.ctx)
	}

	source := make(chan T)

	gofuncy.Go(s.ctx, "goflow.distinct", func(ctx context.Context) error {
		defer close(source)

		seen := make(map[string]struct{})

		for item := range s.source {
			k := key(item)
			if _, ok := seen[k]; ok {
				continue
			}

			seen[k] = struct{}{}

			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	}, s.opts...)

	return Stream[T]{ctx: s.ctx, source: source, opts: s.opts}
}
