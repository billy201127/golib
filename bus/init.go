package bus

var globalEventBus Bus

func init() {
	globalEventBus = New()
}

func Subscribe(topic EventTopic, fn interface{}) error {
	return globalEventBus.Subscribe(topic, fn)
}

func Unsubscribe(topic EventTopic, fn interface{}) error {
	return globalEventBus.Unsubscribe(topic, fn)
}

func Publish(topic EventTopic, args ...interface{}) error {
	return globalEventBus.Publish(topic, args...)
}
