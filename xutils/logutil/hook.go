package logutil

import (
	"context"
	"encoding/json"
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

// HookWriter is an io.Writer implementation that aggregates error logs
// and periodically sends summarized notifications to configured webhooks.
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
// It starts a background goroutine to periodically flush aggregated errors.
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

	// NOTE: Cannot use logx here as it would cause infinite recursion!
	if isErrorLevelLog(msg) {
		event := h.newErrorEvent(msg)
		select {
		case h.msgChan <- event:
			// Event sent successfully
		default:
			// Channel full, drop message to avoid blocking
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

	summaries := h.buildSummaries()
	sendNotify(h.config.NotifyChannel, h.config.NotifyWebhook, h.config.NotifySecret, summaries)

	// Clear buffer
	h.records = make(map[string]*errorRecord)
	h.order = make([]string, 0)
}

func (h *HookWriter) buildSummaries() []string {
	total := len(h.order)
	summaries := make([]string, 0, min(h.limit+1, total))

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

		summary := fmt.Sprintf(
			"[%d] %s:%d\n%s\n%s",
			record.Count,
			record.File,
			record.Line,
			record.FuncName,
			strings.TrimSpace(record.LastMessage),
		)
		summaries = append(summaries, summary)
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

// sendNotify sends aggregated error summaries to the configured webhook.
// It automatically detects JSON vs plain text format and uses appropriate notification method.
func sendNotify(channel, webhook, secret string, msgs []string) {
	if len(msgs) == 0 {
		return
	}

	var notifyChannel notify.NotificationType
	switch channel {
	case "dingtalk":
		notifyChannel = notify.DingTalk
	case "feishu":
		notifyChannel = notify.Feishu
	default:
		notifyChannel = notify.DingTalk
	}

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

	if hasJSONFormat(msgs) {
		sendJSONNotification(robot, msgs)
	} else {
		sendPlainNotification(robot, msgs)
	}
}

// hasJSONFormat checks if any message contains JSON format.
func hasJSONFormat(msgs []string) bool {
	for _, msg := range msgs {
		if isJSONFormat(msg) {
			return true
		}
	}
	return false
}

// sendJSONNotification sends notifications in markdown card format for JSON logs.
func sendJSONNotification(robot notify.Notification, msgs []string) {
	var sb strings.Builder
	sb.Grow(len(msgs) * 200) // Pre-allocate buffer

	for _, msg := range msgs {
		title, body := formatLogMessageParts(msg)
		sb.WriteString("- ")
		if title != "" {
			sb.WriteString(title)
		} else {
			sb.WriteString("日志")
		}
		sb.WriteString(":\n```\n")
		sb.WriteString(body)
		sb.WriteString("\n```\n")
	}

	// Trim trailing newlines to avoid rendering lots of blank lines in markdown
	content := truncateContent(strings.TrimRight(sb.String(), "\n"))
	if err := robot.SendCard(context.Background(), "Error Alert", content); err != nil {
		logx.Errorf("[sendNotify] failed to send markdown card: %v", err)
	}
}

// sendPlainNotification sends notifications in plain text format.
func sendPlainNotification(robot notify.Notification, msgs []string) {
	var sb strings.Builder
	sb.Grow(len(msgs) * 150) // Pre-allocate buffer

	for i, msg := range msgs {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Strip ANSI codes and output original message
		cleanMsg := stripANSI(msg)
		sb.WriteString(cleanMsg)
	}

	content := truncateContent(sb.String())
	if err := robot.SendText(context.Background(), content); err != nil {
		logx.Errorf("[sendNotify] failed to send text: %v", err)
	}
}

// truncateContent ensures content is within DingTalk's 20KB limit.
func truncateContent(content string) string {
	if len(content) <= maxNotifyContentLen {
		return content
	}

	truncated := content[:maxNotifyContentLen]
	if lastNewline := strings.LastIndex(truncated, "\n"); lastNewline > maxNotifyContentLen/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + fmt.Sprintf(
		"\n\n[Message truncated, original size: %d bytes]",
		len(content),
	)
}

// isJSONFormat checks if the message contains valid JSON format.
func isJSONFormat(msg string) bool {
	cleanMsg := stripANSI(msg)
	start := strings.Index(cleanMsg, "{")
	end := strings.LastIndex(cleanMsg, "}")
	if start < 0 || end <= start {
		return false
	}

	jsonPart := cleanMsg[start : end+1]
	var testObj map[string]interface{}
	return json.Unmarshal([]byte(jsonPart), &testObj) == nil
}

// formatLogMessageParts parses log message and returns (title, body).
// Title is the non-JSON prefix (e.g., "[count: 1] ..."),
// body is the formatted key: val lines for JSON, or original text for plain format.
func formatLogMessageParts(s string) (string, string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", ""
	}

	jsonStart, jsonEnd, jsonPart := findJSONObject(s)
	if jsonStart >= 0 && jsonEnd > jsonStart {
		if formatted, ok := tryFormatJSONLines(jsonPart); ok {
			prefix := strings.TrimSpace(s[:jsonStart])
			return prefix, formatted
		}
		// JSON parsing failed, treat as plain text
		return "", s
	}

	// No JSON found, treat as plain text
	return "", s
}

// findJSONObject finds the first valid JSON object in the string.
// Returns start index, end index (exclusive), and the JSON substring.
// Returns -1, -1, "" if no valid JSON is found.
func findJSONObject(s string) (int, int, string) {
	start := strings.Index(s, "{")
	if start < 0 {
		return -1, -1, ""
	}

	// Try to find matching closing brace by parsing JSON incrementally
	for end := start + 1; end <= len(s); end++ {
		candidate := s[start:end]
		if !strings.HasSuffix(candidate, "}") {
			continue
		}

		var testObj map[string]interface{}
		if err := json.Unmarshal([]byte(candidate), &testObj); err == nil {
			remaining := strings.TrimSpace(s[end:])
			// If remaining doesn't start with '{', we found complete JSON
			if remaining == "" || !strings.HasPrefix(remaining, "{") {
				return start, end, candidate
			}
		}
	}

	// Try reverse approach: find last '}' and work backwards
	end := strings.LastIndex(s, "}")
	if end > start {
		candidate := s[start : end+1]
		var testObj map[string]interface{}
		if err := json.Unmarshal([]byte(candidate), &testObj); err == nil {
			return start, end + 1, candidate
		}
	}

	return -1, -1, ""
}

// tryFormatJSONLines attempts to parse and format JSON content.
// Returns "key: val" lines (no braces) and true if successful; otherwise "", false.
func tryFormatJSONLines(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || !strings.HasPrefix(s, "{") || !strings.HasSuffix(s, "}") {
		return "", false
	}

	var obj map[string]interface{}
	decoder := json.NewDecoder(strings.NewReader(s))
	decoder.UseNumber() // Preserve number precision
	if err := decoder.Decode(&obj); err != nil {
		return "", false
	}

	// Check if there's any remaining content (should be empty for valid JSON)
	if _, err := decoder.Token(); err == nil {
		// There's more content, not a complete JSON object
		return "", false
	}

	return formatMapKeyVal(obj), true
}

// formatMapKeyVal renders map as "key: val" lines without braces, sorted by key.
func formatMapKeyVal(m map[string]interface{}) string {
	if len(m) == 0 {
		return ""
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.Grow(len(keys) * 30) // Pre-allocate buffer
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString("**")
		sb.WriteString(k)
		sb.WriteString("**: ")
		sb.WriteString(stringifyVal(m[k]))
	}
	return sb.String()
}

// stringifyVal converts interface{} to compact string representation.
func stringifyVal(v interface{}) string {
	if v == nil {
		return "null"
	}

	switch vv := v.(type) {
	case string:
		return vv
	case json.Number:
		return string(vv)
	case bool:
		if vv {
			return "true"
		}
		return "false"
	case float64:
		// Format float to avoid scientific notation for small numbers
		if vv == float64(int64(vv)) {
			return fmt.Sprintf("%.0f", vv)
		}
		return fmt.Sprintf("%g", vv)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(vv)
	case []interface{}:
		return formatArray(vv)
	case map[string]interface{}:
		return formatNestedObject(vv)
	default:
		return formatUnknownType(v)
	}
}

// formatArray formats array as compact JSON.
func formatArray(arr []interface{}) string {
	b, err := json.Marshal(arr)
	if err != nil {
		return fmt.Sprintf("[error: %v]", err)
	}
	return string(b)
}

// formatNestedObject formats nested object as compact JSON.
func formatNestedObject(obj map[string]interface{}) string {
	b, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("{error: %v}", err)
	}
	return string(b)
}

// formatUnknownType attempts to format unknown types as JSON, falls back to string.
func formatUnknownType(v interface{}) string {
	b, err := json.Marshal(v)
	if err == nil {
		return string(b)
	}
	return fmt.Sprint(v)
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var result strings.Builder
	result.Grow(len(s))

	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Found ANSI escape sequence start, skip until 'm'
			j := i + 2
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1 // Skip the 'm' too
				continue
			}
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// isErrorLevelLog determines whether the log message should be treated as an error.
// NOTE: Cannot use logx here as it's called from Write which would cause infinite recursion!
func isErrorLevelLog(msg string) bool {
	cleanMsg := stripANSI(msg)
	lower := strings.ToLower(cleanMsg)

	// Check structured logs: {"level":"error", ...} or level=error
	if strings.Contains(lower, `"level":"error"`) || strings.Contains(lower, " level=error") {
		return true
	}

	// Check plain text format: "timestamp    error  message"
	fields := strings.Fields(cleanMsg)
	if len(fields) >= 2 {
		level := strings.Trim(fields[1], "[]")
		level = strings.ToLower(level)
		switch level {
		case "error", "err", "erro":
			return true
		}
	}

	// Check for "error" as a standalone word with whitespace around it
	return hasStandaloneError(lower)
}

// hasStandaloneError checks if "error" appears as a standalone word.
func hasStandaloneError(lower string) bool {
	// Check for " error " pattern
	if idx := strings.Index(lower, " error"); idx >= 0 {
		if (idx == 0 || isWhitespace(lower[idx-1])) &&
			(idx+6 >= len(lower) || isWhitespace(lower[idx+6])) {
			return true
		}
	}

	// Check for tab-separated format "\terror"
	if idx := strings.Index(lower, "\terror"); idx >= 0 {
		if idx+7 >= len(lower) || isWhitespace(lower[idx+7]) {
			return true
		}
	}

	return false
}

// isWhitespace checks if a character is whitespace.
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// frameFilter decides which stack frame is considered the "business" caller.
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

	// Skip has higher priority than allow
	if hasAnyPrefix(function, f.skipPrefixes) || hasAnyPrefix(file, f.skipPrefixes) {
		return false
	}

	// Explicit allow for business packages
	if len(f.allowPrefixes) > 0 &&
		(hasAnyPrefix(function, f.allowPrefixes) || hasAnyPrefix(file, f.allowPrefixes)) {
		return true
	}

	// Filter out runtime
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
	goroot := runtime.GOROOT()
	if goroot == "" {
		return strings.Contains(file, "/src/")
	}

	src := filepath.Join(goroot, "src")
	return strings.HasPrefix(file, src)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
