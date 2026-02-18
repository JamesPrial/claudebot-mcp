// Package message provides MCP tool handlers for Discord message operations.
package message

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// DestructiveTools lists the tool names in this package that require
// confirmation before executing.
var DestructiveTools = []string{"discord_delete_message"}

// MessageSummary is the response shape returned by discord_get_messages.
type MessageSummary struct {
	ID             string    `json:"id"`
	AuthorID       string    `json:"author_id"`
	AuthorUsername string    `json:"author_username"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	ReplyTo        string    `json:"reply_to,omitempty"`
}

// resolveChannelParam resolves a channel parameter that may be a name or ID.
// All-digit strings are treated as IDs, otherwise looked up via Resolver.
// Strips leading "#" from names.
func resolveChannelParam(r *resolve.Resolver, channel string) (string, error) {
	channel = strings.TrimPrefix(channel, "#")

	// All-digit strings are already IDs.
	allDigits := len(channel) > 0
	for _, c := range channel {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return channel, nil
	}

	return r.ChannelID(channel)
}

// MessageTools returns all tool registrations for Discord message operations.
func MessageTools(
	dg *discordgo.Session,
	q *queue.Queue,
	r *resolve.Resolver,
	filter *safety.Filter,
	confirm *safety.ConfirmationTracker,
	audit *safety.AuditLogger,
) []tools.Registration {
	return []tools.Registration{
		toolPollMessages(dg, q, r, filter, audit),
		toolSendMessage(dg, r, filter, audit),
		toolGetMessages(dg, r, filter, audit),
		toolEditMessage(dg, r, filter, audit),
		toolDeleteMessage(dg, r, filter, confirm, audit),
	}
}

func toolPollMessages(dg *discordgo.Session, q *queue.Queue, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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
			resolved, err := resolveChannelParam(r, channel)
			if err != nil {
				tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
				return tools.ErrorResult(err.Error()), nil
			}
			channelFilter = resolved
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

func toolSendMessage(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		data := &discordgo.MessageSend{
			Content: content,
		}
		if replyTo != "" {
			data.Reference = &discordgo.MessageReference{MessageID: replyTo}
		}

		msg, err := dg.ChannelMessageSendComplex(channelID, data)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok: "+msg.ID, start)
		return mcp.NewToolResultText(fmt.Sprintf("Message sent (ID: %s)", msg.ID)), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}

func toolGetMessages(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		rawMsgs, err := dg.ChannelMessages(channelID, limit, before, "", "")
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
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

func toolEditMessage(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		if _, err := dg.ChannelMessageEdit(channelID, messageID, content); err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText("Message edited successfully"), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}

func toolDeleteMessage(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, confirm *safety.ConfirmationTracker, audit *safety.AuditLogger) tools.Registration {
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

		channelID, err := resolveChannelParam(r, channel)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		channelName := r.ChannelName(channelID)
		if filter != nil && !filter.IsAllowed(channelName) {
			tools.LogAudit(audit, toolName, params, "denied", start)
			return tools.ErrorResult(fmt.Sprintf("access to channel %q is not allowed", channelName)), nil
		}

		if !confirm.Confirm(token) {
			desc := fmt.Sprintf("This will permanently delete message %q from channel %q.", messageID, channelName)
			return tools.ConfirmPrompt(confirm, toolName, messageID, desc), nil
		}

		if err := dg.ChannelMessageDelete(channelID, messageID); err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText("Message deleted successfully"), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
