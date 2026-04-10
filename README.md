[![Build Status](https://github.com/foomo/goflow/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/goflow/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/goflow)](https://goreportcard.com/report/github.com/foomo/goflow)
[![GoDoc](https://godoc.org/github.com/foomo/goflow?status.svg)](https://godoc.org/github.com/foomo/goflow)

<p align="center">
  <img alt="goflow" src="docs/public/logo.png" width="400" height="400"/>
</p>

# goflow

Type-safe, composable stream processing for Go.

goflow provides a generic `Stream[T]` type backed by channels and context. It ships with functional operators (Map, Filter, Reduce, FlatMap) and concurrency primitives (FanOut, FanIn, Process, Tee). Built on [gofuncy](https://github.com/foomo/gofuncy) for goroutine management with OpenTelemetry tracing.

## Install

```sh
go get github.com/foomo/goflow
```

## Quick Example

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
            return n * n, nil
        },
    ).Collect()

    fmt.Println(result) // [4 16 36 64 100]
}
```

## Features

- Generic `Stream[T]` with compile-time type safety
- Composable functional operators (Map, Filter, FlatMap, Reduce, and more)
- Built-in concurrency (FanOut, FanIn, FanMap, Process, Tee)
- Context-aware cancellation and timeout propagation
- Channel buffering control via `Pipe`
- `iter.Seq`/`iter.Seq2` integration
- OpenTelemetry tracing via gofuncy

## Documentation

- [User Guide](https://foomo.github.io/goflow/)
- [API Reference](https://pkg.go.dev/github.com/foomo/goflow)

## How to Contribute

Contributions are welcome! Please read the [contributing guide](docs/CONTRIBUTING.md).

![Contributors](https://contributors-table.vercel.app/image?repo=foomo/goflow&width=50&columns=15)

## License

Distributed under MIT License, please see the [license](LICENSE) file within the code for more details.

_Made with ♥ [foomo](https://www.foomo.org) by [bestbytes](https://www.bestbytes.com)_
