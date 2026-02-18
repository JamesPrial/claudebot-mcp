package reaction_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/reaction"
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

func Test_ReactionTools_Registration(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(md.Session, r, filter, nil)

	if len(regs) != 2 {
		t.Fatalf("ReactionTools() returned %d registrations, want 2", len(regs))
	}

	expectedNames := map[string]bool{
		"discord_add_reaction":    false,
		"discord_remove_reaction": false,
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

func Test_ReactionTools_HandlersNotNil(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(md.Session, r, filter, nil)

	for _, reg := range regs {
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Tool.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// discord_add_reaction handler
// ---------------------------------------------------------------------------

func Test_AddReaction_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(md.Session, r, filter, nil)
	handler := findHandler(t, regs, "discord_add_reaction")

	req := testutil.NewCallToolRequest("discord_add_reaction", map[string]any{
		"channel":    "123456789012345678",
		"message_id": "msg-100",
		"emoji":      "thumbsup",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	// Should indicate success.
	if strings.HasPrefix(lower, "error:") {
		t.Errorf("expected success for add_reaction, got: %s", text)
	}
}

func Test_AddReaction_DeniedChannel(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	filter := safety.NewFilter(nil, []string{"general"})

	regs := reaction.ReactionTools(md.Session, r, filter, nil)
	handler := findHandler(t, regs, "discord_add_reaction")

	req := testutil.NewCallToolRequest("discord_add_reaction", map[string]any{
		"channel":    "general",
		"message_id": "msg-100",
		"emoji":      "thumbsup",
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
// discord_remove_reaction handler
// ---------------------------------------------------------------------------

func Test_RemoveReaction_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(md.Session, r, filter, nil)
	handler := findHandler(t, regs, "discord_remove_reaction")

	req := testutil.NewCallToolRequest("discord_remove_reaction", map[string]any{
		"channel":    "123456789012345678",
		"message_id": "msg-100",
		"emoji":      "thumbsup",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "error:") {
		t.Errorf("expected success for remove_reaction, got: %s", text)
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
