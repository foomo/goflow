package goflow

// KeyValue holds a single key-value pair from a map.
type KeyValue[K comparable, V any] struct {
	Key   K
	Value V
}
