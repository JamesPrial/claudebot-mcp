// Package user provides MCP tool handlers for Discord user operations.
package user

import (
	"context"
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// UserSummary is the response shape returned by discord_get_user.
type UserSummary struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Bot           bool   `json:"bot"`
	AvatarURL     string `json:"avatar_url"`
}

// UserTools returns all tool registrations for Discord user operations.
func UserTools(
	dg discord.DiscordClient,
	audit *safety.AuditLogger,
	logger *slog.Logger,
) []tools.Registration {
	logger = tools.DefaultLogger(logger)
	return []tools.Registration{
		toolGetUser(dg, audit, logger),
	}
}

func toolGetUser(dg discord.DiscordClient, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_get_user"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Retrieve information about a Discord user by their ID."),
		mcp.WithString("user_id",
			mcp.Required(),
			mcp.Description("Discord user ID"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		userID := req.GetString("user_id", "")
		params := map[string]any{"user_id": userID}

		logger.Debug("fetching user info", "userID", userID)

		u, err := dg.User(userID)
		if err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		summary := UserSummary{
			ID:            u.ID,
			Username:      u.Username,
			Discriminator: u.Discriminator,
			Bot:           u.Bot,
			AvatarURL:     u.AvatarURL(""),
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return tools.JSONResult(summary), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
