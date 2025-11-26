package xtrace

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func InjectDetector() {
	tp := otel.GetTracerProvider()
	r, ok := tp.(*trace.TracerProvider)
	if !ok {
		return
	}

	r.RegisterSpanProcessor(NewSizeDetectorProcessor(SizeLimitConfig{
		AttrMaxBytes: 64 * 1024,       // single attribute max bytes
		SpanMaxBytes: 4 * 1024 * 1024, // single span max bytes
	}))
}
