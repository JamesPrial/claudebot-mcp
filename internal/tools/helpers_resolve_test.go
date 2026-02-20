package tools_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
)

// setupMockResolver returns a MockChannelResolver pre-populated with
// channels "general" (ch-001) and "random" (ch-002).
func setupMockResolver(t *testing.T) *testutil.MockChannelResolver {
	t.Helper()
	return testutil.NewMockChannelResolver()
}

func Test_ResolveAndFilterChannel_Cases(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	tests := []struct {
		name          string
		filter        *safety.Filter
		audit         *safety.AuditLogger
		channel       string
		wantID        string
		wantName      string
		wantErrResult bool
		wantContains  string // substring expected in error result text
	}{
		{
			name:          "resolve by name with nil filter succeeds",
			filter:        nil,
			audit:         nil,
			channel:       "general",
			wantID:        "ch-001",
			wantName:      "general",
			wantErrResult: false,
		},
		{
			name:          "resolve by numeric ID with nil filter succeeds",
			filter:        nil,
			audit:         nil,
			channel:       "9999999",
			wantID:        "9999999",
			wantName:      "9999999",
			wantErrResult: false,
		},
		{
			name:          "resolve by name with allow filter succeeds",
			filter:        safety.NewFilter([]string{"general"}, nil),
			audit:         nil,
			channel:       "general",
			wantID:        "ch-001",
			wantName:      "general",
			wantErrResult: false,
		},
		{
			name:          "resolve by name with deny filter returns error",
			filter:        safety.NewFilter(nil, []string{"general"}),
			audit:         nil,
			channel:       "general",
			wantID:        "",
			wantName:      "",
			wantErrResult: true,
			wantContains:  "not allowed",
		},
		{
			name:          "resolve by name not in allowlist returns error",
			filter:        safety.NewFilter([]string{"random"}, nil),
			audit:         nil,
			channel:       "general",
			wantID:        "",
			wantName:      "",
			wantErrResult: true,
			wantContains:  "not allowed",
		},
		{
			name:          "resolve unknown channel name returns error",
			filter:        nil,
			audit:         nil,
			channel:       "nonexistent",
			wantID:        "",
			wantName:      "",
			wantErrResult: true,
			wantContains:  "not found",
		},
		{
			name:          "empty filter (both nil slices) allows all",
			filter:        safety.NewFilter(nil, nil),
			audit:         nil,
			channel:       "random",
			wantID:        "ch-002",
			wantName:      "random",
			wantErrResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			params := map[string]any{"channel": tt.channel}

			channelID, channelName, errResult := tools.ResolveAndFilterChannel(
				r, tt.filter, tt.audit, logger,
				"test_tool", tt.channel, params, start,
			)

			if tt.wantErrResult {
				if errResult == nil {
					t.Fatal("expected errResult to be non-nil")
				}
				text := testutil.ExtractText(t, errResult)
				if !strings.Contains(text, tt.wantContains) {
					t.Errorf("errResult text = %q, want it to contain %q", text, tt.wantContains)
				}
				if channelID != "" {
					t.Errorf("channelID = %q, want empty on error", channelID)
				}
				if channelName != "" {
					t.Errorf("channelName = %q, want empty on error", channelName)
				}
			} else {
				if errResult != nil {
					text := testutil.ExtractText(t, errResult)
					t.Fatalf("expected errResult to be nil, got: %s", text)
				}
				if channelID != tt.wantID {
					t.Errorf("channelID = %q, want %q", channelID, tt.wantID)
				}
				if channelName != tt.wantName {
					t.Errorf("channelName = %q, want %q", channelName, tt.wantName)
				}
			}
		})
	}
}

func Test_ResolveAndFilterChannel_AuditOnResolveError(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)

	var buf bytes.Buffer
	auditLogger := safety.NewAuditLogger(&buf)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	start := time.Now()
	params := map[string]any{"channel": "nonexistent"}

	_, _, errResult := tools.ResolveAndFilterChannel(
		r, nil, auditLogger, logger,
		"test_tool", "nonexistent", params, start,
	)

	if errResult == nil {
		t.Fatal("expected errResult for unknown channel")
	}
	if buf.Len() == 0 {
		t.Error("expected audit logger to be written to on resolve error")
	}
}

func Test_ResolveAndFilterChannel_AuditOnFilterDenial(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)

	var buf bytes.Buffer
	auditLogger := safety.NewAuditLogger(&buf)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	filter := safety.NewFilter(nil, []string{"general"}) // deny general

	start := time.Now()
	params := map[string]any{"channel": "general"}

	_, _, errResult := tools.ResolveAndFilterChannel(
		r, filter, auditLogger, logger,
		"test_tool", "general", params, start,
	)

	if errResult == nil {
		t.Fatal("expected errResult for denied channel")
	}
	if buf.Len() == 0 {
		t.Error("expected audit logger to be written to on filter denial")
	}

	text := testutil.ExtractText(t, errResult)
	if !strings.Contains(text, "not allowed") {
		t.Errorf("errResult text = %q, want it to contain 'not allowed'", text)
	}
}

func Test_ResolveAndFilterChannel_NilAuditLoggerNoPanic(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	defer func() {
		if rec := recover(); rec != nil {
			t.Errorf("ResolveAndFilterChannel with nil audit logger panicked: %v", rec)
		}
	}()

	// Test with resolve error (unknown channel) and nil audit logger.
	start := time.Now()
	_, _, errResult := tools.ResolveAndFilterChannel(
		r, nil, nil, logger,
		"test_tool", "nonexistent", map[string]any{"channel": "nonexistent"}, start,
	)
	if errResult == nil {
		t.Fatal("expected errResult for unknown channel")
	}

	// Test with filter denial and nil audit logger.
	filter := safety.NewFilter(nil, []string{"general"})
	_, _, errResult2 := tools.ResolveAndFilterChannel(
		r, filter, nil, logger,
		"test_tool", "general", map[string]any{"channel": "general"}, start,
	)
	if errResult2 == nil {
		t.Fatal("expected errResult for denied channel")
	}
}

func Test_ResolveAndFilterChannel_NumericIDPassesThrough(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	start := time.Now()
	channelID, _, errResult := tools.ResolveAndFilterChannel(
		r, nil, nil, logger,
		"test_tool", "9999999", map[string]any{"channel": "9999999"}, start,
	)
	if errResult != nil {
		text := testutil.ExtractText(t, errResult)
		t.Fatalf("expected nil errResult for numeric channel ID, got: %s", text)
	}
	if channelID != "9999999" {
		t.Errorf("channelID = %q, want %q", channelID, "9999999")
	}
}

func Test_ResolveAndFilterChannel_HashPrefixStripped(t *testing.T) {
	t.Parallel()
	r := setupMockResolver(t)

	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	start := time.Now()
	channelID, channelName, errResult := tools.ResolveAndFilterChannel(
		r, nil, nil, logger,
		"test_tool", "#general", map[string]any{"channel": "#general"}, start,
	)
	if errResult != nil {
		text := testutil.ExtractText(t, errResult)
		t.Fatalf("expected nil errResult for #general, got: %s", text)
	}
	if channelID != "ch-001" {
		t.Errorf("channelID = %q, want %q", channelID, "ch-001")
	}
	if channelName != "general" {
		t.Errorf("channelName = %q, want %q", channelName, "general")
	}
}
