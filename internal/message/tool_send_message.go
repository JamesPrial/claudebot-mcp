package message

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func toolSendMessage(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_send_message"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Send a message to a Discord channel."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Message content to send"),
		),
		mcp.WithString("reply_to",
			mcp.Description("Message ID to reply to (optional)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		content := req.GetString("content", "")
		replyTo := req.GetString("reply_to", "")
		params := map[string]any{
			"channel":  channel,
			"content":  content,
			"reply_to": replyTo,
		}

		channelID, _, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		data := &discordgo.MessageSend{
			Content: content,
		}
		if replyTo != "" {
			data.Reference = &discordgo.MessageReference{MessageID: replyTo}
		}

		msg, err := dg.ChannelMessageSendComplex(channelID, data)
		if err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		tools.LogAudit(audit, toolName, params, "ok: "+msg.ID, start)
		return mcp.NewToolResultText(fmt.Sprintf("Message sent (ID: %s)", msg.ID)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
