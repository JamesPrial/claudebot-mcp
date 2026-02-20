// Package guild provides MCP tool handlers for Discord guild operations.
package guild

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

// GuildSummary is the response shape returned by discord_get_guild.
type GuildSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"member_count"`
	OwnerID     string `json:"owner_id"`
	Description string `json:"description,omitempty"`
}

// GuildTools returns all tool registrations for Discord guild operations.
func GuildTools(
	dg discord.DiscordClient,
	defaultGuildID string,
	audit *safety.AuditLogger,
	logger *slog.Logger,
) []tools.Registration {
	logger = tools.DefaultLogger(logger)
	return []tools.Registration{
		toolGetGuild(dg, defaultGuildID, audit, logger),
	}
}

func toolGetGuild(dg discord.DiscordClient, defaultGuildID string, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_get_guild"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Retrieve information about a Discord guild (server)."),
		mcp.WithString("guild_id",
			mcp.Description("Guild (server) ID (optional, uses default guild if omitted)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		guildID := req.GetString("guild_id", "")
		if guildID == "" {
			guildID = defaultGuildID
		}
		params := map[string]any{"guild_id": guildID}

		logger.Debug("fetching guild info", "guildID", guildID)

		g, err := dg.Guild(guildID)
		if err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		summary := GuildSummary{
			ID:          g.ID,
			Name:        g.Name,
			MemberCount: g.MemberCount,
			OwnerID:     g.OwnerID,
			Description: g.Description,
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return tools.JSONResult(summary), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
