package testutil

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
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
