package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

type Stream[T any] struct {
	ctx    context.Context //nolint:containedctx // by design
	source <-chan T
	opts   []gofuncy.GoOption
}

// Context returns the stream's bound context.
func (s Stream[T]) Context() context.Context {
	return s.ctx
}

// WithOptions returns a shallow copy of the stream with the given options appended.
func (s Stream[T]) WithOptions(opts ...gofuncy.GoOption) Stream[T] {
	combined := make([]gofuncy.GoOption, 0, len(s.opts)+len(opts))
	combined = append(combined, s.opts...)
	combined = append(combined, opts...)

	return Stream[T]{ctx: s.ctx, source: s.source, opts: combined}
}

func (s Stream[T]) Chan() <-chan T {
	return s.source
}

func (s Stream[T]) Count() int {
	var count int

	for {
		// Prioritize draining available items over cancellation.
		select {
		case _, ok := <-s.source:
			if !ok {
				return count
			}

			count++
		default:
			select {
			case <-s.ctx.Done():
				return count
			case _, ok := <-s.source:
				if !ok {
					return count
				}

				count++
			}
		}
	}
}

func (s Stream[T]) Collect() []T {
	var out []T

	for {
		// Prioritize draining available items over cancellation.
		select {
		case v, ok := <-s.source:
			if !ok {
				return out
			}

			out = append(out, v)
		default:
			select {
			case <-s.ctx.Done():
				return out
			case v, ok := <-s.source:
				if !ok {
					return out
				}

				out = append(out, v)
			}
		}
	}
}

func Empty[T any]() Stream[T] {
	source := make(chan T)
	close(source)

	return Stream[T]{ctx: context.Background(), source: source}
}

// closed returns an empty, closed stream that preserves the given context.
func closed[T any](ctx context.Context) Stream[T] {
	source := make(chan T)
	close(source)

	return Stream[T]{ctx: ctx, source: source}
}

// Of returns a Stream based on the given elements.
func Of[T any](ctx context.Context, items ...T) Stream[T] {
	n := len(items)
	if n == 0 || ctx.Err() != nil {
		return closed[T](ctx)
	}

	source := make(chan T, n)

	gofuncy.Go(ctx, "goflow.of", func(ctx context.Context) error {
		defer close(source)

		for _, item := range items {
			select {
			case <-ctx.Done():
				return nil
			case source <- item:
			}
		}

		return nil
	})

	return From[T](ctx, source)
}

// From wraps an existing channel as a Stream with the given context.
func From[T any](ctx context.Context, source <-chan T) Stream[T] {
	return Stream[T]{ctx: ctx, source: source}
}
