package goflow

import "context"

// KeyValue holds a single key-value pair from a map.
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}

// OfMap returns a Stream of KeyValue pairs from the given map.
// Iteration order is non-deterministic, matching Go's map semantics.
func OfMap[K comparable, V any](ctx context.Context, m map[K]V) Stream[KeyValue[K, V]] {
	if len(m) == 0 {
		return closed[KeyValue[K, V]](ctx)
	}

	return produce(ctx, "goflow.of-map", len(m), func(_ context.Context, emit func(KeyValue[K, V]) bool) error {
		for k, v := range m {
			if !emit(KeyValue[K, V]{Key: k, Value: v}) {
				return nil
			}
		}

		return nil
	})
}
