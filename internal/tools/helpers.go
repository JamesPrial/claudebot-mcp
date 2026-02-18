// Package tools provides shared helper utilities for MCP tool handlers.
package tools

import (
	"encoding/json"
	"fmt"
	"time"

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
