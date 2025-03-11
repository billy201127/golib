package bus

import (
	"fmt"
	"reflect"
	"sync"
)

type Subscriber interface {
	Subscribe(topic EventTopic, fn interface{}) error
	SubscribeOnce(topic EventTopic, fn interface{}) error
	Unsubscribe(topic EventTopic, handler interface{}) error
}

type Publisher interface {
	Publish(topic EventTopic, args ...interface{}) error
}

type Bus interface {
	Subscriber
	Publisher
}

type eventHandler struct {
	callback reflect.Value
	once     bool
}

type EventBus struct {
	handlers map[EventTopic][]*eventHandler
	mu       sync.RWMutex
}

func (e *EventBus) doSubscribe(topic EventTopic, fn interface{}, handler *eventHandler) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if reflect.TypeOf(fn).Kind() != reflect.Func {
		return fmt.Errorf("%s is not of type reflect.Func", reflect.TypeOf(fn).Kind())
	}
	e.handlers[topic] = append(e.handlers[topic], handler)
	return nil
}

func (e *EventBus) doPublish(handler *eventHandler, args ...interface{}) error {
	result := handler.callback.Call(e.parseArgs(handler, args...))
	err := result[0].Interface()
	if err != nil {
		return err.(error)
	}
	return nil
}

func (e *EventBus) removeHandler(topic EventTopic, idx int) {
	if _, ok := e.handlers[topic]; !ok {
		return
	}
	l := len(e.handlers[topic])

	if !(idx >= 0 && idx < l) {
		return
	}

	copy(e.handlers[topic][idx:], e.handlers[topic][idx+1:])
	e.handlers[topic][l-1] = nil
	e.handlers[topic] = e.handlers[topic][:l-1]
}

func (e *EventBus) findHandlerIdx(topic EventTopic, callback reflect.Value) int {
	if _, ok := e.handlers[topic]; ok {
		for idx, handler := range e.handlers[topic] {
			if handler.callback.Type() == callback.Type() &&
				handler.callback.Pointer() == callback.Pointer() {
				return idx
			}
		}
	}
	return -1
}

func (e *EventBus) parseArgs(callback *eventHandler, args ...interface{}) []reflect.Value {
	funcType := callback.callback.Type()
	parsedArgs := make([]reflect.Value, len(args))
	for i, v := range args {
		if v == nil {
			parsedArgs[i] = reflect.New(funcType.In(i)).Elem()
		} else {
			parsedArgs[i] = reflect.ValueOf(v)
		}
	}

	return parsedArgs
}

func (e *EventBus) Subscribe(topic EventTopic, fn interface{}) error {
	return e.doSubscribe(topic, fn, &eventHandler{reflect.ValueOf(fn), false})
}

func (e *EventBus) SubscribeOnce(topic EventTopic, fn interface{}) error {
	return e.doSubscribe(topic, fn, &eventHandler{reflect.ValueOf(fn), true})
}

func (e *EventBus) Unsubscribe(topic EventTopic, handler interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.handlers[topic]; ok && len(e.handlers[topic]) > 0 {
		e.removeHandler(topic, e.findHandlerIdx(topic, reflect.ValueOf(handler)))
		return nil
	}
	return fmt.Errorf("topic %s doesn't exist", topic)
}

func (e *EventBus) Publish(topic EventTopic, args ...interface{}) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if handlers, ok := e.handlers[topic]; ok && len(handlers) > 0 {
		copyHandlers := make([]*eventHandler, len(handlers))
		copy(copyHandlers, handlers)

		for _, handler := range copyHandlers {
			// if handler.once {
			// e.removeHandler(topic, i)
			// }
			err := e.doPublish(handler, args...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func New() Bus {
	b := &EventBus{
		make(map[EventTopic][]*eventHandler),
		sync.RWMutex{},
	}
	return b
}
