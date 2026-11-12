// Package goflowconv provides typed OpenTelemetry instruments for the goflow
// metrics, one type per instrument.
//
// Each constructor returns a no-op instrument for a nil meter or on creation
// error, so the result is always safe to use. Required attributes are
// positional parameters of the recording methods.
package goflowconv
