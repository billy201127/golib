package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	// "go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"gomod.pri/golib/rocketmq"
)

type Message struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type IncrConsumer struct {
}

func (c *IncrConsumer) Consume(ctx context.Context, message Message) error {
	logc.Info(ctx, "consume message", message)
	return nil
}

func (c *IncrConsumer) ErrorHandler(ctx context.Context, message Message, err error) {
	logc.Error(ctx, "consume message error", err)
}

func main() {
	// 设置全局 propagator
	// otel.SetTextMapPropagator(propagation.TraceContext{})

	// 创建并设置 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	tracer := otel.Tracer("upstream-service")

	ctx, rootSpan := tracer.Start(context.Background(), "upstream-request",
		trace.WithAttributes(
			attribute.String("service.name", "upstream-service"),
			attribute.String("request.id", "req-123"),
		),
	)
	defer rootSpan.End()

	ctx, msgSpan := tracer.Start(ctx, "send-message")
	defer msgSpan.End()

	fmt.Printf("TraceID: %s\nSpanID: %s\n",
		msgSpan.SpanContext().TraceID().String(),
		msgSpan.SpanContext().SpanID().String(),
	)

	producer := rocketmq.NewProducer(&rocketmq.ProducerConfig{
		Endpoint: "127.0.0.1:8081",
		AppId:    "KC",
		Credentials: &rocketmq.SessionCredentials{
			AccessKey:    "KC",
			AccessSecret: "KC",
		},
	})
	producer.Start()

	msg := Message{
		ID:   "1",
		Data: "hello",
	}

	bts, _ := json.Marshal(&msg)
	logc.Infof(ctx, "send message: %s", string(bts))

	producer.Publish(ctx, "test", bts)

	consumer, err := rocketmq.NewConsumer(&rocketmq.ConsumerConfig{
		Endpoint:      "127.0.0.1:8081",
		Topic:         "KC_test",
		ConsumerGroup: "KC_test",
	}, &IncrConsumer{})

	if err != nil {
		fmt.Println(err)
	}

	consumer.Start()

	time.Sleep(time.Minute)
}
