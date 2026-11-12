// Package goflow provides type-safe, composable stream processing built on
// generics, channels and context.
//
// A [Stream] carries elements of one type through a chain of stages. Each
// stage runs in its own goroutine, started with [github.com/foomo/gofuncy],
// and stops when the stream's context ends.
//
// # Constructors
//
// Create a stream from values ([Of], [OfMap], [Empty]), a channel ([From]),
// an iterator ([FromIter]), a producer function ([FromFunc], [Generate],
// [Iterate]) or a writable entry point ([Pipe], [PipeFunc]).
//
// # Operators
//
// Operators that keep the element type are methods, for example
// [Stream.Filter], [Stream.Take] and [Stream.Sort]. Operators that change the
// element type are functions, for example [Map], [FlatMap], [Batch] and
// [Reduce].
//
// Concurrency operators split and merge streams: [Stream.FanOut],
// [Stream.Tee], [FanIn], [FanMap] and [Stream.Process].
//
// # Terminals
//
// Terminal methods read the stream to its end: [Stream.Collect],
// [Stream.Count], [Stream.ForEach], [Stream.Iter] and [Stream.FindFirst].
//
//	ctx := context.Background()
//	evens := goflow.Of(ctx, 1, 2, 3, 4).
//		Filter(func(_ context.Context, n int) bool { return n%2 == 0 }).
//		Collect() // [2 4]
//
// # Pipelines and telemetry
//
// [Pipe] and [PipeFunc] return a [Sender] that feeds a long-lived pipeline.
// Configure them with [Option] values such as [WithName] and [WithBuffer].
// Named pipelines report OpenTelemetry metrics and consumer spans; the
// attribute keys and instruments are defined in
// [github.com/foomo/goflow/semconv] and
// [github.com/foomo/goflow/semconv/goflowconv].
package goflow
