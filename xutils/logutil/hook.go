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

const (
	captureBaseSkip     = 3
	defaultStackDepth   = 32
	defaultIntervalSec  = 60
	runtimePathSegment  = "/runtime/"
	maxNotifyContentLen = 20000
)

var (
	defaultSkipPrefixes = []string{
		"gomod.pri/golib/xutils/logutil",
		"github.com/zeromicro/go-zero/core/",
		"golang.org/x/",
	}

	defaultAllowPrefixes = []string{
		"microloan",
		"gomod.pri",
	}
)

type HookWriter struct {
	w        io.Writer
	msgChan  chan errorEvent
	quit     chan struct{}
	records  map[string]*errorRecord
	order    []string
	mu       sync.Mutex
	once     sync.Once
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

	runtime.SetFinalizer(hw, func(h *HookWriter) {
		h.Close()
	})

	go hw.runNotifier()

	return hw
}

func (h *HookWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	if isErrorLevelLog(msg) {
		event := h.newErrorEvent(msg)
		select {
		case h.msgChan <- event:
		default:
		}
	}

	return h.w.Write(p)
}

func (h *HookWriter) Close() {
	h.once.Do(func() {
		close(h.quit)
	})
}

func (h *HookWriter) runNotifier() {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case event := <-h.msgChan:
			h.handleEvent(event)

		case <-ticker.C:
			h.flush()

		case <-h.quit:
			h.flush()
			return
		}
	}
}

func (h *HookWriter) handleEvent(event errorEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

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
}

func (h *HookWriter) flush() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.records) == 0 {
		return
	}

	summaries := h.buildSummaries()
	sendNotifyMarkdown(h.config.NotifyChannel, h.config.NotifyWebhook, h.config.NotifySecret, summaries)

	h.records = make(map[string]*errorRecord)
	h.order = make([]string, 0)
}

type summaryItem struct {
	Count    int
	File     string
	Line     int
	FuncName string
	Body     string
}

func (h *HookWriter) buildSummaries() []summaryItem {
	total := len(h.order)
	capSize := total
	if h.limit > 0 {
		capSize = minInt(h.limit, total)
	}
	summaries := make([]summaryItem, 0, capSize)

	for i, fingerprint := range h.order {
		if h.limit > 0 && i >= h.limit {
			break
		}

		record := h.records[fingerprint]
		if record == nil {
			continue
		}

		body := strings.TrimSpace(formatPlainMessage(stripANSI(record.LastMessage)))
		summaries = append(summaries, summaryItem{
			Count:    record.Count,
			File:     record.File,
			Line:     record.Line,
			FuncName: record.FuncName,
			Body:     body,
		})
	}

	return summaries
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

func sendNotifyMarkdown(channel, webhook, secret string, items []summaryItem) {
	if len(items) == 0 {
		return
	}

	notifyChannel := parseNotifyChannel(channel)
	robot, err := notify.NewNotification(notify.NotificationConfig{
		Type: notifyChannel,
		Config: notify.Config{
			Webhook: webhook,
			Secret:  secret,
		},
	})
	if err != nil {
		logx.Errorf("[sendNotify] failed to create robot: %v", err)
		return
	}

	content := buildMarkdownContent(items, notifyChannel)
	content = truncateContent(content)
	if err := robot.SendCard(context.Background(), "Error Alert", content); err != nil {
		logx.Errorf("[sendNotify] failed to send markdown card: %v", err)
	}
}

func parseNotifyChannel(channel string) notify.NotificationType {
	switch channel {
	case "dingtalk":
		return notify.DingTalk
	case "feishu":
		return notify.Feishu
	default:
		return notify.DingTalk
	}
}

func buildMarkdownContent(items []summaryItem, channel notify.NotificationType) string {
	var sb strings.Builder
	for i, it := range items {
		if i > 0 {
			sb.WriteString("\n\n")
		}

		// Keep markdown simple for Feishu.
		if channel == notify.Feishu {
			sb.WriteString(fmt.Sprintf("- count=%d\n", it.Count))
			sb.WriteString(fmt.Sprintf("  loc=%s:%d\n", it.File, it.Line))
			sb.WriteString(fmt.Sprintf("  func=%s\n", it.FuncName))
			sb.WriteString("  ```\n")
			sb.WriteString(escapeCodeBlock(it.Body))
			sb.WriteString("\n  ```")
			continue
		}

		// DingTalk supports a bit more markdown, but still keep it tidy.
		sb.WriteString(fmt.Sprintf("- **Count**: %d\n", it.Count))
		sb.WriteString(fmt.Sprintf("  **Location**: %s:%d\n", it.File, it.Line))
		sb.WriteString(fmt.Sprintf("  **Func**: %s\n", it.FuncName))
		sb.WriteString("  ```\n")
		sb.WriteString(escapeCodeBlock(it.Body))
		sb.WriteString("\n  ```")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func escapeCodeBlock(s string) string {
	// Prevent breaking out of fenced code blocks.
	return strings.ReplaceAll(s, "```", "``\u200b`")
}

func formatPlainMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return ""
	}

	parts := strings.Split(msg, "\t")
	if len(parts) < 3 {
		return msg
	}

	timestamp := strings.TrimSpace(parts[0])
	level := strings.ToUpper(strings.TrimSpace(parts[1]))
	content := strings.Join(parts[2:], "\t")

	mainMsg, kv := splitKV(content)

	var sb strings.Builder
	sb.WriteString(timestamp)
	sb.WriteString(" [")
	sb.WriteString(level)
	sb.WriteString("]\n")
	sb.WriteString(mainMsg)

	for _, p := range kv {
		sb.WriteByte('\n')
		sb.WriteString(p)
	}

	return strings.TrimRight(sb.String(), "\n")
}

func splitKV(s string) (string, []string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}

	idx := strings.Index(s, " caller=")
	if idx < 0 {
		idx = strings.Index(s, "\tcaller=")
	}
	if idx < 0 {
		idx = strings.Index(s, "caller=")
		if idx != 0 {
			idx = -1
		}
	}
	if idx < 0 {
		return s, nil
	}

	main := strings.TrimSpace(s[:idx])
	attrs := strings.TrimSpace(s[idx:])

	fields := strings.Fields(attrs)
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f)
	}
	return main, out
}

func truncateContent(content string) string {
	if len(content) <= maxNotifyContentLen {
		return content
	}

	truncated := content[:maxNotifyContentLen]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxNotifyContentLen/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + fmt.Sprintf("\n\n[Message truncated, original size: %d bytes]", len(content))
}

func stripANSI(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

func isErrorLevelLog(msg string) bool {
	cleanMsg := stripANSI(msg)
	lower := strings.ToLower(cleanMsg)

	if strings.Contains(lower, `"level":"error"`) || strings.Contains(lower, " level=error") {
		return true
	}

	fields := strings.Fields(cleanMsg)
	if len(fields) >= 2 {
		level := strings.Trim(fields[1], "[]")
		level = strings.ToLower(level)
		switch level {
		case "error", "err", "erro":
			return true
		}
	}

	return hasStandaloneError(lower)
}

func hasStandaloneError(lower string) bool {
	if idx := strings.Index(lower, " error"); idx >= 0 {
		if (idx == 0 || isWhitespace(lower[idx-1])) &&
			(idx+6 >= len(lower) || isWhitespace(lower[idx+6])) {
			return true
		}
	}

	if idx := strings.Index(lower, "\terror"); idx >= 0 {
		if idx+7 >= len(lower) || isWhitespace(lower[idx+7]) {
			return true
		}
	}

	return false
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

type frameFilter struct {
	allowPrefixes []string
	skipPrefixes  []string
	includeStdlib bool
}

func newFrameFilter() *frameFilter {
	return &frameFilter{
		allowPrefixes: append([]string(nil), defaultAllowPrefixes...),
		skipPrefixes:  append([]string(nil), defaultSkipPrefixes...),
	}
}

func (f *frameFilter) captureCaller() (file string, line int, funcName string) {
	pcs := make([]uintptr, defaultStackDepth)
	n := runtime.Callers(captureBaseSkip, pcs)
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

	if hasAnyPrefix(function, f.skipPrefixes) || hasAnyPrefix(file, f.skipPrefixes) {
		return false
	}

	if len(f.allowPrefixes) > 0 &&
		(hasAnyPrefix(function, f.allowPrefixes) || hasAnyPrefix(file, f.allowPrefixes)) {
		return true
	}

	if strings.Contains(file, runtimePathSegment) {
		return false
	}

	if !f.includeStdlib && isStdLibFile(file) {
		return false
	}

	return true
}

func hasAnyPrefix(target string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if prefix != "" && strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

func isStdLibFile(file string) bool {
	// NOTE: runtime.GOROOT() is deprecated but still fine for a heuristic filter.
	goroot := runtime.GOROOT()
	if goroot == "" {
		return strings.Contains(file, "/src/")
	}

	src := filepath.Join(goroot, "src")
	return strings.HasPrefix(file, src)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
