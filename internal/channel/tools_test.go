package channel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/channel"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_ChannelTools_Registration(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)

	if len(regs) != 2 {
		t.Fatalf("ChannelTools() returned %d registrations, want 2", len(regs))
	}

	expectedNames := map[string]bool{
		"discord_get_channels": false,
		"discord_typing":       false,
	}

	for _, reg := range regs {
		name := reg.Tool.Name
		if _, ok := expectedNames[name]; !ok {
			t.Errorf("unexpected tool name %q", name)
			continue
		}
		expectedNames[name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected tool %q not found in registrations", name)
		}
	}
}

func Test_ChannelTools_HandlersNotNil(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)

	for _, reg := range regs {
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Tool.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// discord_get_channels handler
// ---------------------------------------------------------------------------

func Test_GetChannels_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)
	handler := findHandler(t, regs, "discord_get_channels")

	req := testutil.NewCallToolRequest("discord_get_channels", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Mock returns "general" and "random" channels.
	if !strings.Contains(text, "general") {
		t.Errorf("expected result to contain channel 'general', got: %s", text)
	}
	if !strings.Contains(text, "random") {
		t.Errorf("expected result to contain channel 'random', got: %s", text)
	}
}

func Test_GetChannels_JSONFormat(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)
	handler := findHandler(t, regs, "discord_get_channels")

	req := testutil.NewCallToolRequest("discord_get_channels", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Result should contain channel IDs in the JSON output.
	if !strings.Contains(text, "ch-001") && !strings.Contains(text, "ch-002") {
		t.Errorf("expected result to contain channel IDs, got: %s", text)
	}
}

// ---------------------------------------------------------------------------
// discord_typing handler
// ---------------------------------------------------------------------------

func Test_Typing_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)
	handler := findHandler(t, regs, "discord_typing")

	req := testutil.NewCallToolRequest("discord_typing", map[string]any{
		"channel": "123456789012345678",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "error:") {
		t.Errorf("expected success for typing, got: %s", text)
	}
}

func Test_Typing_DeniedChannel(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	filter := safety.NewFilter(nil, []string{"general"})

	regs := channel.ChannelTools(md.Session, r, "test-guild-id", filter, nil)
	handler := findHandler(t, regs, "discord_typing")

	req := testutil.NewCallToolRequest("discord_typing", map[string]any{
		"channel": "general",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "not allowed") && !strings.Contains(lower, "denied") {
		t.Errorf("expected channel denied error, got: %s", text)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func findHandler(t testing.TB, regs []tools.Registration, name string) server.ToolHandlerFunc {
	t.Helper()
	for _, reg := range regs {
		if reg.Tool.Name == name {
			return reg.Handler
		}
	}
	t.Fatalf("tool %q not found in registrations", name)
	return nil
}

// Compile-time type checks.
var (
	_ mcp.CallToolRequest
	_ server.ToolHandlerFunc
)
