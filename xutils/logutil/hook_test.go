package logutil

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"
)

// testNotifier is a simple stub to capture notifications in tests.
type testNotifier struct {
	mu       sync.Mutex
	messages []string
}

func (n *testNotifier) add(msg string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.messages = append(n.messages, msg)
}

func (n *testNotifier) all() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]string, len(n.messages))
	copy(out, n.messages)
	return out
}

// TestHookWriter_WriteAndFlush verifies that error logs are aggregated and flushed.
func TestHookWriter_WriteAndFlush(t *testing.T) {
	var out bytes.Buffer
	cfg := Config{
		IntervalSec: 1,
		Limit:       10,
	}

	h := NewHookWriter(&out, cfg)
	defer h.Close()

	// write a non-error log, should not be captured by hook
	_, _ = h.Write([]byte("2025-01-01T00:00:00Z info something\n"))

	// write multiple error logs of same fingerprint
	const errorLine = "2025-01-01T00:00:00Z error something bad happened\n"
	const errorLine2 = "2025-01-01T00:00:00Z error something bad happened again\n"

	_, _ = h.Write([]byte(errorLine))
	_, _ = h.Write([]byte(errorLine))
	_, _ = h.Write([]byte(errorLine2))

	// manually flush instead of waiting for ticker
	h.flush()

	// we cannot easily assert exact fingerprint text here, but we can
	// assert that flush produced some output into summaries via notify.
	// Since sendNotify ultimately uses external dependency, here we only
	// check that hook's internal state was cleared.
	if got, want := len(h.records), 0; got != want {
		t.Fatalf("expected records to be cleared after flush, got %d", got)
	}
	if got, want := len(h.order), 0; got != want {
		t.Fatalf("expected order to be cleared after flush, got %d", got)
	}
}

// TestIsErrorLevelLog_Cases checks various log formats.
func TestIsErrorLevelLog_Cases(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want bool
	}{
		{
			name: "plain level in second field",
			msg:  "2025-11-25T14:05:14.798+05:00 error Start URL: /private/forward/send",
			want: true,
		},
		{
			name: "structured json level",
			msg:  `{"time":"2025-11-25T14:05:14.798+05:00","level":"error","msg":"failed"}`,
			want: true,
		},
		{
			name: "kv style level",
			msg:  `time=2025-11-25T14:05:14.798+05:00 level=error msg="failed"`,
			want: true,
		},
		{
			name: "info level should be ignored",
			msg:  "2025-11-25T14:05:14.798+05:00 info normal log",
			want: false,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isErrorLevelLog(tt.msg)
			if got != tt.want {
				t.Fatalf("isErrorLevelLog(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

// TestNewHookWriter_IntervalDefault ensures interval default is applied.
func TestNewHookWriter_IntervalDefault(t *testing.T) {
	var out bytes.Buffer
	h := NewHookWriter(&out, Config{
		IntervalSec: 0,
		Limit:       5,
	})
	defer h.Close()

	if h.interval <= 0 {
		t.Fatalf("expected positive interval, got %v", h.interval)
	}
}

// TestHookWriter_CloseIsIdempotent verifies Close can be called multiple times.
func TestHookWriter_CloseIsIdempotent(t *testing.T) {
	var out bytes.Buffer
	h := NewHookWriter(&out, Config{
		IntervalSec: 1,
		Limit:       1,
	})

	done := make(chan struct{})
	go func() {
		time.Sleep(10 * time.Millisecond)
		h.Close()
		h.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not return in time")
	}
}

// helper function used to test dynamic caller detection.
func helperCaptureCaller(f *frameFilter) (file string, line int, funcName string) {
	return f.captureCaller()
}

// TestFrameFilter_CaptureCaller verifies that captureCaller can dynamically
// find the current test function as the business frame when configured.
func TestFrameFilter_CaptureCaller(t *testing.T) {
	f := &frameFilter{
		allowPrefixes: []string{"gomod.pri/golib/xutils/logutil"},
		skipPrefixes:  nil,
		// include stdlib so that only our allowPrefixes affects the decision.
		includeStdlib: true,
	}

	file, line, funcName := helperCaptureCaller(f)

	if file == "" || line == 0 || funcName == "" {
		t.Fatalf("expected non-empty caller info, got file=%q line=%d func=%q", file, line, funcName)
	}

	if !strings.Contains(funcName, "TestFrameFilter_CaptureCaller") {
		t.Fatalf("expected funcName to contain test function name, got %q", funcName)
	}
}
