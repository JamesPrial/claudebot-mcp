// Package tools provides shared helper utilities for MCP tool handlers.
package tools

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/mark3labs/mcp-go/mcp"
)

// JSONResult marshals v to indented JSON and returns an mcp.CallToolResult.
func JSONResult(v any) *mcp.CallToolResult {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("error marshaling result: %v", err))
	}
	return mcp.NewToolResultText(string(data))
}

// ErrorResult returns an mcp.CallToolResult that describes an error condition.
func ErrorResult(msg string) *mcp.CallToolResult {
	return mcp.NewToolResultError(fmt.Sprintf("error: %s", msg))
}

// LogAudit logs a tool invocation to the audit logger, silently ignoring a nil logger.
func LogAudit(audit *safety.AuditLogger, toolName string, params map[string]any, result string, start time.Time) {
	if audit == nil {
		return
	}
	_ = audit.Log(safety.AuditEntry{
		Timestamp: start,
		Tool:      toolName,
		Params:    params,
		Result:    result,
		Duration:  time.Since(start),
	})
}

// ConfirmPrompt issues a confirmation request and returns the prompt result.
func ConfirmPrompt(confirm *safety.ConfirmationTracker, toolName, resource, description string) *mcp.CallToolResult {
	token := confirm.RequestConfirmation(toolName, resource, description)
	return mcp.NewToolResultText(fmt.Sprintf(
		"Confirmation required for %s on %q.\n\n%s\n\nTo proceed, call %s again with confirmation_token=%q.",
		toolName, resource, description, toolName, token,
	))
}

// DefaultLogger returns l if non-nil, otherwise slog.Default().
func DefaultLogger(l *slog.Logger) *slog.Logger {
	if l == nil {
		return slog.Default()
	}
	return l
}

// AuditErrorResult logs the error to the audit logger and returns an ErrorResult.
func AuditErrorResult(audit *safety.AuditLogger, toolName string, params map[string]any, err error, start time.Time) *mcp.CallToolResult {
	LogAudit(audit, toolName, params, "error: "+err.Error(), start)
	return ErrorResult(err.Error())
}

// ResolveAndFilterChannel resolves a channel parameter to an ID and name, then
// checks whether the channel is permitted by the filter. On success it returns
// the channelID, channelName, and a nil errResult. On any failure it returns
// empty strings and a non-nil errResult that should be returned to the caller.
func ResolveAndFilterChannel(
	r resolve.ChannelResolver,
	filter *safety.Filter,
	audit *safety.AuditLogger,
	logger *slog.Logger,
	toolName string,
	channel string,
	params map[string]any,
	start time.Time,
) (channelID string, channelName string, errResult *mcp.CallToolResult) {
	var err error
	channelID, err = resolve.ResolveChannelParam(r, channel)
	if err != nil {
		LogAudit(audit, toolName, params, "error: "+err.Error(), start)
		return "", "", ErrorResult(err.Error())
	}
	logger.Debug("resolved channel", "input", channel, "channelID", channelID)

	name := r.ChannelName(channelID)
	if filter != nil && !filter.IsAllowed(name) {
		logger.Debug("channel access denied", "channel", name)
		LogAudit(audit, toolName, params, "denied", start)
		return "", "", ErrorResult(fmt.Sprintf("access to channel %q is not allowed", name))
	}
	return channelID, name, nil
}
