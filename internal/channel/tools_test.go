package channel_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/channel"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_ChannelTools_Registration(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(client, r, "test-guild-id", filter, nil, nil)

	testutil.AssertRegistrations(t, regs, []string{
		"discord_get_channels",
		"discord_typing",
	})
}

// ---------------------------------------------------------------------------
// discord_get_channels handler
// ---------------------------------------------------------------------------

func Test_GetChannels_Valid(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(client, r, "test-guild-id", filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_channels")

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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(client, r, "test-guild-id", filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_channels")

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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := channel.ChannelTools(client, r, "test-guild-id", filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_typing")

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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, []string{"general"})

	regs := channel.ChannelTools(client, r, "test-guild-id", filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_typing")

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
