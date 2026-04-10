package goflow

// ToMap collects all stream elements into a map using the key and value functions.
// If duplicate keys occur, the last value wins.
func ToMap[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V) map[K]V {
	out := make(map[K]V)

	for item := range s.source {
		if s.ctx.Err() != nil {
			break
		}

		out[key(item)] = value(item)
	}

	return out
}
