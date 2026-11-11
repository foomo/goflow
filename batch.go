package goflow

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/foomo/goflow/semconv"
)

// Batch groups consecutive elements into batches.
//
// A batch is flushed when it reaches MaxItems, when the next element would
// push it over MaxBytes, when MaxWait has passed since its first element,
// or when the input closes (the partial batch is always flushed).
// Without any of these options Batch behaves like MaxItems(1).
//
// When the input comes from a Pipe, each batch keeps links to the origin spans
// of its elements (up to 128); a directly following ForEach or Process starts
// one span per batch that has links.
func Batch[T any](s Stream[T], opts ...Option) Stream[[]T] {
	c := newConfig(opts)

	var size func(T) int

	if c.sizeFn != nil {
		fn, ok := c.sizeFn.(func(T) int)
		if !ok {
			panic(fmt.Sprintf("goflow: MaxBytes size func %T does not match stream element type", c.sizeFn))
		}

		size = fn
	}

	if c.maxItems <= 0 && size == nil && c.maxWait <= 0 {
		c.maxItems = 1
	}

	if s.ctx.Err() != nil {
		return derive[[]T](s, closedChan[[]T]())
	}

	out := make(chan envelope[[]T])
	envCh, plainCh := s.envelopes()

	goStage(s.ctx, func(ctx context.Context) error {
		defer close(out)

		var (
			batch  []T
			bytes  int
			meta   *batchMeta
			timerC <-chan time.Time
		)

		// One timer serves every batch; it is armed with the first element
		// of a batch when MaxWait is set and released when the stage stops.
		timer := time.NewTimer(0)
		timer.Stop()

		defer timer.Stop()

		addLink := func(l trace.Link) {
			if len(meta.links) < maxLinks {
				meta.links = append(meta.links, l)
			} else {
				meta.droppedLinks++
			}
		}

		flush := func(reason semconv.FlushReasonAttr) bool {
			timer.Stop()

			timerC = nil

			meta.reason, meta.count = reason, len(batch)
			e := envelope[[]T]{v: batch, batch: meta}
			body := bytes
			batch, bytes, meta = nil, 0, nil

			if !send(ctx, out, e) {
				return false
			}

			// Record only batches that were handed off.
			if s.p != nil {
				s.p.m.flushedBatch(ctx, e.batch.count, reason)

				if size != nil {
					s.p.m.flushedBody(ctx, body, reason)
				}
			}

			return true
		}

		for {
			var (
				e  envelope[T]
				ok bool
			)

			select {
			case <-ctx.Done():
				return nil
			case <-timerC:
				if !flush(semconv.FlushReasonInterval) {
					return nil
				}

				continue
			case e, ok = <-envCh:
			case e.v, ok = <-plainCh:
			}

			if !ok {
				if len(batch) > 0 {
					flush(semconv.FlushReasonClose)
				}

				return nil
			}

			n := 0
			if size != nil {
				n = size(e.v)
				if len(batch) > 0 && bytes+n > c.maxBytes {
					if !flush(semconv.FlushReasonBytes) {
						return nil
					}
				}
			}

			if len(batch) == 0 {
				meta = &batchMeta{}

				if c.maxWait > 0 {
					timer.Reset(c.maxWait)
					timerC = timer.C
				}
			}

			batch = append(batch, e.v)
			bytes += n

			if e.sc.IsValid() {
				addLink(trace.Link{SpanContext: e.sc})
			}

			if b := e.batch; b != nil {
				for _, l := range b.links {
					addLink(l)
				}

				meta.droppedLinks += b.droppedLinks
			}

			var reason semconv.FlushReasonAttr

			switch {
			case c.maxItems > 0 && len(batch) >= c.maxItems:
				reason = semconv.FlushReasonCount
			case size != nil && bytes >= c.maxBytes:
				reason = semconv.FlushReasonBytes
			default:
				continue
			}

			if !flush(reason) {
				return nil
			}
		}
	}, s.stageOpts("goflow.batch")...)

	return deriveEnv(s, out)
}
