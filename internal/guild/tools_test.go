package guild_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/guild"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_GuildTools_Registration(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := guild.GuildTools(md.Session, "test-guild-id", nil, nil)

	if len(regs) != 1 {
		t.Fatalf("GuildTools() returned %d registrations, want 1", len(regs))
	}

	if regs[0].Tool.Name != "discord_get_guild" {
		t.Errorf("expected tool name 'discord_get_guild', got %q", regs[0].Tool.Name)
	}
}

func Test_GuildTools_HandlerNotNil(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := guild.GuildTools(md.Session, "test-guild-id", nil, nil)

	for _, reg := range regs {
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Tool.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// discord_get_guild handler
// ---------------------------------------------------------------------------

func Test_GetGuild_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := guild.GuildTools(md.Session, "guild-1", nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_guild")

	req := testutil.NewCallToolRequest("discord_get_guild", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Mock returns a guild with name "Test Guild" and the resolver's guild ID.
	if !strings.Contains(text, "Test Guild") {
		t.Errorf("expected result to contain guild name 'Test Guild', got: %s", text)
	}
	if !strings.Contains(text, "guild-1") {
		t.Errorf("expected result to contain guild ID 'guild-1', got: %s", text)
	}
}

func Test_GetGuild_JSONFormat(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := guild.GuildTools(md.Session, "test-guild-id", nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_guild")

	req := testutil.NewCallToolRequest("discord_get_guild", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Result should be JSON-formatted.
	if !strings.Contains(text, "{") || !strings.Contains(text, "}") {
		t.Errorf("expected JSON-formatted result, got: %s", text)
	}
}

func Test_GetGuild_ContainsMemberCount(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := guild.GuildTools(md.Session, "test-guild-id", nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_guild")

	req := testutil.NewCallToolRequest("discord_get_guild", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Mock guild has MemberCount=42.
	if !strings.Contains(text, "42") {
		t.Errorf("expected result to contain member count '42', got: %s", text)
	}
}

// Compile-time type checks.
var (
	_ mcp.CallToolRequest
	_ server.ToolHandlerFunc
)
