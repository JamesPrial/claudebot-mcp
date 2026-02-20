package message

import (
	"context"
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func toolEditMessage(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_edit_message"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Edit an existing Discord message."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithString("message_id",
			mcp.Required(),
			mcp.Description("ID of the message to edit"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("New message content"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		messageID := req.GetString("message_id", "")
		content := req.GetString("content", "")
		params := map[string]any{
			"channel":    channel,
			"message_id": messageID,
			"content":    content,
		}

		channelID, _, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		if _, err := dg.ChannelMessageEdit(channelID, messageID, content); err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText("Message edited successfully"), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
