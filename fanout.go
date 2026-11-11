package goflow

import "context"

// FanOut distributes elements round-robin across n output streams.
func (s Stream[T]) FanOut(n int) []Stream[T] {
	return s.spread(n, "goflow.fan-out", func(i int, _ []chan T) int { return i % n })
}

// Tee broadcasts every element to n output streams.
// Unlike FanOut which round-robins, Tee sends each element to all streams.
func (s Stream[T]) Tee(n int) []Stream[T] {
	return s.spread(n, "goflow.tee", func(int, []chan T) int { return -1 })
}

// spread starts one stage that feeds n output streams. pick returns the index
// of the output for the i-th element, or -1 to send it to every output.
func (s Stream[T]) spread(n int, name string, pick func(i int, outs []chan T) int) []Stream[T] {
	if n <= 0 || s.ctx.Err() != nil {
		return nil
	}

	outs := make([]chan T, n)
	for i := range outs {
		outs[i] = make(chan T)
	}

	goStage(s.ctx, func(ctx context.Context) error {
		defer func() {
			for _, ch := range outs {
				close(ch)
			}
		}()

		i := 0

		for item := range s.items() {
			if j := pick(i, outs); j >= 0 {
				if !send(ctx, outs[j], item) {
					return nil
				}
			} else {
				for _, ch := range outs {
					if !send(ctx, ch, item) {
						return nil
					}
				}
			}

			i++
		}

		return nil
	}, s.stageOpts(name)...)

	streams := make([]Stream[T], n)
	for i, ch := range outs {
		streams[i] = derive[T](s, ch)
	}

	return streams
}
