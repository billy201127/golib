package rocketmq

import (
	"context"
	"testing"
)

func TestProducer_Publish(t *testing.T) {
	producer := NewProducer("TestTopic", "127.0.0.1:8081")
	err := producer.Publish(context.Background(), Topic("TestTopic"), []byte("test"))
	if err != nil {
		t.Fatalf("publish message failed: %v", err)
	}
}
