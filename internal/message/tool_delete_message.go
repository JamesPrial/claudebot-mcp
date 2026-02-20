package message

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

func toolDeleteMessage(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, confirm *safety.ConfirmationTracker, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_delete_message"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Delete a Discord message. Requires confirmation."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("ID of the message to delete"),
		),
		mcp.WithString("confirmation_token",
			mcp.Description("Confirmation token returned by a prior call to this tool"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		messageID := req.GetString("message_id", "")
		token := req.GetString("confirmation_token", "")
		params := map[string]any{
			"channel":    channel,
			"message_id": messageID,
		}

		channelID, channelName, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		if !confirm.Confirm(token) {
			logger.Debug("confirmation required", "tool", toolName)
			desc := fmt.Sprintf("This will permanently delete message %q from channel %q.", messageID, channelName)
			return tools.ConfirmPrompt(confirm, toolName, messageID, desc), nil
		}

		if err := dg.ChannelMessageDelete(channelID, messageID); err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText("Message deleted successfully"), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
