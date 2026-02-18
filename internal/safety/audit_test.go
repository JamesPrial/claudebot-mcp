package safety

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// NewAuditLogger
// ---------------------------------------------------------------------------

func Test_NewAuditLogger_NilWriter(t *testing.T) {
	t.Parallel()
	logger := NewAuditLogger(nil)
	if logger != nil {
		t.Error("NewAuditLogger(nil) should return nil")
	}
}

func Test_NewAuditLogger_ValidWriter(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)
	if logger == nil {
		t.Fatal("NewAuditLogger with valid writer should return non-nil")
	}
}

// ---------------------------------------------------------------------------
// Log
// ---------------------------------------------------------------------------

func Test_AuditLogger_Log_ValidEntry(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	entry := AuditEntry{
		Timestamp: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC),
		Tool:      "discord_send_message",
		Params:    map[string]any{"channel": "general", "content": "hello"},
		Result:    "success",
		Duration:  100 * time.Millisecond,
	}

	err := logger.Log(entry)
	if err != nil {
		t.Fatalf("Log() unexpected error: %v", err)
	}

	// Should be a single JSON line
	output := buf.String()
	if !strings.HasSuffix(output, "\n") {
		t.Error("Log output should end with newline")
	}

	// Should be valid JSON
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &decoded); err != nil {
		t.Errorf("Log output is not valid JSON: %v\nOutput: %s", err, output)
	}

	// Verify key fields are present
	if decoded["tool"] != "discord_send_message" {
		t.Errorf("tool = %v, want %q", decoded["tool"], "discord_send_message")
	}
	if decoded["result"] != "success" {
		t.Errorf("result = %v, want %q", decoded["result"], "success")
	}
}

func Test_AuditLogger_Log_NilParams(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	entry := AuditEntry{
		Timestamp: time.Now(),
		Tool:      "some_tool",
		Params:    nil,
		Result:    "ok",
		Duration:  time.Second,
	}

	err := logger.Log(entry)
	if err != nil {
		t.Fatalf("Log() with nil params unexpected error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var decoded map[string]any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Log output is not valid JSON: %v", err)
	}

	// nil Params should appear as null in JSON
	if decoded["params"] != nil {
		t.Errorf("params = %v, want null", decoded["params"])
	}
}

func Test_AuditLogger_Log_MultipleEntries(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	entries := []AuditEntry{
		{Timestamp: time.Now(), Tool: "tool1", Params: map[string]any{"k": "v1"}, Result: "r1", Duration: time.Second},
		{Timestamp: time.Now(), Tool: "tool2", Params: map[string]any{"k": "v2"}, Result: "r2", Duration: 2 * time.Second},
		{Timestamp: time.Now(), Tool: "tool3", Params: map[string]any{"k": "v3"}, Result: "r3", Duration: 3 * time.Second},
	}

	for _, e := range entries {
		if err := logger.Log(e); err != nil {
			t.Fatalf("Log() error: %v", err)
		}
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Errorf("line %d is not valid JSON: %v\nLine: %s", i, err, line)
		}
	}
}

// ---------------------------------------------------------------------------
// Log on nil logger (nil receiver safety)
// ---------------------------------------------------------------------------

func Test_AuditLogger_Log_NilReceiver(t *testing.T) {
	t.Parallel()

	var logger *AuditLogger // nil

	// This should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Log on nil AuditLogger panicked: %v", r)
		}
	}()

	_ = logger.Log(AuditEntry{
		Timestamp: time.Now(),
		Tool:      "test",
		Result:    "test",
	})
}

// ---------------------------------------------------------------------------
// Concurrency: 50 concurrent Log calls produce valid non-interleaved JSON
// ---------------------------------------------------------------------------

func Test_AuditLogger_Log_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			entry := AuditEntry{
				Timestamp: time.Now(),
				Tool:      "concurrent_tool",
				Params:    map[string]any{"index": idx},
				Result:    "ok",
				Duration:  time.Millisecond,
			}
			if err := logger.Log(entry); err != nil {
				t.Errorf("goroutine %d: Log() error: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()

	// Every line should be valid JSON
	output := strings.TrimSpace(buf.String())
	lines := strings.Split(output, "\n")
	if len(lines) != numGoroutines {
		t.Fatalf("expected %d lines, got %d", numGoroutines, len(lines))
	}

	for i, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Errorf("line %d is not valid JSON (interleaved?): %v\nLine: %s", i, err, line)
		}
	}
}

// ---------------------------------------------------------------------------
// Benchmark
// ---------------------------------------------------------------------------

func Benchmark_AuditLogger_Log(b *testing.B) {
	var buf bytes.Buffer
	logger := NewAuditLogger(&buf)

	entry := AuditEntry{
		Timestamp: time.Now(),
		Tool:      "benchmark_tool",
		Params:    map[string]any{"key": "value"},
		Result:    "success",
		Duration:  time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = logger.Log(entry)
	}
}
