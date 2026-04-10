package goflow

// Min returns the minimum element according to cmp and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) Min(cmp func(T, T) int) (T, bool) {
	var best T

	found := false

	for item := range s.source {
		if s.ctx.Err() != nil {
			break
		}

		if !found || cmp(item, best) < 0 {
			best = item
			found = true
		}
	}

	return best, found
}

// Max returns the maximum element according to cmp and true,
// or the zero value and false if the stream is empty.
func (s Stream[T]) Max(cmp func(T, T) int) (T, bool) {
	var best T

	found := false

	for item := range s.source {
		if s.ctx.Err() != nil {
			break
		}

		if !found || cmp(item, best) > 0 {
			best = item
			found = true
		}
	}

	return best, found
}
