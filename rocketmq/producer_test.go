package rocketmq

import (
	"context"
	"testing"
)

func TestProducer_Publish(t *testing.T) {
	producer := NewProducer(&ProducerConfig{
		Endpoint: "127.0.0.1:8081",
		AppId:    "KC",
		Credentials: &SessionCredentials{
			AccessKey:    "KC",
			AccessSecret: "KC",
		},
	})
	err := producer.PublishWithPrefix(context.Background(), Topic("test"), []byte("test"))
	if err != nil {
		t.Fatalf("publish message failed: %v", err)
	}
}
