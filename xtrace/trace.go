package xtrace

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func TraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)

	// get current Span's SpanContext
	spanContext := span.SpanContext()
	spanContext.TraceState()

	return spanContext.TraceID().String()
}
