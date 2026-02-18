package tools

import (
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
