package xredis

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type TracingHook struct{}

// buildRedisCommand 构建完整的 Redis 命令字符串
func buildRedisCommand(cmd redis.Cmder) string {
	args := cmd.Args()
	if len(args) == 0 {
		return cmd.Name()
	}

	// 构建命令字符串，包含所有参数
	var parts []string
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			parts = append(parts, v)
		case []byte:
			parts = append(parts, string(v))
		case int64:
			parts = append(parts, fmt.Sprintf("%d", v))
		case float64:
			parts = append(parts, fmt.Sprintf("%f", v))
		case bool:
			parts = append(parts, fmt.Sprintf("%t", v))
		default:
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}

	return strings.Join(parts, " ")
}

// DialHook implements the redis.Hook interface
func (th TracingHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return next(ctx, network, addr)
	}
}

// ProcessHook implements the redis.Hook interface
func (th TracingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		// Start a span before processing the command
		tracer := otel.Tracer("redis")
		spanCtx, span := tracer.Start(ctx, fmt.Sprintf("redis.%s", cmd.Name()))

		// 构建完整的命令字符串
		fullCommand := buildRedisCommand(cmd)

		span.SetAttributes(
			semconv.DBSystemRedis,
			attribute.String("db.statement", fullCommand),
			attribute.String("redis.command", cmd.Name()),
		)

		// Process the command with the new context
		err := next(spanCtx, cmd)

		// Record any errors and end the span
		if err != nil && err != redis.Nil {
			span.RecordError(err)
		}
		span.End()

		return err
	}
}

// ProcessPipelineHook implements the redis.Hook interface
func (th TracingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		// Start a span for the pipeline
		tracer := otel.Tracer("redis")
		spanCtx, span := tracer.Start(ctx, "redis.pipeline")

		// Build a string representation of all commands in the pipeline
		var cmdStrings []string
		for _, cmd := range cmds {
			cmdStrings = append(cmdStrings, buildRedisCommand(cmd))
		}

		span.SetAttributes(
			semconv.DBSystemRedis,
			attribute.Int("db.statement.count", len(cmds)),
			attribute.String("db.statement", strings.Join(cmdStrings, "; ")),
		)

		// Process the pipeline with the new context
		err := next(spanCtx, cmds)

		// Record any errors and end the span
		if err != nil {
			span.RecordError(err)
		}
		span.End()

		return err
	}
}
