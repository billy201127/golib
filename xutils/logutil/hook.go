package logutil

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
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

var defaultSkipPrefixes = []string{
	"gomod.pri/golib/xutils/logutil",
	"github.com/zeromicro/go-zero/core/",
	"golang.org/x/",
}

var defaultAllowPrefixes = []string{
	"microloan",
	"gomod.pri",
}

// HookWriter is an io.Writer implementation that aggregates error logs
// and periodically sends summarized notifications.
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

// NewHookWriter creates a new HookWriter with the given config.
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

	// Ensure background goroutine can be reclaimed if HookWriter is GC'ed.
	runtime.SetFinalizer(hw, func(h *HookWriter) {
		h.Close()
	})

	go hw.runNotifier()

	return hw
}

// Write implements io.Writer and intercepts error-level logs to aggregate them.
func (h *HookWriter) Write(p []byte) (n int, err error) {
	msg := string(p)

	if isErrorLevelLog(msg) {
		event := h.newErrorEvent(msg)
		select {
		case h.msgChan <- event:
		default:
			// channel full, drop msg
			logx.Infof("notify channel full, drop msg")
		}
	}

	return h.w.Write(p)
}

// Close stops the background notifier goroutine.
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

	var summaries []string
	total := len(h.order)

	for i, fingerprint := range h.order {
		if i >= h.limit {
			summaries = append(summaries,
				fmt.Sprintf("... skipped %d more error fingerprints", total-h.limit),
			)
			break
		}

		record := h.records[fingerprint]
		if record == nil {
			continue
		}

		summaries = append(
			summaries,
			fmt.Sprintf(
				"[count: %d] %s:%d %s\n%s",
				record.Count,
				record.File,
				record.Line,
				record.FuncName,
				strings.TrimSpace(record.LastMessage),
			),
		)
	}

	sendNotify(h.config.NotifyWebhook, h.config.NotifySecret, summaries)

	// clear buffer
	h.records = make(map[string]*errorRecord)
	h.order = make([]string, 0)
}

func (h *HookWriter) newErrorEvent(msg string) errorEvent {
	file, line, funcName := h.filter.captureCaller()
	hash := hashMessage(msg)
	fingerprint := fmt.Sprintf("%s:%d:%s:%s", file, line, funcName, hash)

	return errorEvent{
		Fingerprint: fingerprint,
		File:        file,
		Line:        line,
		FuncName:    funcName,
		Message:     msg,
	}
}

// sendNotify sends aggregated error summaries to the configured webhook.
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

	// Format messages into Markdown for better readability.
	var sb strings.Builder
	sb.WriteString("### Summary of Errors\n")
	for _, msg := range msgs {
		title, body := formatLogMessageParts(msg)
		sb.WriteString("- ")
		if title != "" {
			sb.WriteString(title)
		} else {
			sb.WriteString("Log")
		}
		sb.WriteString(":\n```\n")
		sb.WriteString(body)
		sb.WriteString("\n```\n")
	}
	content := sb.String()

	// Ensure message size is within DingTalk's 20KB limit.
	if len(content) > maxNotifyContentLen {
		truncated := content[:maxNotifyContentLen]

		// Try to cut on the last newline to avoid breaking messages.
		if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxNotifyContentLen/2 {
			truncated = truncated[:lastNewline]
		}

		content = truncated + fmt.Sprintf(
			"\n\n[msg truncated, original size: %d bytes]",
			len(content),
		)
	}

	if err := robot.SendCard(context.Background(), "Error Alert", content); err != nil {
		logx.Errorf("failed to send notify (markdown card), fallback to text: %v", err)
	}
}

// formatLogMessageParts returns (title, body). Title is the non-JSON prefix (e.g., "[count: 1] ..."),
// body is the formatted key: val lines. If no prefix, title is empty.
// Supports mixed content like "[count: 1] ...\n{...json...}".
func formatLogMessageParts(s string) (string, string) {
	s = strings.TrimSpace(s)

	// Case 1: entire string is JSON
	if formatted, ok := tryFormatJSONLines(s); ok {
		return "", formatted
	}

	// Case 2: find first '{' and last '}' and try to format that segment
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		jsonPart := s[start : end+1]
		if formatted, ok := tryFormatJSONLines(jsonPart); ok {
			head := strings.TrimSpace(s[:start])
			if head != "" {
				return head, formatted
			}
			return "", formatted
		}
	}

	// Fallback: return original
	return "", s
}

// tryFormatJSON attempts to pretty print JSON log content.
// Returns "key: val" lines (no braces) and true if successful; otherwise "", false.
func tryFormatJSONLines(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return "", false
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return "", false
	}
	return formatMapKeyVal(obj), true
}

// formatMapKeyVal renders map as "key: val" lines without braces, sorted by key.
func formatMapKeyVal(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("**")
		sb.WriteString(k)
		sb.WriteString("**")
		sb.WriteString(": ")
		sb.WriteString(stringifyVal(m[k]))
	}
	return sb.String()
}

// stringifyVal converts interface{} to compact string.
func stringifyVal(v interface{}) string {
	switch vv := v.(type) {
	case string:
		return vv
	case float64, bool, int, int64, uint64, json.Number:
		return fmt.Sprint(vv)
	case []interface{}, map[string]interface{}:
		// For nested structures, keep it compact JSON
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	default:
		return fmt.Sprint(v)
	}
}

// isErrorLevelLog determines whether the log message should be treated as an error.
func isErrorLevelLog(msg string) bool {
	lower := strings.ToLower(msg)

	// structured logs like: {"level":"error", ...}
	if strings.Contains(lower, `"level":"error"`) {
		return true
	}
	if strings.Contains(lower, " level=error") {
		return true
	}

	// fallback: second field as level (e.g. "2025-01-01T00:00:00Z error ...")
	fields := strings.Fields(msg)
	if len(fields) >= 2 {
		level := strings.Trim(fields[1], "[]")
		level = strings.ToLower(level)
		switch level {
		case "error", "err", "erro":
			return true
		}
	}

	return false
}

// frameFilter decides which stack frame is considered the "business" caller.
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

func (f *frameFilter) captureCaller() (file string, line int, funcName string) {
	pcs := make([]uintptr, defaultStackDepth)

	// Start from a minimal base skip; business-frame filtering will handle library frames.
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

	// skip has higher priority than allow
	if hasAnyPrefix(function, f.skipPrefixes) || hasAnyPrefix(file, f.skipPrefixes) {
		return false
	}

	// explicit allow for business packages
	if len(f.allowPrefixes) > 0 &&
		(hasAnyPrefix(function, f.allowPrefixes) || hasAnyPrefix(file, f.allowPrefixes)) {
		return true
	}

	// filter out runtime first
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

func hashMessage(msg string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(msg))
	sum := h.Sum(nil)

	// shorter hex to keep fingerprint compact
	return hex.EncodeToString(sum)
}
