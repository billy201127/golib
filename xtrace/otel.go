package xtrace

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Config for size detector.
type SizeLimitConfig struct {
	AttrMaxBytes int // single attribute max bytes
	SpanMaxBytes int // single span max bytes
}

// NewSizeDetectorProcessor returns a span processor.
func NewSizeDetectorProcessor(cfg SizeLimitConfig) trace.SpanProcessor {
	return &sizeDetectorProcessor{cfg: cfg}
}

type sizeDetectorProcessor struct {
	cfg SizeLimitConfig
}

func (p *sizeDetectorProcessor) OnStart(ctx context.Context, s trace.ReadWriteSpan) {}

func (p *sizeDetectorProcessor) OnEnd(s trace.ReadOnlySpan) {
	p.checkSpan(s)
}

func (p *sizeDetectorProcessor) Shutdown(ctx context.Context) error   { return nil }
func (p *sizeDetectorProcessor) ForceFlush(ctx context.Context) error { return nil }

func (p *sizeDetectorProcessor) checkSpan(s trace.ReadOnlySpan) {
	spanName := s.Name()
	traceID := s.SpanContext().TraceID().String()

	totalSize := 0

	// --- 1. Check attributes ---
	for _, attr := range s.Attributes() {
		k := string(attr.Key)
		attrSize := p.calculateAttributeSize(attr)
		totalSize += attrSize

		if attrSize > p.cfg.AttrMaxBytes {
			logx.Errorf(
				"[OTEL-Detector] Big ATTR detected: span=%s trace=%s attr=%s size=%d bytes (limit=%d)",
				spanName, traceID, k, attrSize, p.cfg.AttrMaxBytes,
			)
		}
	}

	// --- 2. Check events ---
	for _, e := range s.Events() {
		for _, attr := range e.Attributes {
			k := string(attr.Key)
			attrSize := p.calculateAttributeSize(attr)
			totalSize += attrSize

			if attrSize > p.cfg.AttrMaxBytes {
				logx.Errorf(
					"[OTEL-Detector] Big EVENT ATTR detected: span=%s trace=%s event=%s attr=%s size=%d bytes (limit=%d)",
					spanName, traceID, e.Name, k, attrSize, p.cfg.AttrMaxBytes,
				)
			}
		}
	}

	// --- 3. Check resource ---
	res := s.Resource()
	if res != nil {
		for _, attr := range res.Attributes() {
			totalSize += p.calculateAttributeSize(attr)
		}
	}

	// --- 4. Check span total size ---
	if totalSize > p.cfg.SpanMaxBytes {
		logx.Errorf(
			"[OTEL-Detector] Big SPAN detected: span=%s trace=%s totalSize=%d bytes (limit=%d)",
			spanName, traceID, totalSize, p.cfg.SpanMaxBytes,
		)
	}
}

// calculateAttributeSize calculates the size of an attribute value in bytes
func (p *sizeDetectorProcessor) calculateAttributeSize(attr attribute.KeyValue) int {
	key := string(attr.Key)
	keySize := len(key)

	var valueSize int
	switch attr.Value.Type() {
	case attribute.STRING:
		valueSize = len(attr.Value.AsString())
	case attribute.BOOL:
		valueSize = 1 // bool is typically 1 byte
	case attribute.INT64:
		valueSize = 8 // int64 is 8 bytes
	case attribute.FLOAT64:
		valueSize = 8 // float64 is 8 bytes
	case attribute.STRINGSLICE:
		slice := attr.Value.AsStringSlice()
		for _, s := range slice {
			valueSize += len(s)
		}
	case attribute.BOOLSLICE:
		valueSize = len(attr.Value.AsBoolSlice()) // each bool is 1 byte
	case attribute.INT64SLICE:
		valueSize = len(attr.Value.AsInt64Slice()) * 8 // each int64 is 8 bytes
	case attribute.FLOAT64SLICE:
		valueSize = len(attr.Value.AsFloat64Slice()) * 8 // each float64 is 8 bytes
	default:
		// Fallback to string representation
		valueSize = len(fmt.Sprintf("%v", attr.Value.AsInterface()))
	}

	return keySize + valueSize
}
