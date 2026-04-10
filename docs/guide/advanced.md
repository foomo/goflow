# Advanced Usage

## Pipe and PipeFunc: Imperative Push

`Pipe` and `PipeFunc` let you push values into a stream imperatively, bridging callback-based or event-driven code into the stream world.

### Pipe

`Pipe` returns a send function and a readable stream. You call the send function from any goroutine to push values into the stream. The stream closes automatically when the context is cancelled.

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

send, s := goflow.Pipe[string](ctx)

// Push values from a goroutine
go func() {
    send(ctx, "hello")
    send(ctx, "world")
    cancel() // close the stream when done
}()

results := s.Collect()
fmt.Println(results) // [hello world]
```

You can optionally provide a buffer size to reduce blocking:

```go
send, s := goflow.Pipe[int](ctx, 256)
```

### PipeFunc

`PipeFunc` creates a Pipe and launches a consumer function in a managed goroutine. It returns only the send handler, making it ideal for fire-and-forget consumption patterns:

```go
send := goflow.PipeFunc[Event](ctx, func(ctx context.Context, s goflow.Stream[Event]) error {
    return s.ForEach(func(ctx context.Context, e Event) error {
        return store.Save(ctx, e)
    })
})

// Push events from handlers
send(ctx, Event{Type: "click"})
send(ctx, Event{Type: "scroll"})
```

## FromFunc: Bridging External Sources

`FromFunc` creates a stream from a blocking function that produces items via a `send` callback. This is the primary way to bridge external data sources -- message queues, file readers, network connections -- into the stream model.

The function should block until it is done producing items and must respect context cancellation. `bufSize` controls backpressure: a full buffer blocks the send callback until the downstream consumer catches up.

```go
// Bridge a message subscriber into a stream
s := goflow.FromFunc(ctx, 16, func(ctx context.Context, send func(Event) error) error {
    return sub.Subscribe(ctx, "events", func(ctx context.Context, msg courier.Message[Event]) error {
        return send(msg.Payload)
    })
})

// Now process with stream operators
err := s.Filter(func(_ context.Context, e Event) bool {
    return e.Type == "order"
}).ForEach(func(ctx context.Context, e Event) error {
    return processOrder(ctx, e)
})
```

::: tip
Choose `bufSize` based on your producer's burst rate and your consumer's throughput. A larger buffer absorbs temporary speed mismatches but uses more memory.
:::

## WithOptions: Propagating gofuncy Options

`WithOptions` attaches `gofuncy.GoOption` values to a stream. These options are inherited by every downstream operator, controlling error handling, tracing, and recovery behavior.

```go
s := goflow.Of(ctx, items...).WithOptions(
    gofuncy.WithErrorHandler(func(err error) {
        slog.Error("pipeline error", "error", err)
    }),
)

// All operators downstream inherit the error handler
result := goflow.Map(s, transformFn).
    Filter(filterFn).
    Collect()
```

Options accumulate: calling `WithOptions` again appends to the existing set rather than replacing it.

## Channel Buffering

By default, all intermediate operators use unbuffered channels. This provides natural backpressure but can limit throughput when producers and consumers run at different speeds.

Use `Pipe` with a buffer size to introduce buffering at specific points in the pipeline:

```go
send, buffered := goflow.Pipe[int](ctx, 1024)

// Use FromFunc with the send function for a buffered producer
go func() {
    for i := 0; i < 1000000; i++ {
        if err := send(ctx, i); err != nil {
            return
        }
    }
}()

result := buffered.Filter(filterFn).Collect()
```

::: warning
Buffering shifts backpressure from blocking to memory usage. A large buffer can mask a slow consumer while consuming significant memory. Monitor your pipeline's memory footprint in production.
:::

## iter.Seq Integration

goflow interoperates with Go 1.23 range-over-function iterators through three methods.

### Iter -- Stream to iter.Seq

`Iter` returns an `iter.Seq[T]` that yields each element. The iterator drains the stream and can only be used once.

```go
s := goflow.Of(ctx, 1, 2, 3)

for v := range s.Iter() {
    fmt.Println(v)
}
```

### Iter2 -- Stream to iter.Seq2

`Iter2` returns an `iter.Seq2[int, T]` that yields each element with its zero-based index.

```go
s := goflow.Of(ctx, "a", "b", "c")

for i, v := range s.Iter2() {
    fmt.Printf("%d: %s\n", i, v)
}
```

### FromIter -- iter.Seq to Stream

`FromIter` creates a stream from any `iter.Seq[T]`, enabling you to use stream operators on standard library iterators:

```go
import "slices"

s := goflow.FromIter(ctx, slices.Values([]int{10, 20, 30}))
doubled := goflow.Map(s, func(_ context.Context, n int) (int, error) {
    return n * 2, nil
})
```

::: tip
Use `Iter()` when you want to hand off stream data to code that expects a range-over-function iterator. Use `FromIter()` when you want to pull data from an iterator into a stream pipeline.
:::

## Composing Pipelines Across Packages

Because `Stream[T]` is a plain struct, you can pass streams across package boundaries and compose pipelines modularly.

```go
// package ingest
func EventStream(ctx context.Context) goflow.Stream[Event] {
    return goflow.FromFunc(ctx, 64, func(ctx context.Context, send func(Event) error) error {
        return subscribe(ctx, send)
    })
}

// package transform
func EnrichEvents(s goflow.Stream[Event]) goflow.Stream[EnrichedEvent] {
    return goflow.Map(s, func(ctx context.Context, e Event) (EnrichedEvent, error) {
        return enrich(ctx, e)
    })
}

// package main
func main() {
    ctx := context.Background()

    events := ingest.EventStream(ctx)
    enriched := transform.EnrichEvents(events)

    err := enriched.Process(8, func(ctx context.Context, e EnrichedEvent) error {
        return store(ctx, e)
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

Each package defines its own segment of the pipeline. The stream carries its context and options through, so the composed pipeline behaves as a single unit with consistent cancellation and error handling.
