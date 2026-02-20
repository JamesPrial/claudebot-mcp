package message_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jamesprial/claudebot-mcp/internal/message"
	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------------------------------------------------------------------------
// Tool Registration
// ---------------------------------------------------------------------------

func Test_MessageTools_Registration(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker([]string{"discord_delete_message"})

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)

	if len(regs) != 5 {
		t.Fatalf("MessageTools() returned %d registrations, want 5", len(regs))
	}

	expectedNames := map[string]bool{
		"discord_poll_messages":  false,
		"discord_send_message":   false,
		"discord_get_messages":   false,
		"discord_edit_message":   false,
		"discord_delete_message": false,
	}

	for _, reg := range regs {
		name := reg.Tool.Name
		if _, ok := expectedNames[name]; !ok {
			t.Errorf("unexpected tool name %q", name)
			continue
		}
		expectedNames[name] = true
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected tool %q not found in registrations", name)
		}
	}
}

func Test_MessageTools_HandlersNotNil(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker([]string{"discord_delete_message"})

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)

	for _, reg := range regs {
		if reg.Handler == nil {
			t.Errorf("tool %q has nil handler", reg.Tool.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// discord_poll_messages handler
// ---------------------------------------------------------------------------

func Test_PollMessages_EmptyQueue_ShortTimeout(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_poll_messages")

	req := testutil.NewCallToolRequest("discord_poll_messages", map[string]any{
		"timeout_seconds": float64(1),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("handler returned unexpected error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "no new messages") && !strings.Contains(text, "[]") {
		t.Errorf("expected 'no new messages' or empty result, got: %s", text)
	}
}

func Test_PollMessages_QueueHasMessages(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	q.Enqueue(queue.QueuedMessage{
		ID:             "msg-1",
		ChannelID:      "ch-001",
		ChannelName:    "general",
		AuthorID:       "user-1",
		AuthorUsername: "alice",
		Content:        "hello world",
		Timestamp:      time.Now(),
	})

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_poll_messages")

	req := testutil.NewCallToolRequest("discord_poll_messages", map[string]any{
		"timeout_seconds": float64(1),
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	if !strings.Contains(text, "hello world") {
		t.Errorf("expected result to contain message content, got: %s", text)
	}
	if !strings.Contains(text, "alice") {
		t.Errorf("expected result to contain author username, got: %s", text)
	}
}

func Test_PollMessages_TimeoutClamping(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_poll_messages")

	tests := []struct {
		name           string
		timeoutSeconds float64
	}{
		{
			name:           "zero normalized to default (30)",
			timeoutSeconds: 0,
		},
		{
			name:           "very large capped at 300",
			timeoutSeconds: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a short context to prevent actually waiting the full duration.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			req := testutil.NewCallToolRequest("discord_poll_messages", map[string]any{
				"timeout_seconds": tt.timeoutSeconds,
			})

			// Verify it returns without panicking and respects context cancellation.
			_, err := handler(ctx, req)
			if err != nil {
				t.Fatalf("handler error: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// discord_send_message handler
// ---------------------------------------------------------------------------

func Test_SendMessage_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_send_message")

	req := testutil.NewCallToolRequest("discord_send_message", map[string]any{
		"channel": "123456789012345678",
		"content": "test message",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "sent") && !strings.Contains(text, "mock-msg-001") {
		t.Errorf("expected success response with message ID, got: %s", text)
	}
}

func Test_SendMessage_DeniedChannel(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	filter := safety.NewFilter(nil, []string{"general"})
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_send_message")

	req := testutil.NewCallToolRequest("discord_send_message", map[string]any{
		"channel": "general",
		"content": "should be blocked",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "not allowed") && !strings.Contains(lower, "denied") {
		t.Errorf("expected error about channel not allowed, got: %s", text)
	}
}

func Test_SendMessage_WithReplyTo(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_send_message")

	req := testutil.NewCallToolRequest("discord_send_message", map[string]any{
		"channel":  "123456789012345678",
		"content":  "replying to something",
		"reply_to": "original-msg-id",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "error") {
		t.Errorf("expected success with reply_to, got: %s", text)
	}
}

// ---------------------------------------------------------------------------
// discord_get_messages handler
// ---------------------------------------------------------------------------

func Test_GetMessages_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_messages")

	req := testutil.NewCallToolRequest("discord_get_messages", map[string]any{
		"channel": "123456789012345678",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	if !strings.Contains(text, "Hello from mock") {
		t.Errorf("expected mock message content, got: %s", text)
	}
}

func Test_GetMessages_DeniedChannel(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	filter := safety.NewFilter(nil, []string{"general"})
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_get_messages")

	req := testutil.NewCallToolRequest("discord_get_messages", map[string]any{
		"channel": "general",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "not allowed") && !strings.Contains(lower, "denied") {
		t.Errorf("expected denied error, got: %s", text)
	}
}

// ---------------------------------------------------------------------------
// discord_edit_message handler
// ---------------------------------------------------------------------------

func Test_EditMessage_Valid(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_edit_message")

	req := testutil.NewCallToolRequest("discord_edit_message", map[string]any{
		"channel":    "123456789012345678",
		"message_id": "msg-100",
		"content":    "updated text",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	// A successful edit should not start with "error:".
	if strings.HasPrefix(lower, "error:") {
		t.Errorf("expected success for edit, got: %s", text)
	}
}

// ---------------------------------------------------------------------------
// discord_delete_message handler
// ---------------------------------------------------------------------------

func Test_DeleteMessage_NoConfirmationToken(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker([]string{"discord_delete_message"})

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_delete_message")

	req := testutil.NewCallToolRequest("discord_delete_message", map[string]any{
		"channel":    "123456789012345678",
		"message_id": "msg-100",
	})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	text := testutil.ExtractText(t, result)
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "confirmation") {
		t.Errorf("expected confirmation prompt, got: %s", text)
	}
	if !strings.Contains(text, "confirmation_token=") {
		t.Errorf("expected confirmation_token in response, got: %s", text)
	}
}

func Test_DeleteMessage_WithValidConfirmationToken(t *testing.T) {
	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker([]string{"discord_delete_message"})

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(t, regs, "discord_delete_message")

	// First call: get the confirmation token.
	req1 := testutil.NewCallToolRequest("discord_delete_message", map[string]any{
		"channel":    "123456789012345678",
		"message_id": "msg-100",
	})

	result1, err := handler(context.Background(), req1)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	text1 := testutil.ExtractText(t, result1)
	token := extractConfirmationToken(t, text1)

	// Second call: provide the confirmation token.
	req2 := testutil.NewCallToolRequest("discord_delete_message", map[string]any{
		"channel":            "123456789012345678",
		"message_id":         "msg-100",
		"confirmation_token": token,
	})

	result2, err := handler(context.Background(), req2)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	text2 := testutil.ExtractText(t, result2)
	lower := strings.ToLower(text2)
	if strings.Contains(lower, "confirmation required") {
		t.Errorf("expected deletion success after confirmation, got another prompt: %s", text2)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func Benchmark_PollMessages_EmptyQueue(b *testing.B) {
	md := testutil.NewMockDiscordSession(&testing.T{})
	defer md.Close()

	q := queue.New()
	r := resolve.New(md.Session, "guild-1")
	filter := safety.NewFilter(nil, nil)
	confirm := safety.NewConfirmationTracker(nil)

	regs := message.MessageTools(md.Session, q, r, filter, confirm, nil, nil)
	handler := testutil.FindHandler(&testing.T{}, regs, "discord_poll_messages")

	req := testutil.NewCallToolRequest("discord_poll_messages", map[string]any{
		"timeout_seconds": float64(1),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = handler(ctx, req)
		cancel()
	}
}

func extractConfirmationToken(t *testing.T, text string) string {
	t.Helper()
	const prefix = `confirmation_token="`
	idx := strings.Index(text, prefix)
	if idx < 0 {
		const altPrefix = `confirmation_token=`
		idx = strings.Index(text, altPrefix)
		if idx < 0 {
			t.Fatalf("could not find confirmation_token in text: %s", text)
		}
		after := text[idx+len(altPrefix):]
		endIdx := strings.IndexAny(after, `". `+"\n")
		if endIdx < 0 {
			return after
		}
		return after[:endIdx]
	}
	after := text[idx+len(prefix):]
	endIdx := strings.Index(after, `"`)
	if endIdx < 0 {
		t.Fatalf("could not find closing quote for token: %s", text)
	}
	return after[:endIdx]
}

// Ensure imports are used. These variable declarations are a compile-time check
// that the expected types are available.
var (
	_ mcp.CallToolRequest
	_ server.ToolHandlerFunc
)
