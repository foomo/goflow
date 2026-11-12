# Examples

This section shows 10 common problems and how to solve them with goflow. Each page is a short, complete recipe. Replace the placeholder types and functions (`fetch`, `store`, `Event`, ...) with your own.

| # | Pattern | Main operators |
|---|---|---|
| 1 | [Parallel HTTP Requests](./parallel-http-requests.md) | `FanMapFilter` |
| 2 | [File Import with Bulk Inserts](./file-import-with-bulk-inserts.md) | `FromIter`, `MapFilter`, `Split` |
| 3 | [Background Job Queue](./background-job-queue.md) | `PipeFunc`, `Process` |
| 4 | [Event Buffering for a Slow Sink](./event-buffering-for-a-slow-sink.md) | `Pipe`, `Batch` |
| 5 | [Paginated API Crawl](./paginated-api-crawl.md) | `FromFunc`, `Take` |
| 6 | [Rate-Limited Calls with Deduplication](./rate-limited-calls-with-deduplication.md) | `Distinct`, `Throttle` |
| 7 | [Merge Multiple Sources](./merge-multiple-sources.md) | `FanIn` |
| 8 | [Broadcast to Multiple Sinks](./broadcast-to-multiple-sinks.md) | `Tee` |
| 9 | [Moving Average and Anomaly Detection](./moving-average-and-anomaly-detection.md) | `Window` |
| 10 | [First Match with Early Exit](./first-match-with-early-exit.md) | `FanMapFilter`, `FindFirst` |

## Error Handling Rules

Each page shows the happy path first, then an error handling section. These rules apply to all patterns:

1. **An error in a stage does not reach the terminal.** When `Map`, `MapFilter` or the function of `FromFunc` returns an error, the stream closes. `Collect` and `ForEach` then end normally. To report the error, carry it as a value in the element, or write it to a buffered channel before the function returns.
2. **Never stop one branch of a fan-out.** An error in a `FanMap` or `FanMapFilter` function stops one worker. A `Tee` consumer that returns stops one branch. `FanOut` and `Tee` then block, and the stream stalls until the context ends. Return errors as values, or cancel the context.
3. **Cancel the context when a terminal returns early.** `ForEach` returns at the first error, and `FindFirst` returns at the first element. The upstream stages keep running until the context ends.
4. **Retry one element, not the stage.** Wrap the call in `gofuncy.Do(ctx, fn, gofuncy.WithRetry(n))`. `WithRetry` on `WithOptions` restarts the whole stage goroutine.
5. **Keep long-lived consumers alive.** Pass `goflow.WithErrorHandler` to `ForEach` or `Process`. The handler gets each error, and the consumer goes on.

See [Error Handling](../guide/error-handling.md) for the operator details.
