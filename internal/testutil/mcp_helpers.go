package testutil

import (
	"strings"
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewCallToolRequest constructs an mcp.CallToolRequest with the given tool name
// and arguments map. This is the standard way to build requests in tests.
func NewCallToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: args,
		},
	}
}

// ExtractText extracts the text string from a CallToolResult. It assumes the
// result contains at least one TextContent element and fails the test otherwise.
func ExtractText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if len(result.Content) == 0 {
		t.Fatal("result has no content elements")
	}
	tc, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("result content[0] is %T, want mcp.TextContent", result.Content[0])
	}
	return tc.Text
}

// AssertTextContains extracts text from the result and asserts it contains substr.
func AssertTextContains(t *testing.T, result *mcp.CallToolResult, substr string) {
	t.Helper()
	text := ExtractText(t, result)
	if !strings.Contains(text, substr) {
		t.Errorf("result text = %q, want it to contain %q", text, substr)
	}
}

// AssertTextNotContains extracts text from the result and asserts it does NOT contain substr.
func AssertTextNotContains(t *testing.T, result *mcp.CallToolResult, substr string) {
	t.Helper()
	text := ExtractText(t, result)
	if strings.Contains(text, substr) {
		t.Errorf("result text = %q, should NOT contain %q", text, substr)
	}
}

// FindHandler searches the given registrations for a tool with the specified
// name and returns its handler. It fails the test if the tool is not found.
func FindHandler(t testing.TB, regs []tools.Registration, name string) server.ToolHandlerFunc {
	t.Helper()
	for _, reg := range regs {
		if reg.Tool.Name == name {
			return reg.Handler
		}
	}
	t.Fatalf("tool %q not found in registrations", name)
	return nil
}

// AssertRegistrations verifies that the given registrations exactly match
// expectedNames: same count, each expected name present, and each handler non-nil.
func AssertRegistrations(t *testing.T, regs []tools.Registration, expectedNames []string) {
	t.Helper()
	if len(regs) != len(expectedNames) {
		t.Fatalf("got %d registrations, want %d", len(regs), len(expectedNames))
	}
	nameSet := make(map[string]bool, len(expectedNames))
	for _, name := range expectedNames {
		nameSet[name] = false
	}
	for _, reg := range regs {
		name := reg.Tool.Name
		if _, ok := nameSet[name]; !ok {
			t.Errorf("unexpected registration: %q", name)
			continue
		}
		nameSet[name] = true
		if reg.Handler == nil {
			t.Errorf("registration %q has nil handler", name)
		}
	}
	for name, found := range nameSet {
		if !found {
			t.Errorf("expected registration %q not found", name)
		}
	}
}

// AssertNotError asserts that the CallToolResult is not an error result.
func AssertNotError(t *testing.T, result *mcp.CallToolResult) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.IsError {
		text := ExtractText(t, result)
		t.Fatalf("expected non-error result, but got IsError=true with text: %s", text)
	}
}
