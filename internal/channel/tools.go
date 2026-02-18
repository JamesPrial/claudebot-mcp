// Package channel provides MCP tool handlers for Discord channel operations.
package channel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ChannelSummary is the response shape for a single Discord channel entry.
type ChannelSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Topic    string `json:"topic,omitempty"`
	Category string `json:"category,omitempty"`
	Position int    `json:"position"`
}

// resolveChannelParam resolves a channel parameter that may be a name or ID.
// All-digit strings are treated as IDs, otherwise looked up via Resolver.
// Strips leading "#" from names.
func resolveChannelParam(r *resolve.Resolver, channel string) (string, error) {
	channel = strings.TrimPrefix(channel, "#")

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

// ChannelTools returns all tool registrations for Discord channel operations.
func ChannelTools(
	dg *discordgo.Session,
	r *resolve.Resolver,
	defaultGuildID string,
	filter *safety.Filter,
	audit *safety.AuditLogger,
) []tools.Registration {
	return []tools.Registration{
		toolGetChannels(dg, defaultGuildID, audit),
		toolTyping(dg, r, filter, audit),
	}
}

func toolGetChannels(dg *discordgo.Session, defaultGuildID string, audit *safety.AuditLogger) tools.Registration {
	const toolName = "discord_get_channels"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("List text channels in a Discord guild."),
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

		rawChannels, err := dg.GuildChannels(guildID)
		if err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		summaries := make([]ChannelSummary, 0, len(rawChannels))
		for _, ch := range rawChannels {
			// Filter to text channels only (Type == 0).
			if ch.Type != discordgo.ChannelTypeGuildText {
				continue
			}
			summaries = append(summaries, ChannelSummary{
				ID:       ch.ID,
				Name:     ch.Name,
				Topic:    ch.Topic,
				Category: ch.ParentID,
				Position: ch.Position,
			})
		}

		tools.LogAudit(audit, toolName, params, fmt.Sprintf("ok: %d channels", len(summaries)), start)
		return tools.JSONResult(summaries), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}

func toolTyping(dg *discordgo.Session, r *resolve.Resolver, filter *safety.Filter, audit *safety.AuditLogger) tools.Registration {
	const toolName = "discord_typing"

	tool := mcp.NewTool(toolName,
		mcp.WithDescription("Send a typing indicator to a Discord channel."),
		mcp.WithString("channel",
			mcp.Required(),
			mcp.Description("Channel name or ID"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		channel := req.GetString("channel", "")
		params := map[string]any{"channel": channel}

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

		if err := dg.ChannelTyping(channelID); err != nil {
			tools.LogAudit(audit, toolName, params, "error: "+err.Error(), start)
			return tools.ErrorResult(err.Error()), nil
		}

		tools.LogAudit(audit, toolName, params, "ok", start)
		return mcp.NewToolResultText("Typing indicator sent"), nil
	}

	return tools.Registration{Tool: tool, Handler: server.ToolHandlerFunc(handler)}
}
