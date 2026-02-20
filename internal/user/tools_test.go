package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/jamesprial/claudebot-mcp/internal/user"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_UserTools_Registration(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := user.UserTools(client, nil, nil)

	testutil.AssertRegistrations(t, regs, []string{
		"discord_get_user",
	})
}

// ---------------------------------------------------------------------------
// discord_get_user handler
// ---------------------------------------------------------------------------

func Test_GetUser_Valid(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := user.UserTools(client, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_user")

	req := testutil.NewCallToolRequest("discord_get_user", map[string]any{
		"user_id": "user-123",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// Mock returns a user with username "mockuser" and the requested ID.
	if !strings.Contains(text, "mockuser") {
		t.Errorf("expected result to contain username 'mockuser', got: %s", text)
	}
	if !strings.Contains(text, "user-123") {
		t.Errorf("expected result to contain user ID 'user-123', got: %s", text)
	}
}

func Test_GetUser_MissingUserID(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := user.UserTools(client, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_user")

	req := testutil.NewCallToolRequest("discord_get_user", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// When user_id is empty, the handler passes it to the Discord API.
	// The result depends on the API — it may return an error or a user.
	// We just verify the handler doesn't panic and returns a result.
	text := testutil.ExtractText(t, result)
	if text == "" {
		t.Error("expected non-empty result")
	}
}

func Test_GetUser_JSONFormat(t *testing.T) {
	t.Parallel()
	client := &testutil.MockDiscordClient{}
	regs := user.UserTools(client, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_user")

	req := testutil.NewCallToolRequest("discord_get_user", map[string]any{
		"user_id": "user-456",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	// The result should be JSON-formatted (contain braces).
	if !strings.Contains(text, "{") || !strings.Contains(text, "}") {
		t.Errorf("expected JSON-formatted result, got: %s", text)
	}
}
