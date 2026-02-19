// Package reaction provides MCP tool handlers for Discord message reaction operations.
package reaction

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ReactionTools returns all tool registrations for Discord reaction operations.
func ReactionTools(
	dg *discordgo.Session,
	r *resolve.Resolver,
	filter *safety.Filter,
	audit *safety.AuditLogger,
) []tools.Registration {
	return []tools.Registration{
		toolAddReaction(dg, r, filter, audit),
		toolRemoveReaction(dg, r, filter, audit),
	}
}

func toolAddReaction(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolve.ResolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		if err := dg.MessageReactionAdd(channelID, messageID, emoji); err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText(fmt.Sprintf("Reaction %q added successfully", emoji)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}

func toolRemoveReaction(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolve.ResolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		if err := dg.MessageReactionRemove(channelID, messageID, emoji, "@me"); err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText(fmt.Sprintf("Reaction %q removed successfully", emoji)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
