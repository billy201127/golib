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
	Count        int
	File         string
	Line         int
	FuncNameFull string
	FuncName     string
	Message      string
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

		funcFull := record.FuncName
		summaries = append(summaries, summaryItem{
			Count:        record.Count,
			File:         record.File,
			Line:         record.Line,
			FuncNameFull: funcFull,
			FuncName:     simplifyFuncName(funcFull),
			Message:      stripANSI(record.LastMessage),
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

	content := buildMarkdownCard(items)
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

func buildMarkdownCard(items []summaryItem) string {
	var sb strings.Builder

	if len(items) > 0 {
		if host := extractHostname(items[0].Message); host != "" {
			writeKVLine(&sb, "host", host)
			sb.WriteString("\n")
		}
	}

	for i, it := range items {
		if i > 0 {
			sb.WriteString("---\n\n")
		}

		msg, attrs, extras := parseLogMessage(it.Message)

		file := fmt.Sprintf("%s:%d", it.File, it.Line)
		callerPath := truncateCallerPath(file)

		writeKVLine(&sb, "count", fmt.Sprint(it.Count))
		if v := attrs["time"]; v != "" {
			writeKVLine(&sb, "time", v)
		}
		if it.FuncName != "" {
			writeKVLine(&sb, "func", it.FuncName)
		}
		writeKVLine(&sb, "caller", callerPath)
		if v := attrs["trace"]; v != "" {
			writeKVLine(&sb, "trace", v)
		}
		if v := attrs["span"]; v != "" {
			writeKVLine(&sb, "span", v)
		}

		for _, e := range extras {
			sb.WriteString(e)
			sb.WriteString("  \n")
		}

		writeKVLine(&sb, "msg", msg)
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

func writeKVLine(sb *strings.Builder, key, value string) {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key == "" {
		return
	}
	// Markdown line break: "  \n"
	sb.WriteString("**")
	sb.WriteString(key)
	sb.WriteString(":** ")
	sb.WriteString(escapeMarkdownInline(value))
	sb.WriteString("  \n")
}

func escapeMarkdownInline(s string) string {
	// Prevent value from breaking markdown. Keep it readable.
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "`", "\\`")
	s = strings.ReplaceAll(s, "*", "\\*")
	return s
}

func extractHostname(msg string) string {
	start := strings.Index(msg, "hostname: [")
	if start < 0 {
		return ""
	}
	end := strings.Index(msg[start:], "]")
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(msg[start+11 : start+end])
}

func parseLogMessage(s string) (msg string, attrs map[string]string, extras []string) {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "]\n"); idx > 0 {
		s = strings.TrimSpace(s[idx+2:])
	}

	parts := strings.Split(s, "\t")
	if len(parts) < 3 {
		return s, map[string]string{}, nil
	}

	attrs = make(map[string]string, 8)
	seen := make(map[string]struct{}, 16)
	attrs["time"] = strings.TrimSpace(parts[0])

	content := strings.Join(parts[2:], "\t")
	fields := strings.Fields(content)
	mainParts := make([]string, 0, len(fields))
	extras = make([]string, 0, 8)

	for _, f := range fields {
		if !isKVToken(f) {
			mainParts = append(mainParts, f)
			continue
		}

		k, v := splitKVToken(f)
		if k == "" {
			continue
		}

		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}

		switch k {
		case "trace", "span":
			attrs[k] = v
		case "caller":
			// Ignore go-zero caller; we force runtime caller path
			continue
		default:
			extras = append(extras, fmt.Sprintf("**%s:** %s", k, escapeMarkdownInline(v)))
		}
	}

	msg = strings.TrimSpace(strings.Join(mainParts, " "))
	return msg, attrs, extras
}

func isKVToken(s string) bool {
	idx := strings.IndexByte(s, '=')
	return idx > 0 && idx < len(s)-1
}

func splitKVToken(s string) (key, val string) {
	idx := strings.IndexByte(s, '=')
	if idx <= 0 || idx >= len(s)-1 {
		return "", ""
	}
	return s[:idx], s[idx+1:]
}

func simplifyFuncName(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	if idx := strings.LastIndex(full, "/"); idx >= 0 && idx+1 < len(full) {
		return full[idx+1:]
	}
	return full
}

func truncateCallerPath(fileLine string) string {
	idx := strings.Index(fileLine, "microloan/")
	if idx < 0 {
		idx = strings.Index(fileLine, "microloan\\")
	}
	if idx < 0 {
		return fileLine
	}
	return fileLine[idx:]
}

func truncateContent(content string) string {
	if len(content) <= maxNotifyContentLen {
		return content
	}

	truncated := content[:maxNotifyContentLen]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxNotifyContentLen/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + fmt.Sprintf("\n\n[Truncated, size: %d]", len(content))
}

func stripANSI(s string) string {
	var result strings.Builder
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

	parts := strings.Split(cleanMsg, "\t")
	if len(parts) >= 2 {
		level := strings.ToLower(strings.TrimSpace(parts[1]))
		level = strings.Trim(level, "[]")
		switch level {
		case "error", "err", "erro":
			return true
		default:
			return false
		}
	}

	lower := strings.ToLower(cleanMsg)
	if strings.Contains(lower, `"level":"error"`) || strings.Contains(lower, " level=error") {
		return true
	}

	return false
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
			return frame.File, frame.Line, frame.Function
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
	if len(f.allowPrefixes) > 0 && (hasAnyPrefix(function, f.allowPrefixes) || hasAnyPrefix(file, f.allowPrefixes)) {
		return true
	}
	if strings.Contains(file, runtimePathSegment) || (!f.includeStdlib && isStdLibFile(file)) {
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
	return strings.HasPrefix(file, filepath.Join(goroot, "src"))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
