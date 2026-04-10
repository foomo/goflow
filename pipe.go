package goflow

import (
	"context"
	"sync"

	"github.com/foomo/gofuncy"
)

// Pipe creates a writable stream entry point.
// Returns a send function and the readable stream.
// The send function returns ctx.Err() if the context is cancelled.
// The channel is closed when ctx is done.
func Pipe[T any](ctx context.Context, bufferSize ...int) (func(context.Context, T) error, Stream[T]) {
	size := 0
	if len(bufferSize) > 0 {
		size = bufferSize[0]
	}

	ch := make(chan T, size)

	var mu sync.Mutex

	isClosed := false

	go func() {
		<-ctx.Done()
		mu.Lock()
		isClosed = true

		close(ch)
		mu.Unlock()
	}()

	send := func(sctx context.Context, v T) error {
		mu.Lock()
		defer mu.Unlock()

		if isClosed {
			return ctx.Err()
		}

		select {
		case <-sctx.Done():
			return sctx.Err()
		case ch <- v:
			return nil
		}
	}

	return send, From[T](ctx, ch)
}

// PipeFunc creates a Pipe and launches the consumer fn in a gofuncy.Go goroutine.
// Returns only the send handler.
func PipeFunc[T any](ctx context.Context, fn func(context.Context, Stream[T]) error, opts ...gofuncy.GoOption) func(context.Context, T) error {
	send, s := Pipe[T](ctx)
	gofuncy.Go(ctx, "goflow.pipe-func", func(ctx context.Context) error {
		return fn(ctx, s)
	}, opts...)

	return send
}
