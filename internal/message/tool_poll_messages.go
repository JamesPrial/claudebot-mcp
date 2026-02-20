package message

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func toolPollMessages(q *queue.Queue, r resolve.ChannelResolver, filter *safety.Filter, audit *safety.AuditLogger, logger *slog.Logger) tools.Registration {
	const toolName = "discord_poll_messages"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Long-poll the message queue for incoming Discord messages."),
		mcp.WithNumber("timeout_seconds",
			mcp.Description("Seconds to wait for messages (default: 30, max: 300)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of messages to return (default: 50)"),
		),
		mcp.WithString("channel",
			mcp.Description("Channel name or ID to filter messages (optional)"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		timeoutSec := req.GetInt("timeout_seconds", 30)
		if timeoutSec <= 0 {
			timeoutSec = 30
		}
		if timeoutSec > 300 {
			timeoutSec = 300
		}

		limit := req.GetInt("limit", 50)
		if limit <= 0 {
			limit = 50
		}

		channel := req.GetString("channel", "")
		params := map[string]any{
			"timeout_seconds": timeoutSec,
			"limit":           limit,
			"channel":         channel,
		}

		// Resolve channel filter if provided.
		var channelFilter string
		if channel != "" {
			resolved, err := resolve.ResolveChannelParam(r, channel)
			if err != nil {
				return tools.AuditErrorResult(audit, toolName, params, err, start), nil
			}
			channelFilter = resolved
			logger.Debug("resolved channel", "input", channel, "channelID", channelFilter)
		}

		msgs := q.Poll(ctx, time.Duration(timeoutSec)*time.Second, limit, channelFilter)
		if len(msgs) == 0 {
			tools.LogAudit(audit, toolName, params, "no messages", start)
			return mcp.NewToolResultText("No new messages"), nil
		}

		tools.LogAudit(audit, toolName, params, fmt.Sprintf("ok: %d messages", len(msgs)), start)
		return tools.JSONResult(msgs), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
