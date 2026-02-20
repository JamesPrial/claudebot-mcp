package reaction_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/reaction"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_ReactionTools_Registration(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(client, r, filter, nil, nil)

	testutil.AssertRegistrations(t, regs, []string{
		"discord_add_reaction",
		"discord_remove_reaction",
	})
}

// ---------------------------------------------------------------------------
// discord_add_reaction handler
// ---------------------------------------------------------------------------

func Test_AddReaction_Valid(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(client, r, filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_add_reaction")

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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, []string{"general"})

	regs := reaction.ReactionTools(client, r, filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_add_reaction")

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
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	r := testutil.NewMockChannelResolver()
	filter := safety.NewFilter(nil, nil)

	regs := reaction.ReactionTools(client, r, filter, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_remove_reaction")

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
