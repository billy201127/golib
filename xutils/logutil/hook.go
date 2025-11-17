package logutil

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"gomod.pri/golib/notify"
)

type HookWriter struct {
	w        io.Writer
	msgChan  chan errorEvent
	quit     chan struct{}
	records  map[string]*errorRecord
	order    []string
	mu       sync.Mutex
	interval time.Duration
	limit    int
	config   Config
}

type errorEvent struct {
	Fingerprint string
	File        string
	Line        int
	FuncName    string
	Message     string
}

type errorRecord struct {
	Fingerprint string
	File        string
	Line        int
	FuncName    string
	Count       int
	LastMessage string
}

func NewHookWriter(w io.Writer, config Config) *HookWriter {
	hw := &HookWriter{
		w:        w,
		msgChan:  make(chan errorEvent, 1000),
		quit:     make(chan struct{}),
		records:  make(map[string]*errorRecord),
		order:    make([]string, 0),
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
		event := newErrorEvent(msg)
		select {
		case h.msgChan <- event:
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
		case event := <-h.msgChan:
			h.mu.Lock()
			record, ok := h.records[event.Fingerprint]
			if !ok {
				record = &errorRecord{
					Fingerprint: event.Fingerprint,
					File:        event.File,
					Line:        event.Line,
					FuncName:    event.FuncName,
				}
				h.records[event.Fingerprint] = record
				h.order = append(h.order, event.Fingerprint)
			}
			record.Count++
			record.LastMessage = event.Message
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

	if len(h.records) == 0 {
		return
	}

	var summaries []string
	total := len(h.order)
	for i, fingerprint := range h.order {
		if i >= h.limit {
			summaries = append(summaries, fmt.Sprintf("... skipped %d more error fingerprints", total-h.limit))
			break
		}

		record := h.records[fingerprint]
		if record == nil {
			continue
		}
		summaries = append(summaries, fmt.Sprintf("[%d] %s:%d %s - %s", record.Count, record.File, record.Line, record.FuncName, strings.TrimSpace(record.LastMessage)))
	}

	// batch send
	sendNotify(h.config.NotifyWebhook, h.config.NotifySecret, summaries)

	// clear buffer
	h.records = make(map[string]*errorRecord)
	h.order = make([]string, 0)
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

func newErrorEvent(msg string) errorEvent {
	file, line, funcName := captureCaller()
	fingerprint := fmt.Sprintf("%s:%d:%s", file, line, funcName)

	return errorEvent{
		Fingerprint: fingerprint,
		File:        file,
		Line:        line,
		FuncName:    funcName,
		Message:     msg,
	}
}

func captureCaller() (file string, line int, funcName string) {
	const maxDepth = 16
	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			if !more {
				break
			}
			continue
		}

		// 跳过 logutil 自身以及日志库内部的调用栈，找到真正业务代码位置
		if isBusinessFrame(frame.Function, frame.File) {
			return filepath.Base(frame.File), frame.Line, frame.Function
		}

		if !more {
			break
		}
	}

	return "unknown", 0, "unknown"
}

// isBusinessFrame 判断当前帧是否为业务代码帧，而不是日志库/包装器自身
func isBusinessFrame(function, file string) bool {
	// 过滤本包
	if strings.Contains(function, "xutils/logutil") {
		return false
	}

	// 过滤 go-zero logx 相关调用（函数名或路径中包含 logx）
	if strings.Contains(function, "logx.") || strings.Contains(function, "log.(*Logger)") {
		return false
	}
	if strings.Contains(file, "/core/logx/") {
		return false
	}

	return true
}
