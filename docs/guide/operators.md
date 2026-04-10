# Operators

This page lists every goflow operator grouped by category. Each entry shows the Go signature, a description, and a short usage example.

## Constructors

### Of

```go
func Of[T any](ctx context.Context, items ...T) Stream[T]
```

Creates a stream from variadic values. The items are buffered into a channel of matching capacity.

```go
s := goflow.Of(ctx, "a", "b", "c")
```

### From

```go
func From[T any](ctx context.Context, source <-chan T) Stream[T]
```

Wraps an existing receive channel as a stream with the given context.

```go
ch := make(chan int, 10)
s := goflow.From(ctx, ch)
```

### FromIter

```go
func FromIter[T any](ctx context.Context, seq iter.Seq[T]) Stream[T]
```

Creates a stream from a Go 1.23 `iter.Seq[T]` by pushing elements into a channel.

```go
s := goflow.FromIter(ctx, slices.Values([]int{1, 2, 3}))
```

### FromFunc

```go
func FromFunc[T any](ctx context.Context, bufSize int, fn func(ctx context.Context, send func(T) error) error, opts ...gofuncy.GoOption) Stream[T]
```

Creates a stream from a blocking function that sends items via the provided `send` callback. The function should block until it is done producing items and must respect context cancellation. `bufSize` controls backpressure.

```go
s := goflow.FromFunc(ctx, 16, func(ctx context.Context, send func(int) error) error {
    for i := 0; i < 100; i++ {
        if err := send(i); err != nil {
            return err
        }
    }
    return nil
})
```

### Generate

```go
func Generate[T any](ctx context.Context, fn func() T) Stream[T]
```

Returns an infinite stream where each element is produced by `fn`. The stream runs until the context is cancelled.

```go
s := goflow.Generate(ctx, func() int { return rand.Intn(100) })
```

### Iterate

```go
func Iterate[T any](ctx context.Context, seed T, fn func(T) T) Stream[T]
```

Returns an infinite stream: `seed`, `fn(seed)`, `fn(fn(seed))`, and so on. The stream runs until the context is cancelled.

```go
s := goflow.Iterate(ctx, 1, func(n int) int { return n * 2 })
// 1, 2, 4, 8, 16, ...
```

### OfMap

```go
func OfMap[K comparable, V any](ctx context.Context, m map[K]V) Stream[KeyValue[K, V]]
```

Returns a stream of `KeyValue[K, V]` pairs from the given map. Iteration order is non-deterministic, matching Go map semantics.

```go
s := goflow.OfMap(ctx, map[string]int{"x": 1, "y": 2})
```

### Empty

```go
func Empty[T any]() Stream[T]
```

Returns an immediately closed, empty stream.

```go
s := goflow.Empty[int]()
```

### Pipe

```go
func Pipe[T any](ctx context.Context, bufferSize ...int) (func(context.Context, T) error, Stream[T])
```

Creates a writable stream entry point. Returns a `send` function and the readable stream. The send function returns `ctx.Err()` if the context is cancelled. The channel is closed when the context is done.

```go
send, s := goflow.Pipe[string](ctx)
go func() {
    send(ctx, "hello")
    send(ctx, "world")
}()
```

### PipeFunc

```go
func PipeFunc[T any](ctx context.Context, fn func(context.Context, Stream[T]) error, opts ...gofuncy.GoOption) func(context.Context, T) error
```

Creates a Pipe and launches the consumer `fn` in a `gofuncy.Go` goroutine. Returns only the send handler.

```go
send := goflow.PipeFunc[int](ctx, func(ctx context.Context, s goflow.Stream[int]) error {
    return s.ForEach(func(_ context.Context, n int) error {
        fmt.Println(n)
        return nil
    })
})
send(ctx, 42)
```

---

## Transformers

### Map

```go
func Map[T, U any](s Stream[T], fn func(context.Context, T) (U, error)) Stream[U]
```

Returns a new stream by applying `fn` to each element. If `fn` returns an error, the stream closes and the error is handled by `gofuncy.Go`.

```go
doubled := goflow.Map(s, func(_ context.Context, n int) (int, error) {
    return n * 2, nil
})
```

### FlatMap

```go
func FlatMap[T, U any](s Stream[T], fn func(context.Context, T) Stream[U]) Stream[U]
```

Applies `fn` to each element, producing a sub-stream per element, and flattens the results into a single output stream sequentially.

```go
expanded := goflow.FlatMap(s, func(ctx context.Context, n int) goflow.Stream[int] {
    return goflow.Of(ctx, n, n*10, n*100)
})
```

### MapFilter

```go
func MapFilter[T, U any](s Stream[T], fn func(context.Context, T) (U, bool, error)) Stream[U]
```

Maps and filters in a single pass. `(val, true, nil)` emits `val`; `(_, false, nil)` skips the item; `(_, _, err)` stops the stream.

```go
parsed := goflow.MapFilter(s, func(_ context.Context, raw string) (int, bool, error) {
    n, err := strconv.Atoi(raw)
    if err != nil {
        return 0, false, nil // skip unparseable
    }
    return n, true, nil
})
```

### Flatten

```go
func Flatten[T any](s Stream[[]T]) Stream[T]
```

Flattens a stream of slices into a stream of individual elements.

```go
flat := goflow.Flatten(goflow.Of(ctx, []int{1, 2}, []int{3, 4}))
// 1, 2, 3, 4
```

### Reverse

```go
func (s Stream[T]) Reverse() Stream[T]
```

Collects all elements and emits them in reverse order.

```go
reversed := goflow.Of(ctx, 1, 2, 3).Reverse()
// 3, 2, 1
```

::: warning
Reverse buffers the entire stream in memory before emitting. Do not use on unbounded streams.
:::

### Sort

```go
func (s Stream[T]) Sort(cmp func(T, T) int) Stream[T]
```

Collects all elements, sorts them using `cmp`, and emits in sorted order.

```go
sorted := goflow.Of(ctx, 3, 1, 2).Sort(func(a, b int) int { return a - b })
// 1, 2, 3
```

::: warning
Sort buffers the entire stream in memory before emitting. Do not use on unbounded streams.
:::

### Split

```go
func Split[T any](s Stream[T], n int) Stream[[]T]
```

Groups consecutive elements into batches of size `n`. The last batch may contain fewer than `n` elements.

```go
batches := goflow.Split(goflow.Of(ctx, 1, 2, 3, 4, 5), 2)
// [1,2], [3,4], [5]
```

### Window

```go
func Window[T any](s Stream[T], n int) Stream[[]T]
```

Emits sliding windows of `n` consecutive elements. If the source has fewer than `n` elements, no windows are emitted.

```go
windows := goflow.Window(goflow.Of(ctx, 1, 2, 3, 4), 3)
// [1,2,3], [2,3,4]
```

---

## Filters

### Filter

```go
func (s Stream[T]) Filter(fn func(context.Context, T) bool) Stream[T]
```

Returns a stream containing only elements where `fn` returns true.

```go
evens := goflow.Of(ctx, 1, 2, 3, 4).Filter(func(_ context.Context, n int) bool {
    return n%2 == 0
})
```

### Distinct

```go
func (s Stream[T]) Distinct(key func(T) string) Stream[T]
```

Deduplicates elements using a key function. First occurrence wins.

```go
unique := s.Distinct(func(item Item) string { return item.ID })
```

::: warning
Distinct maintains a map of all seen keys in memory. For streams with high cardinality, memory usage grows without bound.
:::

### Take

```go
func (s Stream[T]) Take(n int) Stream[T]
```

Emits the first `n` elements then closes the stream.

```go
first3 := goflow.Of(ctx, 1, 2, 3, 4, 5).Take(3)
// 1, 2, 3
```

### Skip

```go
func (s Stream[T]) Skip(n int) Stream[T]
```

Drops the first `n` elements and emits the rest.

```go
rest := goflow.Of(ctx, 1, 2, 3, 4, 5).Skip(2)
// 3, 4, 5
```

### Peek

```go
func (s Stream[T]) Peek(fn func(context.Context, T)) Stream[T]
```

Calls `fn` for each element as a side-effect and forwards the element unchanged. Useful for logging or debugging.

```go
s.Peek(func(_ context.Context, n int) {
    log.Printf("processing: %d", n)
})
```

### Throttle

```go
func (s Stream[T]) Throttle(d time.Duration) Stream[T]
```

Rate-limits the stream to at most one element per duration `d`.

```go
limited := s.Throttle(100 * time.Millisecond)
```

---

## Consumers

### Collect

```go
func (s Stream[T]) Collect() []T
```

Drains the stream and returns all elements as a slice.

```go
items := goflow.Of(ctx, 1, 2, 3).Collect()
```

### Count

```go
func (s Stream[T]) Count() int
```

Returns the number of elements in the stream.

```go
n := goflow.Of(ctx, 1, 2, 3).Count()
// 3
```

### ForEach

```go
func (s Stream[T]) ForEach(fn func(context.Context, T) error) error
```

Consumes the stream, calling `fn` for each element. Returns the first error from `fn` or the context, nil when fully consumed.

```go
err := s.ForEach(func(_ context.Context, item Item) error {
    return db.Save(item)
})
```

### Reduce

```go
func Reduce[T, U any](s Stream[T], initial U, fn func(context.Context, U, T) (U, error)) (U, error)
```

Folds all elements into a single value using `fn`. Returns the accumulated result or the first error from `fn`.

```go
sum, err := goflow.Reduce(goflow.Of(ctx, 1, 2, 3), 0,
    func(_ context.Context, acc, n int) (int, error) {
        return acc + n, nil
    },
)
```

### ToMap

```go
func ToMap[T any, K comparable, V any](s Stream[T], key func(T) K, value func(T) V) map[K]V
```

Collects all stream elements into a map using the key and value functions. If duplicate keys occur, the last value wins.

```go
m := goflow.ToMap(s, func(u User) string { return u.ID }, func(u User) User { return u })
```

### FindFirst

```go
func (s Stream[T]) FindFirst() (T, bool)
```

Returns the first element and true, or the zero value and false if the stream is empty.

```go
first, ok := s.FindFirst()
```

### FindFirstMatch

```go
func (s Stream[T]) FindFirstMatch(fn func(context.Context, T) bool) (T, bool)
```

Returns the first element matching the predicate and true, or the zero value and false if no element matches.

```go
admin, ok := s.FindFirstMatch(func(_ context.Context, u User) bool {
    return u.Role == "admin"
})
```

### AllMatch

```go
func (s Stream[T]) AllMatch(fn func(context.Context, T) bool) bool
```

Returns true if all elements match the predicate. Short-circuits on the first non-matching element. Returns true for an empty stream.

```go
allPositive := s.AllMatch(func(_ context.Context, n int) bool { return n > 0 })
```

### AnyMatch

```go
func (s Stream[T]) AnyMatch(fn func(context.Context, T) bool) bool
```

Returns true if any element matches the predicate. Short-circuits on the first matching element. Returns false for an empty stream.

```go
hasNegative := s.AnyMatch(func(_ context.Context, n int) bool { return n < 0 })
```

### NoneMatch

```go
func (s Stream[T]) NoneMatch(fn func(context.Context, T) bool) bool
```

Returns true if no elements match the predicate. Short-circuits on the first matching element. Returns true for an empty stream.

```go
noErrors := s.NoneMatch(func(_ context.Context, r Result) bool { return r.Err != nil })
```

### Min

```go
func (s Stream[T]) Min(cmp func(T, T) int) (T, bool)
```

Returns the minimum element according to `cmp` and true, or the zero value and false if the stream is empty.

```go
smallest, ok := s.Min(func(a, b int) int { return a - b })
```

### Max

```go
func (s Stream[T]) Max(cmp func(T, T) int) (T, bool)
```

Returns the maximum element according to `cmp` and true, or the zero value and false if the stream is empty.

```go
largest, ok := s.Max(func(a, b int) int { return a - b })
```

### Process

```go
func (s Stream[T]) Process(n int, fn func(context.Context, T) error, opts ...gofuncy.GroupOption) error
```

Consumes the stream, dispatching each element to a worker pool of size `n`. All errors are collected and returned via `errors.Join`.

```go
err := s.Process(4, func(ctx context.Context, item Item) error {
    return upload(ctx, item)
})
```

---

## Concurrency

### FanOut

```go
func (s Stream[T]) FanOut(n int) []Stream[T]
```

Distributes elements round-robin across `n` output streams.

```go
streams := s.FanOut(3)
```

::: warning
FanOut blocks on the slowest consumer. If one output stream is not being consumed, the entire pipeline stalls.
:::

### FanIn

```go
func FanIn[T any](streams []Stream[T]) Stream[T]
```

Combines multiple streams into a single stream. Elements arrive in non-deterministic order as they become available. Uses the context and options from the first stream.

```go
merged := goflow.FanIn(streams)
```

### FanMap

```go
func FanMap[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, error)) Stream[U]
```

Fans out a stream into `n` partitions, maps each concurrently, and fans in the results. Output order is non-deterministic. This is a shorthand for `FanIn(MapEach(s.FanOut(n), fn))`.

```go
results := goflow.FanMap(s, 4, func(ctx context.Context, url string) (Response, error) {
    return httpGet(ctx, url)
})
```

### FanMapFilter

```go
func FanMapFilter[T, U any](s Stream[T], n int, fn func(context.Context, T) (U, bool, error)) Stream[U]
```

Fans out, applies MapFilter concurrently, and fans in the results. Equivalent to `FanIn(MapFilterEach(s.FanOut(n), fn))`.

```go
results := goflow.FanMapFilter(s, 4, func(ctx context.Context, id string) (User, bool, error) {
    u, err := fetchUser(ctx, id)
    if err != nil {
        return User{}, false, nil // skip failures
    }
    return u, true, nil
})
```

### Tee

```go
func (s Stream[T]) Tee(n int) []Stream[T]
```

Broadcasts every element to `n` output streams. Unlike FanOut which round-robins, Tee sends each element to all streams.

```go
copies := s.Tee(2)
// copies[0] and copies[1] both receive every element
```

::: warning
Tee blocks on the slowest consumer. If any output stream is not consumed, the pipeline stalls. Consider adding buffered channels via Pipe if consumers have different speeds.
:::

### MapEach

```go
func MapEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, error)) []Stream[U]
```

Applies Map to each stream in a slice, returning a slice of transformed streams.

```go
transformed := goflow.MapEach(streams, fn)
```

### MapFilterEach

```go
func MapFilterEach[T, U any](streams []Stream[T], fn func(context.Context, T) (U, bool, error)) []Stream[U]
```

Applies MapFilter to each stream in a slice.

```go
filtered := goflow.MapFilterEach(streams, fn)
```

---

## Combinators

### Concat

```go
func Concat[T any](streams ...Stream[T]) Stream[T]
```

Returns a stream that emits all elements from each input stream in order: first all elements from `streams[0]`, then `streams[1]`, and so on. Uses the context and options from the first stream.

```go
all := goflow.Concat(s1, s2, s3)
```

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

Returns a shallow copy of the stream with the given options appended. Options are propagated to all downstream operators.

### Chan

```go
func (s Stream[T]) Chan() <-chan T
```

Returns the underlying receive channel.

### Iter

```go
func (s Stream[T]) Iter() iter.Seq[T]
```

Returns an `iter.Seq[T]` that yields each element of the stream. The returned iterator drains the stream and can only be used once.

### Iter2

```go
func (s Stream[T]) Iter2() iter.Seq2[int, T]
```

Returns an `iter.Seq2[int, T]` that yields each element with its zero-based index. The returned iterator drains the stream and can only be used once.
