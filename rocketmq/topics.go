package rocketmq

import "fmt"

type Topic string

func GetTopicName(appName string, topic Topic) string {
	return fmt.Sprintf("%s_%s", appName, string(topic))
}
