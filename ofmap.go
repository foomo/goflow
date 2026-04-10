package goflow

import (
	"context"

	"github.com/foomo/gofuncy"
)

// KeyValue holds a single key-value pair from a map.
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

// OfMap returns a Stream of KeyValue pairs from the given map.
// Iteration order is non-deterministic, matching Go's map semantics.
func OfMap[K comparable, V any](ctx context.Context, m map[K]V) Stream[KeyValue[K, V]] {
	if len(m) == 0 || ctx.Err() != nil {
		return closed[KeyValue[K, V]](ctx)
	}

	source := make(chan KeyValue[K, V], len(m))

	gofuncy.Go(ctx, "goflow.of-map", func(ctx context.Context) error {
		defer close(source)

		for k, v := range m {
			select {
			case <-ctx.Done():
				return nil
			case source <- KeyValue[K, V]{Key: k, Value: v}:
			}
		}

		return nil
	})

	return From[KeyValue[K, V]](ctx, source)
}
