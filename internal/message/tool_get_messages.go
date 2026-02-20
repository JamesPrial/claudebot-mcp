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

func toolGetMessages(dg discord.DiscordClient, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_get_messages"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Retrieve recent messages from a Discord channel."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of messages to retrieve (default: 50, max: 100)"),
		),
		mcp.WithString("before",
			mcp.Description("Retrieve messages before this message ID (optional)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		limit := req.GetInt("limit", 50)
		before := req.GetString("before", "")

		if limit <= 0 {
			limit = 50
		}
		if limit > 100 {
			limit = 100
		}

		params := map[string]any{
			"channel": channel,
			"limit":   limit,
			"before":  before,
		}

		channelID, _, errResult := tools.ResolveAndFilterChannel(r, filter, audit, logger, toolName, channel, params, start)
		if errResult != nil {
			return errResult, nil
		}

		rawMsgs, err := dg.ChannelMessages(channelID, limit, before, "", "")
		if err != nil {
			return tools.AuditErrorResult(audit, toolName, params, err, start), nil
		}

		summaries := make([]MessageSummary, 0, len(rawMsgs))
		for _, m := range rawMsgs {
			s := MessageSummary{
				ID:        m.ID,
				Content:   m.Content,
				Timestamp: m.Timestamp,
			}
			if m.Author != nil {
				s.AuthorID = m.Author.ID
				s.AuthorUsername = m.Author.Username
			}
			if m.MessageReference != nil {
				s.ReplyTo = m.MessageReference.MessageID
			}
			summaries = append(summaries, s)
		}

		tools.LogAudit(audit, toolName, params, fmt.Sprintf("ok: %d messages", len(summaries)), start)
		return tools.JSONResult(summaries), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
