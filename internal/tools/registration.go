// Package tools provides shared types and helpers for registering MCP tools
// on an MCP server instance.
package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Registration pairs an MCP tool definition with its handler function.
type Registration struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

// RegisterAll adds every Registration in the provided slice to the given MCP
// server.
func RegisterAll(s *server.MCPServer, registrations []Registration) {
	for _, r := range registrations {
		s.AddTool(r.Tool, r.Handler)
	}
}
