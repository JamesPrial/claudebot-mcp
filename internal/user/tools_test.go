package user_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/jamesprial/claudebot-mcp/internal/user"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_UserTools_Registration(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := user.UserTools(md.Session, nil)

	if len(regs) != 1 {
		t.Fatalf("UserTools() returned %d registrations, want 1", len(regs))
	}

	if regs[0].Tool.Name != "discord_get_user" {
		t.Errorf("expected tool name 'discord_get_user', got %q", regs[0].Tool.Name)
	}
}

func Test_UserTools_HandlerNotNil(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := user.UserTools(md.Session, nil)

	for _, reg := range regs {
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Tool.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// discord_get_user handler
// ---------------------------------------------------------------------------

func Test_GetUser_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := user.UserTools(md.Session, nil)
	handler := findHandler(t, regs, "discord_get_user")

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
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := user.UserTools(md.Session, nil)
	handler := findHandler(t, regs, "discord_get_user")

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
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	regs := user.UserTools(md.Session, nil)
	handler := findHandler(t, regs, "discord_get_user")

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
