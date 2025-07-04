package rocketmq

import (
	"context"
	"encoding/json"
	"errors"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	rmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	v2 "github.com/apache/rocketmq-clients/golang/v5/protocol/v2"
	"github.com/zeromicro/go-zero/core/logx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var (
	// maximum waiting time for receive func
	awaitDuration = time.Second * 5
	// maximum number of messages received at once time
	maxMessageNum int32 = 16
	// invisibleDuration should > 20s
	invisibleDuration = time.Second * 120
)

type ConsumerConfig struct {
	Endpoint      string              `json:"endpoint"`
	Topic         string              `json:"topic"`
	ConsumerGroup string              `json:"consumerGroup"`
	Tags          []string            `json:"tags,optional"`
	Credentials   *SessionCredentials `json:"credentials,optional"`
}
type SessionCredentials struct {
	AccessKey    string `json:"accessKey"`
	AccessSecret string `json:"accessSecret"`
}

type ConsumeHandler[T any] interface {
	Consume(ctx context.Context, message T) error
	ErrorHandler(ctx context.Context, message T, err error)
}

func NewConsumer[T any](conf *ConsumerConfig, handler ConsumeHandler[T]) (*Consumer[T], error) {
	if conf == nil {
		return nil, errors.New("NewRocketMqConsumer config is nil")
	}
	SetLogger()
	opts := []rmq.SimpleConsumerOption{rmq.WithAwaitDuration(awaitDuration)}
	tagsExp := rmq.SUB_ALL
	if len(conf.Tags) > 0 {
		tagsExp = rmq.NewFilterExpression(strings.Join(conf.Tags, "||"))
	}

	opts = append(opts, rmq.WithSubscriptionExpressions(map[string]*rmq.FilterExpression{
		conf.Topic: tagsExp,
	}))

	cfg := &rmq.Config{
		Endpoint:      conf.Endpoint,
		ConsumerGroup: conf.ConsumerGroup,
	}

	if conf.Credentials != nil {
		cfg.Credentials = &credentials.SessionCredentials{
			AccessKey:    conf.Credentials.AccessKey,
			AccessSecret: conf.Credentials.AccessSecret,
		}
	} else {
		cfg.Credentials = &credentials.SessionCredentials{}
	}

	simpleConsumer, err := rmq.NewSimpleConsumer(cfg, opts...)
	if err != nil {
		return nil, err
	}

	if simpleConsumer == nil {
		return nil, errors.New("NewRocketMqConsumer simpleConsumer is nil")
	}

	return &Consumer[T]{consumer: simpleConsumer,
		handler: handler,
		conf:    conf,
		done:    make(chan struct{}),
	}, nil
}

type Consumer[T any] struct {
	conf     *ConsumerConfig
	consumer rmq.SimpleConsumer
	handler  ConsumeHandler[T]
	workers  int
	done     chan struct{}
	wg       sync.WaitGroup
}

func (c *Consumer[T]) Start() {
	if err := c.consumer.Start(); err != nil {
		logx.Errorf("start consumer failed: %v", err)
		return
	}

	if c.workers == 0 {
		c.workers = 1
	}

	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			// 这个 sleep 是必要的，5.x 版本的 proxy 有 bug，导致第一次接收消息失败
			time.Sleep(time.Millisecond * 100)
			c.consume()
		}()
	}

	c.wg.Wait()
}

func (c *Consumer[T]) Stop() {
	close(c.done)
	_ = c.consumer.GracefulStop()
}

func (c *Consumer[T]) consume() {
	tracer := otel.Tracer("rocket-consumer")
	prop := propagation.TraceContext{}

	for {
		select {
		case <-c.done:
			return
		default:
			msgs, err := c.consumer.Receive(context.Background(), maxMessageNum, invisibleDuration)
			if err != nil {
				if rpcErr, ok := err.(*rmq.ErrRpcStatus); ok && v2.Code(rpcErr.Code) == v2.Code_MESSAGE_NOT_FOUND {
					// 消息未找到是正常情况，静默处理并等待
					time.Sleep(awaitDuration)
					continue
				}
				// 只有在非 MESSAGE_NOT_FOUND 的错误情况下才打印日志
				logx.Errorf("receive message failed: %v", err)
				continue
			}

			for _, msg := range msgs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							stack := string(debug.Stack())
							logx.Errorf("panic in message processing: %v\nstack: %s", r, stack)
							// 确保消息被确认，避免重复消费
							if ackErr := c.consumer.Ack(context.Background(), msg); ackErr != nil {
								logx.Errorf("failed to ack message after panic: %v", ackErr)
							}
						}
					}()

					props := msg.GetProperties()
					carrier := propagation.MapCarrier{}
					for k, v := range props {
						carrier[k] = v
					}

					ctx, cancel := context.WithTimeout(context.Background(), invisibleDuration-time.Second*2)
					defer cancel()

					ctx = prop.Extract(ctx, carrier)

					msgCtx, msgSpan := tracer.Start(ctx, "rocket.Consumer.ProcessMessage",
						trace.WithAttributes(
							attribute.String("message.topic", msg.GetTopic()),
							attribute.String("message.id", msg.GetMessageId()),
						),
						trace.WithSpanKind(trace.SpanKindConsumer),
					)
					defer msgSpan.End()

					var data T
					if err = json.Unmarshal(msg.GetBody(), &data); err != nil {
						c.handler.ErrorHandler(msgCtx, data, err)
						msgSpan.RecordError(err)
						msgSpan.SetStatus(codes.Error, err.Error())
						if ackErr := c.consumer.Ack(msgCtx, msg); ackErr != nil {
							msgSpan.RecordError(ackErr)
						}
						return
					}

					if err = c.handler.Consume(msgCtx, data); err != nil {
						c.handler.ErrorHandler(msgCtx, data, err)
						msgSpan.RecordError(err)
						msgSpan.SetStatus(codes.Error, err.Error())
						if ackErr := c.consumer.Ack(msgCtx, msg); ackErr != nil {
							msgSpan.RecordError(ackErr)
						}
						return
					}

					// ack
					if err = c.consumer.Ack(msgCtx, msg); err != nil {
						msgSpan.RecordError(err)
						msgSpan.SetStatus(codes.Error, err.Error())
					} else {
						msgSpan.SetStatus(codes.Ok, "")
					}
				}()
			}
		}
	}
}

func RegisterConsumer[T any](conf *ConsumerConfig, handler ConsumeHandler[T]) *Consumer[T] {
	consumer, err := NewConsumer(conf, handler)
	if err != nil {
		panic(err)
	}

	return consumer
}
