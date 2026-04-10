# API Reference

Complete reference for all exported symbols in the `goflow` package.

Import path: `github.com/foomo/goflow`

## Types

### Stream

```go
type Stream[T any] struct {
    // unexported fields
}
```

The core type representing a lazy, context-aware, channel-backed sequence of elements. A stream carries a `context.Context`, a receive channel `<-chan T`, and optional `gofuncy.GoOption` values. Streams are consumed once -- after a terminal operation drains the channel, the stream is exhausted.

### KeyValue

```go
type KeyValue[K comparable, V any] struct {
    Key   K
    Value V
}
```

Holds a single key-value pair. Used by `OfMap` and related operators when streaming over map entries.

---

## Constructors

### Of

```go
func Of[T any](ctx context.Context, items ...T) Stream[T]
```

Returns a stream that emits the given items. The items are buffered into a channel with capacity equal to `len(items)`. If `items` is empty or the context is already cancelled, returns a closed empty stream.

**Parameters:**
- `ctx` -- context for cancellation propagation.
- `items` -- variadic elements to emit.

### From

```go
func From[T any](ctx context.Context, source <-chan T) Stream[T]
```

Wraps an existing receive channel as a stream with the given context. Does not launch any goroutines. The caller is responsible for closing the channel when done.

**Parameters:**
- `ctx` -- context bound to the stream.
- `source` -- the channel to wrap.

### FromIter

```go
func FromIter[T any](ctx context.Context, seq iter.Seq[T]) Stream[T]
```

Creates a stream from a Go 1.23 `iter.Seq[T]` by pushing elements into a channel in a background goroutine. The stream closes when the iterator is exhausted or the context is cancelled.

**Parameters:**
- `ctx` -- context for cancellation.
- `seq` -- the iterator to consume.

### FromFunc

```go
func FromFunc[T any](ctx context.Context, bufSize int, fn func(ctx context.Context, send func(T) error) error, opts ...gofuncy.GoOption) Stream[T]
```

Creates a stream from a blocking function that sends items via the provided `send` callback. The function should block until it is done producing items and must respect context cancellation. The stream closes automatically when `fn` returns. If `fn` returns a non-nil error, it is handled by `gofuncy.Go`.

**Parameters:**
- `ctx` -- context for cancellation.
- `bufSize` -- channel buffer size controlling backpressure. A full buffer blocks the send callback.
- `fn` -- the producer function. Receives a context and a `send` callback.
- `opts` -- optional `gofuncy.GoOption` for error handling and tracing.

### Generate

```go
func Generate[T any](ctx context.Context, fn func() T) Stream[T]
```

Returns an infinite stream where each element is produced by calling `fn`. The stream runs until the context is cancelled.

**Parameters:**
- `ctx` -- context for cancellation (required to stop the infinite stream).
- `fn` -- generator function called repeatedly for each element.

### Iterate

```go
func Iterate[T any](ctx context.Context, seed T, fn func(T) T) Stream[T]
```

Returns an infinite stream: `seed`, `fn(seed)`, `fn(fn(seed))`, and so on. The stream runs until the context is cancelled.

**Parameters:**
- `ctx` -- context for cancellation.
- `seed` -- the first element emitted.
- `fn` -- function applied to the previous element to produce the next.

### OfMap

```go
func OfMap[K comparable, V any](ctx context.Context, m map[K]V) Stream[KeyValue[K, V]]
```

Returns a stream of `KeyValue[K, V]` pairs from the given map. Iteration order is non-deterministic, matching Go map semantics. The channel is buffered to `len(m)`.

**Parameters:**
- `ctx` -- context for cancellation.
- `m` -- the map to stream over.

### Empty

```go
func Empty[T any]() Stream[T]
```

Returns an immediately closed, empty stream with a background context.

### Pipe

```go
func Pipe[T any](ctx context.Context, bufferSize ...int) (func(context.Context, T) error, Stream[T])
```

Creates a writable stream entry point. Returns a `send` function and the readable stream. The send function returns `ctx.Err()` if the stream's context is cancelled, or `sctx.Err()` if the send context is cancelled. The channel is closed when `ctx` is done.

**Parameters:**
- `ctx` -- context controlling the stream's lifetime.
- `bufferSize` -- optional channel buffer size (default 0, unbuffered).

**Returns:**
- `send` -- function to push values; safe to call from multiple goroutines.
- `Stream[T]` -- the readable stream.

### PipeFunc

```go
func PipeFunc[T any](ctx context.Context, fn func(context.Context, Stream[T]) error, opts ...gofuncy.GoOption) func(context.Context, T) error
```

Creates a Pipe and launches the consumer `fn` in a `gofuncy.Go` goroutine. Returns only the send handler.

**Parameters:**
- `ctx` -- context controlling the stream's lifetime.
- `fn` -- consumer function that processes the stream.
- `opts` -- optional `gofuncy.GoOption` for the consumer goroutine.

---

## Transformers

### Map

```go
func Map[T, U any](s Stream[T], fn func(context.Context, T) (U, error)) Stream[U]
```

Returns a new stream by applying `fn` to each element of the source stream. If `fn` returns an error, the stream closes and the error is handled by `gofuncy.Go`. Propagates the source stream's options.

**Parameters:**
- `s` -- the source stream.
- `fn` -- transformation function. Receives the stream's context and the element.

### FlatMap

```go
func FlatMap[T, U any](s Stream[T], fn func(context.Context, T) Stream[U]) Stream[U]
```

Applies `fn` to each element of the source stream, producing a sub-stream per element, and flattens the results into a single output stream. Sub-streams are consumed sequentially in source order.

**Parameters:**
- `s` -- the source stream.
- `fn` -- function that maps each element to a sub-stream.

### MapFilter

```go
func MapFilter[T, U any](s Stream[T], fn func(context.Context, T) (U, bool, error)) Stream[U]
```

Maps and filters elements in a single pass. The boolean controls emission: `(val, true, nil)` emits `val`; `(_, false, nil)` skips the item; `(_, _, err)` stops the stream with an error.

**Parameters:**
- `s` -- the source stream.
- `fn` -- function returning the mapped value, whether to emit, and an optional error.

### Flatten

```go
func Flatten[T any](s Stream[[]T]) Stream[T]
```

Flattens a stream of slices into a stream of individual elements. Each slice is emitted element by element in order.

**Parameters:**
- `s` -- a stream of slices.

### Reverse

```go
func (s Stream[T]) Reverse() Stream[T]
```

Collects all elements into memory and emits them in reverse order. This is a blocking operation that buffers the entire stream.

### Sort

```go
func (s Stream[T]) Sort(cmp func(T, T) int) Stream[T]
```

Collects all elements, sorts them using `cmp` (via `slices.SortFunc`), and emits in sorted order. This is a blocking operation that buffers the entire stream.

**Parameters:**
- `cmp` -- comparison function. Returns negative if a < b, zero if equal, positive if a > b.

### Split

```go
func Split[T any](s Stream[T], n int) Stream[[]T]
```

Groups consecutive elements into batches of size `n`. The last batch may contain fewer than `n` elements.

**Parameters:**
- `s` -- the source stream.
- `n` -- batch size. Must be > 0.

### Window

```go
func Window[T any](s Stream[T], n int) Stream[[]T]
```

Emits sliding windows of `n` consecutive elements. Each window is a fresh slice. If the source has fewer than `n` elements, no windows are emitted.

**Parameters:**
- `s` -- the source stream.
- `n` -- window size. Must be > 0.

---

## Filters

### Filter

```go
func (s Stream[T]) Filter(fn func(context.Context, T) bool) Stream[T]
```

Returns a stream containing only elements where `fn` returns true.

**Parameters:**
- `fn` -- predicate function. Receives the stream's context and the element.

### Distinct

```go
func (s Stream[T]) Distinct(key func(T) string) Stream[T]
```

Deduplicates elements using a key function. First occurrence wins; subsequent elements with the same key are dropped. Maintains a `map[string]struct{}` of seen keys internally.

**Parameters:**
- `key` -- function that extracts a string key from each element.

### Take

```go
func (s Stream[T]) Take(n int) Stream[T]
```

Emits the first `n` elements then closes the stream. If the source has fewer than `n` elements, all are emitted.

**Parameters:**
- `n` -- maximum number of elements to emit.

### Skip

```go
func (s Stream[T]) Skip(n int) Stream[T]
```

Drops the first `n` elements and emits the rest.

**Parameters:**
- `n` -- number of elements to skip.

### Peek

```go
func (s Stream[T]) Peek(fn func(context.Context, T)) Stream[T]
```

Calls `fn` for each element as a side-effect and forwards the element unchanged. Useful for logging, metrics, or debugging without altering the stream.

**Parameters:**
- `fn` -- side-effect function. Receives the stream's context and the element.

### Throttle

```go
func (s Stream[T]) Throttle(d time.Duration) Stream[T]
```

Rate-limits the stream to at most one element per duration `d`. The first element passes through immediately; subsequent elements are delayed by at least `d` from the previous emission.

**Parameters:**
- `d` -- minimum duration between elements.

---

## Consumers

### Collect

```go
func (s Stream[T]) Collect() []T
```

Drains the stream and returns all elements as a slice. Prioritizes draining available items over context cancellation. Returns the elements received so far if the context is cancelled.

### Count

```go
func (s Stream[T]) Count() int
```

Returns the number of elements in the stream. Prioritizes draining available items over context cancellation.

### ForEach

```go
func (s Stream[T]) ForEach(fn func(context.Context, T) error) error
```

Consumes the stream, calling `fn` for each element. Returns the first error from `fn` or `ctx.Err()`, nil when fully consumed.

**Parameters:**
- `fn` -- consumer function. Receives the stream's context and the element.

### Reduce

```go
func Reduce[T, U any](s Stream[T], initial U, fn func(context.Context, U, T) (U, error)) (U, error)
```

Folds all elements into a single value using `fn`. Returns the accumulated result or the first error from `fn`. If the context is cancelled mid-reduction, returns the accumulated value so far and `ctx.Err()`.

**Parameters:**
- `s` -- the source stream.
- `initial` -- the starting accumulator value.
- `fn` -- fold function. Receives the context, the current accumulator, and the next element.

### ToMap

```go
func ToMap[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V) map[K]V
```

Collects all stream elements into a map using the key and value functions. If duplicate keys occur, the last value wins. Stops early if the context is cancelled.

**Parameters:**
- `s` -- the source stream.
- `key` -- function extracting the map key from each element.
- `value` -- function extracting the map value from each element.

### FindFirst

```go
func (s Stream[T]) FindFirst() (T, bool)
```

Returns the first element of the stream and true, or the zero value and false if the stream is empty. Short-circuits after the first element.

### FindFirstMatch

```go
func (s Stream[T]) FindFirstMatch(fn func(context.Context, T) bool) (T, bool)
```

Returns the first element matching the predicate and true, or the zero value and false if no element matches. Short-circuits on the first match.

**Parameters:**
- `fn` -- predicate function.

### AllMatch

```go
func (s Stream[T]) AllMatch(fn func(context.Context, T) bool) bool
```

Returns true if all elements match the predicate. Short-circuits on the first non-matching element. Returns true for an empty stream. Returns false if the context is cancelled.

**Parameters:**
- `fn` -- predicate function.

### AnyMatch

```go
func (s Stream[T]) AnyMatch(fn func(context.Context, T) bool) bool
```

Returns true if any element matches the predicate. Short-circuits on the first matching element. Returns false for an empty stream. Returns false if the context is cancelled.

**Parameters:**
- `fn` -- predicate function.

### NoneMatch

```go
func (s Stream[T]) NoneMatch(fn func(context.Context, T) bool) bool
```

Returns true if no elements match the predicate. Short-circuits on the first matching element. Returns true for an empty stream. Implemented as `!s.AnyMatch(fn)`.

**Parameters:**
- `fn` -- predicate function.

### Min

```go
func (s Stream[T]) Min(cmp func(T, T) int) (T, bool)
```

Returns the minimum element according to `cmp` and true, or the zero value and false if the stream is empty. Consumes the entire stream.

**Parameters:**
- `cmp` -- comparison function. Returns negative if a < b, zero if equal, positive if a > b.

### Max

```go
func (s Stream[T]) Max(cmp func(T, T) int) (T, bool)
```

Returns the maximum element according to `cmp` and true, or the zero value and false if the stream is empty. Consumes the entire stream.

**Parameters:**
- `cmp` -- comparison function. Returns negative if a < b, zero if equal, positive if a > b.

### Process

```go
func (s Stream[T]) Process(n int, fn func(context.Context, T) error, opts ...gofuncy.GroupOption) error
```

Consumes the stream, dispatching each element to a worker pool of size `n`. Uses `gofuncy.NewGroup` with a concurrency limit. All errors are collected and returned via `errors.Join`.

**Parameters:**
- `n` -- maximum number of concurrent workers.
- `fn` -- worker function. Receives the stream's context and the element.
- `opts` -- optional `gofuncy.GroupOption` for the worker group.

---

## Concurrency

### FanOut

```go
func (s Stream[T]) FanOut(n int) []Stream[T]
```

Distributes elements round-robin across `n` output streams. Each output stream receives approximately `1/n` of the elements. Uses unbuffered channels, so it blocks on the slowest consumer. Returns nil if `n <= 0` or the context is cancelled.

**Parameters:**
- `n` -- number of output partitions.

### FanIn

```go
func FanIn[T any](streams []Stream[T]) Stream[T]
```

Combines multiple streams into a single stream. Elements arrive in non-deterministic order as they become available from the input streams. Uses the context and options from the first stream. Returns `Empty[T]()` if the slice is empty.

**Parameters:**
- `streams` -- the streams to merge.

### FanMap

```go
func FanMap[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, error)) Stream[U]
```

Fans out a stream into `n` partitions, maps each concurrently, and fans in the results. Output order is non-deterministic. Equivalent to `FanIn(MapEach(s.FanOut(n), fn))`.

**Parameters:**
- `s` -- the source stream.
- `n` -- number of concurrent workers.
- `fn` -- transformation function.

### FanMapFilter

```go
func FanMapFilter[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, bool, error)) Stream[U]
```

Fans out, applies MapFilter concurrently across `n` workers, and fans in the results. Equivalent to `FanIn(MapFilterEach(s.FanOut(n), fn))`.

**Parameters:**
- `s` -- the source stream.
- `n` -- number of concurrent workers.
- `fn` -- map-filter function.

### Tee

```go
func (s Stream[T]) Tee(n int) []Stream[T]
```

Broadcasts every element to `n` output streams. Unlike FanOut which round-robins, Tee sends each element to all streams. Uses unbuffered channels, so it blocks on the slowest consumer. Returns nil if `n <= 0` or the context is cancelled.

**Parameters:**
- `n` -- number of output copies.

### MapEach

```go
func MapEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, error)) []Stream[U]
```

Applies `Map` to each stream in a slice, returning a slice of transformed streams.

**Parameters:**
- `streams` -- the input streams.
- `fn` -- transformation function applied to each stream.

### MapFilterEach

```go
func MapFilterEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, bool, error)) []Stream[U]
```

Applies `MapFilter` to each stream in a slice, returning a slice of transformed streams.

**Parameters:**
- `streams` -- the input streams.
- `fn` -- map-filter function applied to each stream.

---

## Combinators

### Concat

```go
func Concat[T any](streams ...Stream[T]) Stream[T]
```

Returns a stream that emits all elements from each input stream in order: first all elements from `streams[0]`, then `streams[1]`, and so on. Uses the context and options from the first stream. Returns `Empty[T]()` if no streams are provided.

**Parameters:**
- `streams` -- the streams to concatenate in order.

---

## Utility Methods

### Context

```go
func (s Stream[T]) Context() context.Context
```

Returns the stream's bound context.

### WithOptions

```go
func (s Stream[T]) WithOptions(opts ...gofuncy.GoOption) Stream[T]
```

Returns a shallow copy of the stream with the given options appended to any existing options. Options are propagated to all downstream operators spawned from this stream.

**Parameters:**
- `opts` -- `gofuncy.GoOption` values to append.

### Chan

```go
func (s Stream[T]) Chan() <-chan T
```

Returns the underlying receive channel. Reading from this channel directly bypasses stream operators. The channel is closed when the stream is exhausted or the context is cancelled.

### Iter

```go
func (s Stream[T]) Iter() iter.Seq[T]
```

Returns an `iter.Seq[T]` that yields each element of the stream. The returned iterator drains the stream; it can only be used once. Respects context cancellation.

### Iter2

```go
func (s Stream[T]) Iter2() iter.Seq2[int, T]
```

Returns an `iter.Seq2[int, T]` that yields each element with its zero-based index. The returned iterator drains the stream; it can only be used once. Respects context cancellation.
