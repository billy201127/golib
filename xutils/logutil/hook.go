package logutil

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gomod.pri/golib/notify"
)

type HookWriter struct {
	w        io.Writer
	msgChan  chan string
	quit     chan struct{}
	buffer   []string
	mu       sync.Mutex
	interval time.Duration
	limit    int
	config   Config
}

func NewHookWriter(w io.Writer, config Config) *HookWriter {
	hw := &HookWriter{
		w:        w,
		msgChan:  make(chan string, 1000),
		quit:     make(chan struct{}),
		buffer:   make([]string, 0),
		interval: time.Duration(config.IntervalSec) * time.Second,
		limit:    config.Limit,
		config:   config,
	}

	go hw.runNotifier()
	return hw
}

func (h *HookWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	// only error/fatal
	if strings.Contains(msg, ` error `) {
		select {
		case h.msgChan <- msg:
		default:
			// channel full , drop msg
			logx.Infof("notify channel full, drop msg")
		}
	}

	return h.w.Write(p)
}

func (h *HookWriter) runNotifier() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-h.msgChan:
			h.mu.Lock()
			h.buffer = append(h.buffer, msg)
			h.mu.Unlock()

		case <-ticker.C:
			h.flush()

		case <-h.quit:
			h.flush()
			return
		}
	}
}

func (h *HookWriter) flush() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.buffer) == 0 {
		return
	}

	// limit
	toSend := h.buffer
	if len(toSend) > h.limit {
		toSend = toSend[:h.limit]
		toSend = append(toSend, fmt.Sprintf("... skipped %d more errors", len(h.buffer)-h.limit))
	}

	// batch send
	sendNotify(h.config.NotifyWebhook, h.config.NotifySecret, toSend)

	// clear buffer
	h.buffer = make([]string, 0)
}

func (h *HookWriter) Close() {
	close(h.quit)
}

func sendNotify(webhook, secret string, msgs []string) {
	robot, err := notify.NewNotification(notify.NotificationConfig{
		Type: notify.DingTalk,
		Config: notify.Config{
			Webhook: webhook,
			Secret:  secret,
		},
	})

	if err != nil {
		logx.Errorf("failed to create robot: %v", err)
		return
	}

	robot.SendText(context.Background(), strings.Join(msgs, "\n"))
}
