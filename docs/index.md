---
layout: home

hero:
  name: goflow
  image:
    src: /logo.png
  text: Type-safe Stream Processing for Go
  tagline: Composable, concurrent, context-aware stream pipelines built on Go generics
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: API Reference
      link: /api/reference

features:
  - title: Type-safe Generics
    details: Every operator is fully generic. The compiler catches type mismatches at build time, not at runtime.
  - title: Composable Operators
    details: Chain Map, Filter, FlatMap, Reduce, and 30+ other operators into expressive, readable pipelines.
  - title: Concurrency Primitives
    details: FanOut, FanIn, FanMap, Tee, and Process give you parallel pipelines with minimal boilerplate.
  - title: Context-Aware
    details: Every stream carries a context.Context. Cancellation propagates automatically through the entire pipeline.
  - title: OpenTelemetry Tracing
    details: Built on gofuncy, every goroutine is named and traceable. Plug in your own error handler and span options.
  - title: iter.Seq Integration
    details: Convert between streams and Go 1.23 iterators with Iter(), Iter2(), and FromIter() for seamless interop.
---
