# Graceful Shutdown

Every goflow pipeline is context-aware. When the context is cancelled, the entire pipeline shuts down cooperatively -- no forced kills, no leaked goroutines.

## How Cancellation Propagates

Each `Stream[T]` carries a `context.Context`. Every operator checks `ctx.Done()` before sending an element to the next stage:

```
  ctx cancelled
       │
       ▼
  Generate ──► Map ──► Filter ──► Collect
     │           │        │          │
   stops      stops    stops    returns partial
  producing  sending  sending      results
```

The shutdown sequence:

1. **Context cancels** -- via `cancel()`, timeout, or OS signal.
2. **Operators detect `ctx.Done()`** -- each operator's goroutine exits its `select` loop.
3. **Channels close via `defer`** -- every operator uses `defer close(source)` to signal downstream.
4. **Terminal operators return** -- `Collect()` returns partial results; `ForEach()`/`Process()` return errors.

### The `closed()` Fast Path

If the context is already cancelled when an operator is called, it returns an immediately-closed stream without spawning a goroutine. This avoids unnecessary work in pre-cancelled pipelines:

```go
ctx, cancel := context.WithCancel(context.Background())
cancel() // cancel before building the pipeline

s := goflow.Of(ctx, 1, 2, 3) // returns closed stream instantly
s.Count()                      // 0
```

## Signal-Based Shutdown

For long-running services, use `signal.NotifyContext` to tie the pipeline to OS signals:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()

err := goflow.FromFunc(ctx, 64, subscribeToEvents).
    ForEach(func(ctx context.Context, event Event) error {
        return handleEvent(ctx, event)
    })

if err != nil && !errors.Is(err, context.Canceled) {
    log.Fatalf("pipeline failed: %v", err)
}

log.Println("shutdown complete")
```

When `SIGINT` or `SIGTERM` arrives, the context cancels and the pipeline drains cleanly.

## Timeout-Based Shutdown

Use `context.WithTimeout` for pipelines that must complete within a deadline:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results := goflow.Generate(ctx, produceItem).
    Filter(isValid).
    Collect()

fmt.Printf("collected %d items before deadline\n", len(results))
```

## Partial Results and In-Flight Items

When the context cancels:

- **`Collect()`** returns elements received so far. It prioritizes draining buffered channel items before checking cancellation, so you get as much data as possible.
- **`ForEach()` / `Process()`** stop accepting new items and return errors. `Process` collects all worker errors via `errors.Join`.
- **In-flight items** in unbuffered channels are dropped -- the sending operator exits its `select` and the item is never delivered.

::: warning
`Collect()` does not return an error on cancellation. If you need to distinguish a completed pipeline from a cancelled one, check `ctx.Err()` after the terminal operation.
:::

## Distinguishing Cancellation from Errors

```go
err := pipeline.ForEach(func(ctx context.Context, item Item) error {
    return process(ctx, item)
})

switch {
case err == nil:
    log.Println("pipeline completed normally")
case errors.Is(err, context.Canceled):
    log.Println("pipeline was cancelled")
case errors.Is(err, context.DeadlineExceeded):
    log.Println("pipeline timed out")
default:
    log.Printf("pipeline failed: %v", err)
}
```

## Goroutine Safety

All operator goroutines are managed by `gofuncy.Go`, which ensures:

- Goroutines exit when their context cancels.
- Channels are closed via `defer`, unblocking all readers.
- No goroutine leaks -- verified in the test suite with `go.uber.org/goleak`.

This means you can safely cancel a pipeline at any point without worrying about orphaned goroutines or resource leaks.
