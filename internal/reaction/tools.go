// Package reaction provides MCP tool handlers for Discord message reaction operations.
package reaction

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ReactionTools returns all tool registrations for Discord reaction operations.
func ReactionTools(
	dg discord.DiscordClient,
	r resolve.ChannelResolver,
	filter *safety.Filter,
	audit *safety.AuditLogger,
	logger *slog.Logger,
) []tools.Registration {
	logger = tools.DefaultLogger(logger)
	return []tools.Registration{
		toolAddReaction(dg, r, filter, audit, logger),
		toolRemoveReaction(dg, r, filter, audit, logger),
	}
}

func toolAddReaction(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_add_reaction"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Add a reaction emoji to a Discord message."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("ID of the message to react to"),
		),
		mcp.WithString("emoji",
			mcp.Required(),
			mcp.Description("Emoji to add as a reaction (e.g. '👍' or 'custom_emoji:123456')"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		messageID := req.GetString("message_id", "")
		emoji := req.GetString("emoji", "")
		params := map[string]any{
			"channel":    channel,
			"message_id": messageID,
			"emoji":      emoji,
		}

		channelID, _, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		if err := dg.MessageReactionAdd(channelID, messageID, emoji); err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText(fmt.Sprintf("Reaction %q added successfully", emoji)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}

func toolRemoveReaction(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_remove_reaction"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Remove a reaction emoji from a Discord message."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("ID of the message to remove the reaction from"),
		),
		mcp.WithString("emoji",
			mcp.Required(),
			mcp.Description("Emoji to remove (e.g. '👍' or 'custom_emoji:123456')"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		messageID := req.GetString("message_id", "")
		emoji := req.GetString("emoji", "")
		params := map[string]any{
			"channel":    channel,
			"message_id": messageID,
			"emoji":      emoji,
		}

		channelID, _, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		if err := dg.MessageReactionRemove(channelID, messageID, emoji, "@me"); err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText(fmt.Sprintf("Reaction %q removed successfully", emoji)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
