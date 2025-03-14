package rocketmq

import (
	"context"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func NewProducer(appId string, endpoint string) *Producer {
	SetLogger()
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

	// 检查输入的 context 中是否已有 trace
	// if spanCtx := trace.SpanContextFromContext(ctx); spanCtx.IsValid() {
	// 	logx.Infof("Input context trace_id: %s", spanCtx.TraceID().String())
	// }

	ctx, span := otel.Tracer("rocket-producer").Start(ctx, "rocket.Producer.Publish",
		trace.WithAttributes(
			attribute.String("topic", actualTopic),
			attribute.Int("message.size", len(msg)),
		),
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()

	// 使用 W3C trace context 格式
	prop := propagation.TraceContext{}
	carrier := propagation.MapCarrier{}
	prop.Inject(ctx, carrier)

	message := &rmq.Message{
		Topic: actualTopic,
		Body:  msg,
	}

	// 打印要传递的 trace context
	// logx.Infof("Injecting trace context: %+v", carrier)

	// 将 trace context 添加到消息属性
	for k, v := range carrier {
		message.AddProperty(k, v)
	}

	// 为了兼容性，同时保留原有的 trace_id 和 span_id
	message.AddProperty("trace_id", span.SpanContext().TraceID().String())
	message.AddProperty("span_id", span.SpanContext().SpanID().String())

	opt := &PublishOption{
		timeout: 5 * time.Second,
	}
	for _, o := range opts {
		o(opt)
	}

	if opt.ShardingKey != "" {
		message.SetKeys(opt.ShardingKey)
	}

	// 如果设置了延迟时间，设置延迟投递
	if opt.delay > 0 {
		deliveryTime := time.Now().Add(opt.delay)
		message.SetDelayTimestamp(deliveryTime)
		span.SetAttributes(attribute.Int64("delay.ms", opt.delay.Milliseconds()))
	}

	// 使用超时上下文发送消息
	sendCtx, cancel := context.WithTimeout(ctx, opt.timeout)
	defer cancel()

	result, err := p.Send(sendCtx, message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		logx.WithContext(ctx).Errorf("send message failed: %v, topic: %s, msg: %s", err, actualTopic, string(msg))
		return err
	}

	// 设置成功状态和消息ID
	span.SetStatus(codes.Ok, "")
	span.SetAttributes(attribute.String("message.id", result[0].MessageID))

	return nil
}
