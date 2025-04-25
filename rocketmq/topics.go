package rocketmq

import "fmt"

type Topic string

func GetTopicName(prefix string, topic Topic) string {
	if prefix == "" {
		return string(topic)
	}

	return fmt.Sprintf("%s_%s", prefix, string(topic))
}
