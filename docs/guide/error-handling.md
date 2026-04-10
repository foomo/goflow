# Error Handling

goflow operators fall into two categories based on how they handle errors.

## Error-Propagating Operators

These operators accept callbacks that return an `error`. When the callback returns a non-nil error, the stream closes and the error is forwarded to gofuncy's error handler (which can log, trace, or propagate it).

| Operator | Callback Signature |
|---|---|
| `Map` | `func(context.Context, T) (U, error)` |
| `MapFilter` | `func(context.Context, T) (U, bool, error)` |
| `FlatMap` | `func(context.Context, T) Stream[U]` |
| `Reduce` | `func(context.Context, U, T) (U, error)` |
| `ForEach` | `func(context.Context, T) error` |
| `Process` | `func(context.Context, T) error` |
| `FromFunc` | `func(context.Context, func(T) error) error` |
| `PipeFunc` | `func(context.Context, Stream[T]) error` |

For intermediate operators (`Map`, `MapFilter`, `FlatMap`), an error closes the output channel and the error is passed to gofuncy.Go's error handler. Downstream operators see a closed channel and terminate normally.

For terminal operators (`Reduce`, `ForEach`), the error is returned directly to the caller.

For `Process`, all worker errors are collected and returned via `errors.Join`, so you see every failure.

## Non-Error Operators

These operators do not return errors from their callbacks. They either use simple predicates or perform side-effect-free operations:

- **Predicates:** `Filter`, `Distinct`, `FindFirstMatch`, `AllMatch`, `AnyMatch`, `NoneMatch`
- **Side-effects:** `Peek`
- **Ordering:** `Sort`, `Reverse`
- **Slicing:** `Take`, `Skip`, `Throttle`
- **Aggregation:** `Min`, `Max`, `FindFirst`, `Collect`, `Count`

## How Errors Flow Through gofuncy

Every goroutine in goflow is launched via `gofuncy.Go`, which provides:

1. **Named goroutines** -- each operator registers a name (e.g., `"goflow.map"`, `"goflow.filter"`) for tracing and debugging.
2. **Error handler** -- when a callback returns an error, gofuncy invokes the configured error handler before closing the stream.
3. **OpenTelemetry spans** -- if configured, each operator creates a trace span that captures the error.

You configure these behaviors via `gofuncy.GoOption` on the stream:

```go
s := goflow.Of(ctx, items...).WithOptions(
    gofuncy.WithErrorHandler(func(err error) {
        slog.Error("stream operator failed", "error", err)
    }),
)
```

All downstream operators from `s` inherit these options.

## Wrapping Callbacks for Error Context

When building complex pipelines, add context to errors so you can identify which stage failed:

```go
result := goflow.Map(s, func(ctx context.Context, item Item) (Result, error) {
    out, err := transform(ctx, item)
    if err != nil {
        return Result{}, fmt.Errorf("transform item %s: %w", item.ID, err)
    }
    return out, nil
})
```

::: tip
Always wrap errors with `fmt.Errorf` and `%w` to preserve the error chain. This makes it possible to use `errors.Is` and `errors.As` on the returned error.
:::

## Context Cancellation as Error Signal

Context cancellation is the primary mechanism for stopping a pipeline. When the context is cancelled:

1. All operators check `ctx.Done()` on every send/receive cycle.
2. Channels close in cascade from source to sink.
3. Terminal operators return whatever has been collected so far.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

// This pipeline automatically stops after 5 seconds
results := goflow.Generate(ctx, produceItem).
    Filter(func(_ context.Context, item Item) bool {
        return item.IsValid()
    }).
    Collect()
```

::: warning
When a context is cancelled, `Collect()` returns the elements received so far -- it does not return an error. If you need to distinguish between a completed stream and a cancelled one, check `ctx.Err()` after the terminal operation.
:::

## Pattern: Graceful Shutdown

For long-running pipelines, combine context cancellation with ForEach to handle errors explicitly:

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

err := goflow.FromFunc(ctx, 64, subscribeToEvents).
    ForEach(func(ctx context.Context, event Event) error {
        if err := handleEvent(ctx, event); err != nil {
            return fmt.Errorf("handle event %s: %w", event.ID, err)
        }
        return nil
    })

if err != nil && !errors.Is(err, context.Canceled) {
    log.Fatalf("pipeline failed: %v", err)
}
```

::: tip
Use `ForEach` or `Process` as terminal operators when you need to observe errors. `Collect`, `Count`, and other aggregation terminals silently stop on context cancellation.
:::

## Pattern: Skip vs Stop on Error

With `MapFilter`, you can choose whether an error should stop the pipeline or just skip the problematic element:

```go
// Stop on error
results := goflow.MapFilter(s, func(ctx context.Context, raw string) (Parsed, bool, error) {
    p, err := parse(raw)
    if err != nil {
        return Parsed{}, false, err // stops the stream
    }
    return p, true, nil
})

// Skip on error
results := goflow.MapFilter(s, func(ctx context.Context, raw string) (Parsed, bool, error) {
    p, err := parse(raw)
    if err != nil {
        return Parsed{}, false, nil // skips this element
    }
    return p, true, nil
})
```

::: danger
Silently skipping errors can hide bugs. Consider logging skipped items via Peek or a structured logger so failures remain visible.
:::
