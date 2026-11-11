package goflow

// Split groups consecutive elements into batches of size n.
// The last batch may contain fewer than n elements.
// It is Batch with MaxItems(n).
func Split[T any](s Stream[T], n int) Stream[[]T] {
	if n <= 0 {
		return closed[[]T](s.ctx)
	}

	return Batch(s, MaxItems(n))
}
