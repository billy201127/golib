package rocketmq

import (
	"context"
	"fmt"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func NewProducer(appId string, endpoint string) *Producer {
	producer, err := rmq.NewProducer(&rmq.Config{
		Endpoint:    endpoint,
		Credentials: &credentials.SessionCredentials{},
	})
	if err != nil {
		logx.Errorf("init producer failed: %v", err)
		panic(err)
	}

	err = producer.Start()
	if err != nil {
		logx.Errorf("start producer failed: %v", err)
		panic(err)
	}

	return &Producer{
		Producer: producer,
		appId:    appId,
	}
}

type Producer struct {
	rmq.Producer
	appId string
}

func (p *Producer) Stop() {
	_ = p.GracefulStop()
}

type PublishOption struct {
	delay       time.Duration
	timeout     time.Duration
	ShardingKey string
}

type PublishOptionFunc func(*PublishOption)

func WithDelay(delay time.Duration) PublishOptionFunc {
	return func(opt *PublishOption) {
		opt.delay = delay
	}
}

func WithTimeout(timeout time.Duration) PublishOptionFunc {
	return func(opt *PublishOption) {
		opt.timeout = timeout
	}
}

// use when ensuring order
func WithShardingKey(shardingKey string) PublishOptionFunc {
	return func(opt *PublishOption) {
		opt.ShardingKey = shardingKey
	}
}

func (p *Producer) Publish(ctx context.Context, topic Topic, msg []byte, opts ...PublishOptionFunc) error {
	actualTopic := GetTopicName(string(p.appId), topic)

	ctx, span := otel.Tracer("rocket-producer").Start(ctx, "rocket.Producer.Publish",
		trace.WithAttributes(
			attribute.String("topic", actualTopic),
			attribute.Int("message.size", len(msg)),
		),
	)
	defer span.End()

	opt := &PublishOption{
		timeout: 5 * time.Second,
	}
	for _, o := range opts {
		o(opt)
	}

	message := &rmq.Message{
		Topic: actualTopic,
		Body:  msg,
	}
	if opt.ShardingKey != "" {
		message.SetKeys(opt.ShardingKey)
	}

	// add trace context as message attribute
	traceID := span.SpanContext().TraceID().String()
	spanID := span.SpanContext().SpanID().String()
	message.AddProperty("trace_id", traceID)
	message.AddProperty("span_id", spanID)

	// if delay time is set, set delay delivery
	if opt.delay > 0 {
		deliveryTime := time.Now().Add(opt.delay)
		message.SetDelayTimestamp(deliveryTime)

		span.SetAttributes(attribute.Int64("delay.ms", opt.delay.Milliseconds()))
	}

	// send message with timeout context
	sendCtx, cancel := context.WithTimeout(ctx, opt.timeout)
	fmt.Println(p)
	fmt.Println(message)
	fmt.Println(sendCtx)

	result, err := p.Send(sendCtx, message)
	defer cancel()

	if err != nil {
		logx.WithContext(ctx).Errorf("send message failed: %v, topic: %s, msg: %s", err, actualTopic, string(msg))
		span.RecordError(err)
		logx.WithContext(ctx).Errorf("send message failed: %v", err)
		return err
	}

	// 这里需要添加结果的空值检查
	span.SetAttributes(
		attribute.String("message.id", result[0].MessageID),
	)

	logx.WithContext(ctx).Infof("send message success, messageID: %s", result[0].MessageID)
	return nil
}
