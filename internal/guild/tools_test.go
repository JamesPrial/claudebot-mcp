package guild_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/guild"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_GuildTools_Registration(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := guild.GuildTools(client, "test-guild-id", nil, nil)

	testutil.AssertRegistrations(t, regs, []string{
		"discord_get_guild",
	})
}

// ---------------------------------------------------------------------------
// discord_get_guild handler
// ---------------------------------------------------------------------------

func Test_GetGuild_Valid(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := guild.GuildTools(client, "guild-1", nil, nil)
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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := guild.GuildTools(client, "test-guild-id", nil, nil)
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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := guild.GuildTools(client, "test-guild-id", nil, nil)
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
