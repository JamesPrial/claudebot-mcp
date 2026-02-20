package tools

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/mark3labs/mcp-go/mcp"
)

// extractText is a test helper that extracts the text string from a
// CallToolResult. It assumes the result contains exactly one TextContent
// element and fails the test otherwise.
func extractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content elements")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("result content[0] is %T, want mcp.TextContent", result.Content[0])
	}
	return tc.Text
}

// ---------------------------------------------------------------------------
// JSONResult
// ---------------------------------------------------------------------------

func Test_JSONResult_Cases(t *testing.T) {
	t.Parallel()

	type namedStruct struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name         string
		input        any
		wantNonNil   bool
		wantContains string
	}{
		{
			name:         "struct with Name field",
			input:        namedStruct{Name: "test"},
			wantNonNil:   true,
			wantContains: "test",
		},
		{
			name:         "nil input produces null",
			input:        nil,
			wantNonNil:   true,
			wantContains: "null",
		},
		{
			name:         "empty map produces empty object",
			input:        map[string]string{},
			wantNonNil:   true,
			wantContains: "{}",
		},
		{
			name:         "map with entries",
			input:        map[string]string{"key": "value"},
			wantNonNil:   true,
			wantContains: "value",
		},
		{
			name:         "slice of ints",
			input:        []int{1, 2, 3},
			wantNonNil:   true,
			wantContains: "1",
		},
		{
			name:         "boolean true",
			input:        true,
			wantNonNil:   true,
			wantContains: "true",
		},
		{
			name:         "string value",
			input:        "hello world",
			wantNonNil:   true,
			wantContains: "hello world",
		},
		{
			name:         "integer value",
			input:        42,
			wantNonNil:   true,
			wantContains: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := JSONResult(tt.input)
			if tt.wantNonNil && result == nil {
				t.Fatal("JSONResult() returned nil, want non-nil")
			}
			if result == nil {
				return
			}

			text := extractText(t, result)
			if !strings.Contains(text, tt.wantContains) {
				t.Errorf("JSONResult() text = %q, want it to contain %q", text, tt.wantContains)
			}
		})
	}
}

func Test_JSONResult_StructFields(t *testing.T) {
	t.Parallel()

	type person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	result := JSONResult(person{Name: "Alice", Age: 30})
	text := extractText(t, result)

	if !strings.Contains(text, `"name": "Alice"`) {
		t.Errorf("expected JSON to contain name field, got: %s", text)
	}
	if !strings.Contains(text, `"age": 30`) {
		t.Errorf("expected JSON to contain age field, got: %s", text)
	}
}

func Test_JSONResult_IsNotError(t *testing.T) {
	t.Parallel()

	result := JSONResult(map[string]string{"ok": "true"})
	if result.IsError {
		t.Error("JSONResult for valid input should not set IsError")
	}
}

// ---------------------------------------------------------------------------
// ErrorResult
// ---------------------------------------------------------------------------

func Test_ErrorResult_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		msg          string
		wantContains string
	}{
		{
			name:         "specific error message",
			msg:          "not found",
			wantContains: "error: not found",
		},
		{
			name:         "empty message",
			msg:          "",
			wantContains: "error: ",
		},
		{
			name:         "message with special characters",
			msg:          "channel #general not accessible",
			wantContains: "error: channel #general not accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := ErrorResult(tt.msg)
			if result == nil {
				t.Fatal("ErrorResult() returned nil")
			}

			text := extractText(t, result)
			if !strings.Contains(text, tt.wantContains) {
				t.Errorf("ErrorResult(%q) text = %q, want it to contain %q", tt.msg, text, tt.wantContains)
			}
		})
	}
}

func Test_ErrorResult_NonNil(t *testing.T) {
	t.Parallel()
	result := ErrorResult("any error")
	if result == nil {
		t.Fatal("ErrorResult() should always return non-nil")
	}
}

// ---------------------------------------------------------------------------
// LogAudit
// ---------------------------------------------------------------------------

func Test_LogAudit_NilLogger(t *testing.T) {
	t.Parallel()

	// LogAudit with nil audit logger should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogAudit with nil logger panicked: %v", r)
		}
	}()

	LogAudit(nil, "test_tool", map[string]any{"key": "value"}, "ok", time.Now())
}

func Test_LogAudit_WritesToLogger(t *testing.T) {
	t.Parallel()

	// Use a real AuditLogger to verify LogAudit actually writes.
	// We need a buffer; we can import bytes indirectly via safety.NewAuditLogger.
	// Actually we need bytes directly for the buffer.
	// Since this is the tools package, we test via the safety package types.
	// The implementation of LogAudit calls audit.Log(...).
	// We just verify it does not error/panic with a real logger.

	// Create a writer that records whether Write was called.
	w := &trackingWriter{}
	logger := safety.NewAuditLogger(w)

	LogAudit(logger, "test_tool", map[string]any{"key": "val"}, "success", time.Now())

	if !w.called {
		t.Error("LogAudit should have written to the audit logger")
	}
}

// trackingWriter is a minimal io.Writer that records whether Write was called.
type trackingWriter struct {
	called bool
}

func (tw *trackingWriter) Write(p []byte) (int, error) {
	tw.called = true
	return len(p), nil
}

// ---------------------------------------------------------------------------
// ConfirmPrompt
// ---------------------------------------------------------------------------

func Test_ConfirmPrompt_ContainsToolName(t *testing.T) {
	t.Parallel()

	tracker := safety.NewConfirmationTracker([]string{"discord_delete_message"})
	result := ConfirmPrompt(tracker, "discord_delete_message", "channel-123", "Delete important message")

	text := extractText(t, result)

	if !strings.Contains(text, "discord_delete_message") {
		t.Errorf("ConfirmPrompt text should contain tool name, got: %s", text)
	}
	if !strings.Contains(text, "channel-123") {
		t.Errorf("ConfirmPrompt text should contain resource, got: %s", text)
	}
	if !strings.Contains(text, "Delete important message") {
		t.Errorf("ConfirmPrompt text should contain description, got: %s", text)
	}
	if !strings.Contains(text, "confirmation_token=") {
		t.Errorf("ConfirmPrompt text should contain confirmation_token, got: %s", text)
	}
}

func Test_ConfirmPrompt_TokenIsConfirmable(t *testing.T) {
	t.Parallel()

	tracker := safety.NewConfirmationTracker([]string{"discord_delete_message"})
	result := ConfirmPrompt(tracker, "discord_delete_message", "res", "desc")

	text := extractText(t, result)

	// Extract the token from the text. It appears after confirmation_token="
	// and before the closing quote.
	const prefix = `confirmation_token="`
	idx := strings.Index(text, prefix)
	if idx < 0 {
		t.Fatalf("could not find confirmation_token in text: %s", text)
	}
	after := text[idx+len(prefix):]
	endIdx := strings.Index(after, `"`)
	if endIdx < 0 {
		t.Fatalf("could not find closing quote for token in text: %s", text)
	}
	token := after[:endIdx]

	if token == "" {
		t.Fatal("extracted token is empty")
	}

	// The token should be confirmable via the tracker.
	if !tracker.Confirm(token) {
		t.Error("token from ConfirmPrompt should be confirmable via the tracker")
	}

	// Second confirm should fail (single-use).
	if tracker.Confirm(token) {
		t.Error("token should be single-use, second Confirm should return false")
	}
}

// ---------------------------------------------------------------------------
// DefaultLogger
// ---------------------------------------------------------------------------

func Test_DefaultLogger_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *slog.Logger
		wantNil  bool // if true, we expect slog.Default() (non-nil)
		wantSame bool // if true, we expect the returned logger to be the same pointer as input
	}{
		{
			name:     "nil input returns slog.Default",
			input:    nil,
			wantSame: false,
		},
		{
			name:     "non-nil input returns same logger",
			input:    slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)),
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := DefaultLogger(tt.input)
			if got == nil {
				t.Fatal("DefaultLogger() returned nil, should always return non-nil")
			}

			if tt.wantSame {
				if got != tt.input {
					t.Error("DefaultLogger() with non-nil input should return the same logger pointer")
				}
			} else {
				// nil input case: should return slog.Default()
				if got != slog.Default() {
					t.Error("DefaultLogger(nil) should return slog.Default()")
				}
			}
		})
	}
}

func Test_DefaultLogger_NilReturnsDefault(t *testing.T) {
	t.Parallel()

	result := DefaultLogger(nil)
	if result == nil {
		t.Fatal("DefaultLogger(nil) should never return nil")
	}
	if result != slog.Default() {
		t.Error("DefaultLogger(nil) should return slog.Default()")
	}
}

func Test_DefaultLogger_NonNilReturnsSame(t *testing.T) {
	t.Parallel()

	custom := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	result := DefaultLogger(custom)
	if result != custom {
		t.Error("DefaultLogger(custom) should return the exact same logger instance")
	}
}

// ---------------------------------------------------------------------------
// AuditErrorResult
// ---------------------------------------------------------------------------

func Test_AuditErrorResult_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		auditLogger  *safety.AuditLogger
		toolName     string
		params       map[string]any
		err          error
		wantContains string
		wantAuditLog bool // whether we expect a write to the audit logger
	}{
		{
			name:         "logs error and returns error result",
			auditLogger:  safety.NewAuditLogger(&bytes.Buffer{}),
			toolName:     "discord_send_message",
			params:       map[string]any{"channel": "general"},
			err:          errors.New("channel not found"),
			wantContains: "error: channel not found",
			wantAuditLog: true,
		},
		{
			name:         "nil audit logger does not panic",
			auditLogger:  nil,
			toolName:     "discord_send_message",
			params:       map[string]any{"channel": "general"},
			err:          errors.New("some error"),
			wantContains: "error: some error",
			wantAuditLog: false,
		},
		{
			name:         "empty params map",
			auditLogger:  safety.NewAuditLogger(&bytes.Buffer{}),
			toolName:     "discord_delete_message",
			params:       map[string]any{},
			err:          errors.New("missing message_id"),
			wantContains: "error: missing message_id",
			wantAuditLog: true,
		},
		{
			name:         "nil params map",
			auditLogger:  safety.NewAuditLogger(&bytes.Buffer{}),
			toolName:     "discord_edit_message",
			params:       nil,
			err:          errors.New("bad request"),
			wantContains: "error: bad request",
			wantAuditLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Should not panic for any input combination.
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("AuditErrorResult panicked: %v", r)
				}
			}()

			start := time.Now()
			result := AuditErrorResult(tt.auditLogger, tt.toolName, tt.params, tt.err, start)

			if result == nil {
				t.Fatal("AuditErrorResult() returned nil, want non-nil")
			}

			text := extractText(t, result)
			if !strings.Contains(text, tt.wantContains) {
				t.Errorf("AuditErrorResult() text = %q, want it to contain %q", text, tt.wantContains)
			}

			// Verify it is marked as an error result.
			if !result.IsError {
				t.Error("AuditErrorResult() should produce a result with IsError=true")
			}
		})
	}
}

func Test_AuditErrorResult_WritesToAuditLog(t *testing.T) {
	t.Parallel()

	w := &trackingWriter{}
	auditLogger := safety.NewAuditLogger(w)

	start := time.Now()
	_ = AuditErrorResult(auditLogger, "discord_send_message", map[string]any{"channel": "general"}, errors.New("test error"), start)

	if !w.called {
		t.Error("AuditErrorResult should write to the audit logger")
	}
}

func Test_AuditErrorResult_AuditEntryContainsError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	auditLogger := safety.NewAuditLogger(&buf)

	start := time.Now()
	_ = AuditErrorResult(auditLogger, "discord_send_message", map[string]any{"channel": "general"}, errors.New("permission denied"), start)

	logged := buf.String()
	if !strings.Contains(logged, "error: permission denied") {
		t.Errorf("audit log entry should contain the error message, got: %s", logged)
	}
	if !strings.Contains(logged, "discord_send_message") {
		t.Errorf("audit log entry should contain the tool name, got: %s", logged)
	}
}

func Test_AuditErrorResult_NilAuditLoggerNoPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AuditErrorResult with nil audit logger panicked: %v", r)
		}
	}()

	result := AuditErrorResult(nil, "test_tool", map[string]any{"key": "val"}, errors.New("oops"), time.Now())
	if result == nil {
		t.Fatal("AuditErrorResult() should return non-nil even with nil audit logger")
	}

	text := extractText(t, result)
	if !strings.Contains(text, "error: oops") {
		t.Errorf("AuditErrorResult() text = %q, want it to contain %q", text, "error: oops")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func Benchmark_JSONResult_Struct(b *testing.B) {
	type sample struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	input := sample{Name: "bench", Value: 42}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = JSONResult(input)
	}
}

func Benchmark_ErrorResult(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ErrorResult("benchmark error")
	}
}

func Benchmark_DefaultLogger_Nil(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultLogger(nil)
	}
}

func Benchmark_DefaultLogger_NonNil(b *testing.B) {
	l := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultLogger(l)
	}
}

func Benchmark_AuditErrorResult(b *testing.B) {
	auditLogger := safety.NewAuditLogger(&bytes.Buffer{})
	params := map[string]any{"channel": "general"}
	err := fmt.Errorf("test error")
	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AuditErrorResult(auditLogger, "discord_send_message", params, err, start)
	}
}
