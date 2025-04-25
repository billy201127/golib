package rocketmq

import "fmt"

type Topic string

func GetTopicWithPrefix(prefix string, topic Topic) string {
	if prefix == "" {
		return string(topic)
	}

	return fmt.Sprintf("%s_%s", prefix, string(topic))
}
