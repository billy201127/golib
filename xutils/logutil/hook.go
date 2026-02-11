package logutil

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"sort"
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
	Message  string
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
		summaries = append(summaries, summaryItem{
			Count:    record.Count,
			File:     record.File,
			Line:     record.Line,
			FuncName: record.FuncName,
			Message:  stripANSI(record.LastMessage),
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

	content := buildMarkdownCard(items, notifyChannel)
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

func buildMarkdownCard(items []summaryItem, channel notify.NotificationType) string {
	var sb strings.Builder

	// Attempt to extract hostname from the first message
	if len(items) > 0 {
		if host := extractHostname(items[0].Message); host != "" {
			sb.WriteString("#### Host: ")
			sb.WriteString(host)
			sb.WriteString("\n\n")
		}
	}

	for i, it := range items {
		if i > 0 {
			sb.WriteString("---\n\n")
		}

		// Header: Count and Location
		if channel == notify.Feishu {
			fmt.Fprintf(&sb, "Count: %d | Loc: %s:%d\n", it.Count, it.File, it.Line)
			fmt.Fprintf(&sb, "Func: %s\n", it.FuncName)
		} else {
			fmt.Fprintf(&sb, "**Count**: %d | **Loc**: %s:%d\n", it.Count, it.File, it.Line)
			fmt.Fprintf(&sb, "**Func**: `%s`\n", it.FuncName)
		}

		// Log Body in code block
		msg, kv := parseLogMessage(it.Message)
		sb.WriteString("```text\n")
		sb.WriteString(msg)
		sb.WriteString("\n```\n")

		// Metadata (trace/span/caller) as quotes
		for _, v := range kv {
			sb.WriteString("> ")
			sb.WriteString(v)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
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

func parseLogMessage(s string) (msg string, kv []string) {
	s = strings.TrimSpace(s)
	// Skip hostname prefix if present
	if idx := strings.Index(s, "]\n"); idx > 0 {
		s = strings.TrimSpace(s[idx+2:])
	}

	parts := strings.Split(s, "\t")
	if len(parts) < 3 {
		return s, nil
	}

	// [0] timestamp, [1] level, [2:] content
	header := fmt.Sprintf("%s [%s]", strings.TrimSpace(parts[0]), strings.ToUpper(strings.TrimSpace(parts[1])))
	content := strings.Join(parts[2:], "\t")

	// Split main message and attributes.
	// Rule: tokens that look like "k=v" are attributes; others are part of the human-readable message.
	fields := strings.Fields(content)
	mainParts := make([]string, 0, len(fields))
	attrs := make([]string, 0, 8)

	// Add timestamp to attrs
	attrs = append(attrs, fmt.Sprintf("time=%s", strings.TrimSpace(parts[0])))

	for _, f := range fields {
		if isKVToken(f) {
			attrs = append(attrs, f)
			continue
		}
		mainParts = append(mainParts, f)
	}

	mainMsg := strings.TrimSpace(strings.Join(mainParts, " "))
	attrs = stableOrderAttrs(attrs)

	return mainMsg, attrs
}

func isKVToken(s string) bool {
	idx := strings.Index(s, "=")
	return idx > 0 && idx < len(s)-1
}

func stableOrderAttrs(attrs []string) []string {
	if len(attrs) <= 1 {
		return attrs
	}
	// Sort attributes to keep time/trace/span/caller at a stable position if possible
	sort.Slice(attrs, func(i, j int) bool {
		return getAttrWeight(attrs[i]) < getAttrWeight(attrs[j])
	})
	return attrs
}

func getAttrWeight(s string) int {
	if strings.HasPrefix(s, "time=") {
		return 0
	}
	if strings.HasPrefix(s, "trace=") {
		return 1
	}
	if strings.HasPrefix(s, "span=") {
		return 2
	}
	if strings.HasPrefix(s, "caller=") {
		return 3
	}
	if strings.HasPrefix(s, "app=") {
		return 4
	}
	return 10
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
	return strings.Contains(lower, " error") || strings.Contains(lower, "\terror")
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
