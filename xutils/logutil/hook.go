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
	filter   *frameFilter
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
	filter := newFrameFilter()

	intervalSec := config.IntervalSec
	if intervalSec <= 0 {
		intervalSec = defaultIntervalSec
	}

	hw := &HookWriter{
		w:        w,
		msgChan:  make(chan errorEvent, 1000),
		quit:     make(chan struct{}),
		records:  make(map[string]*errorRecord),
		order:    make([]string, 0),
		interval: time.Duration(intervalSec) * time.Second,
		limit:    config.Limit,
		config:   config,
		filter:   filter,
	}

	go hw.runNotifier()
	return hw
}

func (h *HookWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	// only error/fatal
	if strings.Contains(msg, ` error `) {
		event := h.newErrorEvent(msg)
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
		summaries = append(summaries, fmt.Sprintf("[count: %d] %s:%d %s\n%s", record.Count, record.File, record.Line, record.FuncName, strings.TrimSpace(record.LastMessage)))
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

	if err := robot.SendText(context.Background(), strings.Join(msgs, "\n")); err != nil {
		logx.Errorf("failed to send notify: %v", err)
	}
}

func (h *HookWriter) newErrorEvent(msg string) errorEvent {
	file, line, funcName := h.filter.captureCaller()
	fingerprint := fmt.Sprintf("%s:%d:%s", file, line, funcName)

	return errorEvent{
		Fingerprint: fingerprint,
		File:        file,
		Line:        line,
		FuncName:    funcName,
		Message:     msg,
	}
}

const (
	captureBaseSkip    = 3
	defaultStackDepth  = 32
	defaultAutoSkip    = 4
	defaultIntervalSec = 60
	runtimePathSegment = "/runtime/"
)

type frameFilter struct {
	allowPrefixes []string
	skipPrefixes  []string
	includeStdlib bool
}

func newFrameFilter() *frameFilter {
	filter := &frameFilter{
		allowPrefixes: make([]string, len(defaultAllowPrefixes)),
		skipPrefixes:  make([]string, len(defaultSkipPrefixes)),
	}

	copy(filter.allowPrefixes, defaultAllowPrefixes)
	copy(filter.skipPrefixes, defaultSkipPrefixes)

	return filter
}

var defaultSkipPrefixes = []string{
	"gomod.pri/golib/xutils/logutil",
	"github.com/zeromicro/go-zero/core/",
	"golang.org/x/",
}

var defaultAllowPrefixes = []string{
	"microloan",
	"gomod.pri",
}

func (f *frameFilter) captureCaller() (file string, line int, funcName string) {
	pcs := make([]uintptr, defaultStackDepth)
	skip := captureBaseSkip + defaultAutoSkip
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return "unknown", 0, "unknown"
	}

	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if frame.Function != "" && f.isBusinessFrame(frame.Function, frame.File) {
			return filepath.Base(frame.File), frame.Line, frame.Function
		}

		if !more {
			break
		}
	}

	return "unknown", 0, "unknown"
}

func (f *frameFilter) isBusinessFrame(function, file string) bool {
	if function == "" || file == "" {
		return false
	}

	if len(f.allowPrefixes) > 0 && (hasAnyPrefix(function, f.allowPrefixes) || hasAnyPrefix(file, f.allowPrefixes)) {
		return true
	}

	if strings.Contains(file, runtimePathSegment) {
		return false
	}

	if hasAnyPrefix(function, f.skipPrefixes) || hasAnyPrefix(file, f.skipPrefixes) {
		return false
	}

	if !f.includeStdlib && isStdLibFile(file) {
		return false
	}

	return true
}

func hasAnyPrefix(target string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

func isStdLibFile(file string) bool {
	goroot := runtime.GOROOT()
	if goroot == "" {
		return strings.Contains(file, "/src/")
	}

	src := filepath.Join(goroot, "src")
	return strings.HasPrefix(file, src)
}
