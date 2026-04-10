# Concurrency

goflow provides several primitives for parallel stream processing. All concurrency operators respect context cancellation and propagate `gofuncy.GoOption` settings for tracing and error handling.

## FanOut / FanIn

The FanOut/FanIn pattern distributes work across multiple goroutines and merges the results back into a single stream.

**FanOut** distributes elements round-robin across `n` output streams. **FanIn** merges multiple streams into one, with elements arriving in non-deterministic order.

```
                    +--- Stream[T] (worker 0) ---+
                    |                             |
  Stream[T] --FanOut--> Stream[T] (worker 1) --FanIn--> Stream[T]
                    |                             |
                    +--- Stream[T] (worker 2) ---+
```

```go
ctx := context.Background()

s := goflow.Of(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9)

// Fan out to 3 workers
partitions := s.FanOut(3)

// Apply a transformation to each partition
transformed := goflow.MapEach(partitions, func(ctx context.Context, n int) (int, error) {
    return n * n, nil
})

// Merge results back
result := goflow.FanIn(transformed).Collect()
fmt.Println(result) // order is non-deterministic
```

::: warning
FanOut sends to unbuffered channels. It blocks on the slowest consumer. If one partition is not being consumed, the entire upstream pipeline stalls.
:::

## FanMap -- The Shorthand

`FanMap` combines FanOut, MapEach, and FanIn into a single call. It is the most common way to parallelise a transformation.

```go
results := goflow.FanMap(s, 4, func(ctx context.Context, url string) (Response, error) {
    return httpGet(ctx, url)
})
```

This is equivalent to:

```go
results := goflow.FanIn(goflow.MapEach(s.FanOut(4), fn))
```

`FanMapFilter` does the same but with a MapFilter function, allowing you to skip items during the concurrent transformation:

```go
results := goflow.FanMapFilter(s, 4, func(ctx context.Context, id string) (User, bool, error) {
    u, err := fetchUser(ctx, id)
    if err != nil {
        return User{}, false, nil // skip on error
    }
    return u, true, nil
})
```

## Process -- Parallel Consumption

`Process` is a terminal operator that dispatches each element to a bounded worker pool. Unlike FanMap which produces a new stream, Process consumes the stream and collects errors.

```go
err := goflow.Of(ctx, urls...).Process(8, func(ctx context.Context, url string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return saveResponse(ctx, url, resp)
})
if err != nil {
    log.Fatal(err) // errors.Join of all worker errors
}
```

Process uses `gofuncy.NewGroup` under the hood with a concurrency limit. All errors are collected and returned via `errors.Join`, so you see every failure rather than just the first.

## Tee -- Broadcasting

`Tee` sends every element to all `n` output streams. This is useful when you need to process the same data in multiple ways simultaneously.

```
                    +--- Stream[T] (log)
                    |
  Stream[T] --Tee---+
                    |
                    +--- Stream[T] (process)
```

```go
copies := s.Tee(2)

// Consumer 1: log everything
go func() {
    copies[0].ForEach(func(_ context.Context, item Item) error {
        log.Printf("received: %v", item)
        return nil
    })
}()

// Consumer 2: process items
err := copies[1].ForEach(func(ctx context.Context, item Item) error {
    return process(ctx, item)
})
```

::: warning
Tee blocks on the slowest consumer. Every output stream must be actively consumed, or the pipeline will deadlock. If consumers run at different speeds, consider inserting a buffered Pipe between Tee and the slower consumer.
:::

## Tuning with WithOptions and Buffer Sizes

For high-throughput pipelines, you can tune behavior with buffer sizes and gofuncy options.

**Buffer sizes** on Pipe reduce blocking between producers and consumers:

```go
send, s := goflow.Pipe[int](ctx, 1024) // 1024-element buffer
```

**WithOptions** propagates gofuncy options (error handlers, tracing) to all downstream operators:

```go
s := goflow.Of(ctx, items...).WithOptions(
    gofuncy.WithErrorHandler(func(err error) {
        log.Printf("stream error: %v", err)
    }),
)
```

## Real-World Example: Parallel HTTP Fetching

```go
package main

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/foomo/goflow"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    urls := []string{
        "https://example.com/api/1",
        "https://example.com/api/2",
        "https://example.com/api/3",
        "https://example.com/api/4",
        "https://example.com/api/5",
    }

    // Fetch up to 3 URLs concurrently, skip failures
    bodies := goflow.FanMapFilter(
        goflow.Of(ctx, urls...),
        3,
        func(ctx context.Context, url string) (string, bool, error) {
            req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
            if err != nil {
                return "", false, nil
            }

            resp, err := http.DefaultClient.Do(req)
            if err != nil {
                return "", false, nil // skip failed requests
            }
            defer resp.Body.Close()

            body, err := io.ReadAll(resp.Body)
            if err != nil {
                return "", false, nil
            }

            return string(body), true, nil
        },
    )

    results := bodies.Collect()
    fmt.Printf("fetched %d responses\n", len(results))
}
```
