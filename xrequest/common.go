package xrequest

import (
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"gomod.pri/golib/xerror"

	"gomod.pri/golib/xtrace"
)

const RespCodeOK = 200
const RespCodeMsg = "success"

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	ErrMsg  string `json:"err_msg,omitempty"`
	TraceId string `json:"trace_id,omitempty"`
	Data    T      `json:"data,omitempty"`
}

func NewErrRespWithCtx(ctx context.Context, err error) *Response[any] {
	var ce *xerror.Error

	switch typ := err.(type) {
	case *xerror.Error:
		ce = typ
	default:
		if errors.Is(typ, sqlx.ErrNotFound) {
			ce = xerror.New(xerror.CodeDataNotFound, err)
		} else {
			ce = xerror.New(xerror.CodeInternalError, err)
		}
	}

	resp := &Response[any]{
		Code:    ce.Code(),
		Message: ce.Message(),
		TraceId: xtrace.TraceID(ctx),
		Data:    struct{}{},
	}

	if ce.Cause() != nil {
		resp.ErrMsg = ce.Cause().Error()
	}

	return resp
}

func NewErrLoginFailResp(err error) *Response[any] {
	var ce *xerror.Error
	ce = xerror.New(xerror.CodeUnauthorized, err)

	resp := &Response[any]{
		Code:    ce.Code(),
		Message: ce.Message(),
		Data:    struct{}{},
	}

	if ce.Cause() != nil {
		resp.ErrMsg = ce.Cause().Error()
	}

	return resp
}

func NewDataRespWithCtx(ctx context.Context, data any) *Response[any] {
	return &Response[any]{
		Code:    RespCodeOK,
		Message: RespCodeMsg,
		TraceId: xtrace.TraceID(ctx),
		Data:    data,
	}
}

func NewNoneResp() *Response[any] {
	return &Response[any]{
		Code:    RespCodeOK,
		Message: RespCodeMsg,
	}
}

// NewContext
// Important: Need add 'defer span.End()' after create trace context
func NewContext(request *http.Request, serviceName string, withoutCancel bool) (spanCtx context.Context, span oteltrace.Span) {
	tracer := otel.Tracer(trace.TraceName)
	propagator := otel.GetTextMapPropagator()
	spanName := request.URL.Path

	ctx := request.Context()

	if withoutCancel {
		ctx = context.WithoutCancel(ctx)
	}

	return tracer.Start(
		propagator.Extract(ctx, propagation.HeaderCarrier(request.Header)),
		spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithAttributes(semconv.HTTPServerAttributesFromHTTPRequest(serviceName, spanName, request)...),
	)
}
