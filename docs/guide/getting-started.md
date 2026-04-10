# Getting Started

## What is goflow?

goflow is a generic stream processing library for Go. It provides a `Stream[T]` type backed by channels and `context.Context`, a rich set of functional operators for transformation, filtering, and aggregation, and concurrency primitives for parallel pipelines. It is built on top of [gofuncy](https://github.com/foomo/gofuncy) for goroutine lifecycle management, error handling, and OpenTelemetry tracing.

## Installation

```sh
go get github.com/foomo/goflow
```

## Core Concept

The fundamental type is `Stream[T]`, which wraps three things:

- A `context.Context` for cancellation and deadline propagation
- A `<-chan T` as the underlying data source
- Optional `gofuncy.GoOption` values for error handling and tracing

Every operator either produces a new `Stream` (intermediate) or consumes a stream to produce a result (terminal). Intermediate operators are lazy -- they set up the pipeline but do not pull data until a terminal operator drives consumption.

## Creating Streams

goflow offers several constructors to create streams from different sources:

```go
package main

import (
	"context"
	"iter"

	"github.com/foomo/goflow"
)

func main() {
	ctx := context.Background()

	// From variadic values
	s1 := goflow.Of(ctx, 1, 2, 3, 4, 5)

	// From an existing channel
	ch := make(chan string, 3)
	s2 := goflow.From(ctx, ch)

	// From an iter.Seq
	seq := func(yield func(int) bool) {
		for i := 0; i < 10; i++ {
			if !yield(i) {
				return
			}
		}
	}
	s3 := goflow.FromIter(ctx, iter.Seq[int](seq))

	// Infinite stream via generator
	counter := 0
	s4 := goflow.Generate(ctx, func() int {
		counter++
		return counter
	})

	// Infinite stream via seed + function
	s5 := goflow.Iterate(ctx, 1, func(n int) int { return n * 2 })

	// From a blocking producer function
	s6 := goflow.FromFunc(ctx, 16, func(ctx context.Context, send func(int) error) error {
		for i := 0; i < 100; i++ {
			if err := send(i); err != nil {
				return err
			}
		}
		return nil
	})

	// From a map
	s7 := goflow.OfMap(ctx, map[string]int{"a": 1, "b": 2})

	// Writable pipe
	send, s8 := goflow.Pipe[int](ctx)

	_, _, _, _, _, _, _, _ = s1, s2, s3, s4, s5, s6, s7, s8
	_ = send
}
```

## Basic Pipeline

Here is a complete, runnable example that creates a stream of integers, filters for even numbers, doubles them, and collects the results:

```go
package main

import (
	"context"
	"fmt"

	"github.com/foomo/goflow"
)

func main() {
	ctx := context.Background()

	result := goflow.Map(
		goflow.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10).
			Filter(func(_ context.Context, n int) bool {
				return n%2 == 0
			}),
		func(_ context.Context, n int) (int, error) {
			return n * 2, nil
		},
	).Collect()

	fmt.Println(result) // [4 8 12 16 20]
}
```

## Terminal Operations

Terminal operations consume the stream and produce a final result. Once a terminal operation runs, the stream is fully drained.

| Operation | Description |
|---|---|
| `Collect() []T` | Collects all elements into a slice. |
| `Count() int` | Returns the number of elements. |
| `ForEach(fn) error` | Calls fn for each element; returns the first error. |
| `Reduce(initial, fn) (U, error)` | Folds all elements into a single value. |
| `ToMap(key, value) map[K]V` | Collects elements into a map. |
| `FindFirst() (T, bool)` | Returns the first element, or false if empty. |
| `FindFirstMatch(fn) (T, bool)` | Returns the first element matching fn. |
| `AllMatch(fn) bool` | True if all elements match fn. |
| `AnyMatch(fn) bool` | True if any element matches fn. |
| `NoneMatch(fn) bool` | True if no elements match fn. |
| `Min(cmp) (T, bool)` | Returns the minimum element according to cmp. |
| `Max(cmp) (T, bool)` | Returns the maximum element according to cmp. |
| `Process(n, fn, opts...) error` | Dispatches elements to a worker pool of size n. |

::: tip
`FindFirst`, `FindFirstMatch`, `AnyMatch`, and `NoneMatch` short-circuit: they stop consuming the stream as soon as the result is determined.
:::
