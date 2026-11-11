// Package semconv defines the OpenTelemetry attribute keys used by goflow
// metrics and traces.
//
// All keys live under the goflow namespace. Values that are recorded on
// metrics are bounded: pipeline names are static labels set with
// goflow.WithName, all other metric attributes are enums.
package semconv
