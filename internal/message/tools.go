// Package message provides MCP tool handlers for Discord message operations.
package message

import (
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/discord"
	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
)

// destructiveTools lists the tool names in this package that require
// confirmation before executing.
var destructiveTools = []string{"discord_delete_message"}

// DestructiveToolNames returns a copy of the destructive tool names list.
func DestructiveToolNames() []string {
	out := make([]string, len(destructiveTools))
	copy(out, destructiveTools)
	return out
}

// MessageSummary is the response shape returned by discord_get_messages.
type MessageSummary struct {
	ID             string    `json:"id"`
	AuthorID       string    `json:"author_id"`
	AuthorUsername string    `json:"author_username"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
	ReplyTo        string    `json:"reply_to,omitempty"`
}

// MessageTools returns all tool registrations for Discord message operations.
func MessageTools(
	dg discord.DiscordClient,
	q *queue.Queue,
	r resolve.ChannelResolver,
	filter *safety.Filter,
	confirm *safety.ConfirmationTracker,
	audit *safety.AuditLogger,
	logger *slog.Logger,
) []tools.Registration {
	logger = tools.DefaultLogger(logger)
	return []tools.Registration{
		toolPollMessages(q, r, filter, audit, logger),
		toolSendMessage(dg, r, filter, audit, logger),
		toolGetMessages(dg, r, filter, audit, logger),
		toolEditMessage(dg, r, filter, audit, logger),
		toolDeleteMessage(dg, r, filter, confirm, audit, logger),
	}
}
