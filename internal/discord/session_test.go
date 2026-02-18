package discord

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestSession constructs a *Session using newFromSessionFull so that a
// custom safety.Filter and a silent logger can be injected. The returned queue
// can be inspected after handler invocations to verify what was enqueued.
func newTestSession(t *testing.T, guildID string, filter *safety.Filter) (*Session, *queue.Queue) {
	t.Helper()

	dg, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}

	q := queue.New()
	r := resolve.New(dg, guildID)

	// Use a silent logger so tests don't spam stderr.
	silent := log.New(log.Writer(), "", 0)
	s := newFromSessionFull(dg, q, r, filter, silent)

	return s, q
}

// drainQueue polls all messages from q with a very short timeout (non-blocking).
func drainQueue(q *queue.Queue, limit int) []queue.QueuedMessage {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	return q.Poll(ctx, 10*time.Millisecond, limit, "")
}

// ---------------------------------------------------------------------------
// NewFromSession
// ---------------------------------------------------------------------------

func Test_NewFromSession_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	dg, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}

	q := queue.New()
	r := resolve.New(dg, "guild-1")

	s := NewFromSession(dg, q, r)
	if s == nil {
		t.Fatal("NewFromSession() returned nil")
	}
}

func Test_NewFromSession_StoresComponents(t *testing.T) {
	t.Parallel()

	dg, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}

	q := queue.New()
	r := resolve.New(dg, "guild-1")

	s := NewFromSession(dg, q, r)
	if s == nil {
		t.Fatal("NewFromSession() returned nil")
	}

	// Verify the underlying discordgo session is accessible.
	got := s.DiscordSession()
	if got != dg {
		t.Error("DiscordSession() did not return the same *discordgo.Session that was passed in")
	}
}

// ---------------------------------------------------------------------------
// DiscordSession
// ---------------------------------------------------------------------------

func Test_DiscordSession_ReturnsUnderlyingSession(t *testing.T) {
	t.Parallel()

	dg, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}

	q := queue.New()
	r := resolve.New(dg, "guild-1")

	s := NewFromSession(dg, q, r)
	if s == nil {
		t.Fatal("NewFromSession() returned nil")
	}

	underlying := s.DiscordSession()
	if underlying == nil {
		t.Fatal("DiscordSession() returned nil")
	}
	if underlying != dg {
		t.Error("DiscordSession() returned a different session than the one provided to NewFromSession")
	}
}

func Test_DiscordSession_TokenPreserved(t *testing.T) {
	t.Parallel()

	dg, err := discordgo.New("Bot test-token-abc")
	if err != nil {
		t.Fatalf("discordgo.New() error = %v", err)
	}

	q := queue.New()
	r := resolve.New(dg, "guild-1")

	s := NewFromSession(dg, q, r)
	underlying := s.DiscordSession()

	// The token should be preserved through the wrapper.
	if underlying.Token != "Bot test-token-abc" {
		t.Errorf("DiscordSession().Token = %q, want %q", underlying.Token, "Bot test-token-abc")
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate
// ---------------------------------------------------------------------------

func Test_onMessageCreate_BotMessage_NotEnqueued(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-1",
			ChannelID: "chan-1",
			GuildID:   "guild-1",
			Content:   "hello from bot",
			Author: &discordgo.User{
				ID:       "bot-user-1",
				Username: "TestBot",
				Bot:      true,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected queue to be empty for bot message, got Len() = %d", q.Len())
	}
}

func Test_onMessageCreate_WrongGuild_NotEnqueued(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-2",
			ChannelID: "chan-1",
			GuildID:   "guild-OTHER",
			Content:   "hello from wrong guild",
			Author: &discordgo.User{
				ID:       "user-1",
				Username: "Alice",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected queue to be empty for wrong guild, got Len() = %d", q.Len())
	}
}

func Test_onMessageCreate_DeniedChannel_NotEnqueued(t *testing.T) {
	t.Parallel()

	// Create a filter that denies "secret-channel".
	filter := safety.NewFilter(nil, []string{"secret-channel"})
	s, q := newTestSession(t, "guild-1", filter)

	// The resolver does not have the channel cached, so ChannelName returns the
	// raw ID. We use the channel name directly as the channel ID so that the
	// resolver falls through and returns it as-is, matching the denylist.
	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-3",
			ChannelID: "secret-channel",
			GuildID:   "guild-1",
			Content:   "should be filtered",
			Author: &discordgo.User{
				ID:       "user-1",
				Username: "Alice",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected queue to be empty for denied channel, got Len() = %d", q.Len())
	}
}

func Test_onMessageCreate_NormalMessage_Enqueued(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	now := time.Now().Truncate(time.Second)
	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-4",
			ChannelID: "chan-42",
			GuildID:   "guild-1",
			Content:   "hello world",
			Timestamp: now,
			Author: &discordgo.User{
				ID:       "user-2",
				Username: "Bob",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 1 {
		t.Fatalf("expected queue Len() = 1, got %d", q.Len())
	}

	msgs := drainQueue(q, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message from Poll, got %d", len(msgs))
	}

	msg := msgs[0]
	if msg.ID != "msg-4" {
		t.Errorf("QueuedMessage.ID = %q, want %q", msg.ID, "msg-4")
	}
	if msg.ChannelID != "chan-42" {
		t.Errorf("QueuedMessage.ChannelID = %q, want %q", msg.ChannelID, "chan-42")
	}
	// The resolver cache is empty, so ChannelName falls back to the channel ID.
	if msg.ChannelName != "chan-42" {
		t.Errorf("QueuedMessage.ChannelName = %q, want %q", msg.ChannelName, "chan-42")
	}
	if msg.AuthorID != "user-2" {
		t.Errorf("QueuedMessage.AuthorID = %q, want %q", msg.AuthorID, "user-2")
	}
	if msg.AuthorUsername != "Bob" {
		t.Errorf("QueuedMessage.AuthorUsername = %q, want %q", msg.AuthorUsername, "Bob")
	}
	if msg.Content != "hello world" {
		t.Errorf("QueuedMessage.Content = %q, want %q", msg.Content, "hello world")
	}
	if !msg.Timestamp.Equal(now) {
		t.Errorf("QueuedMessage.Timestamp = %v, want %v", msg.Timestamp, now)
	}
	if msg.MessageReference != "" {
		t.Errorf("QueuedMessage.MessageReference = %q, want empty string", msg.MessageReference)
	}
}

func Test_onMessageCreate_NilAuthor_NoPanic(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-5",
			ChannelID: "chan-1",
			GuildID:   "guild-1",
			Content:   "nil author message",
			Author:    nil,
		},
	}

	// This must not panic.
	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected queue to be empty for nil-author message, got Len() = %d", q.Len())
	}
}

func Test_onMessageCreate_WithMessageReference_EnqueuedWithRef(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-6",
			ChannelID: "chan-1",
			GuildID:   "guild-1",
			Content:   "replying to something",
			Author: &discordgo.User{
				ID:       "user-3",
				Username: "Carol",
				Bot:      false,
			},
			MessageReference: &discordgo.MessageReference{
				MessageID: "original-msg-99",
				ChannelID: "chan-1",
				GuildID:   "guild-1",
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 1 {
		t.Fatalf("expected queue Len() = 1, got %d", q.Len())
	}

	msgs := drainQueue(q, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message from Poll, got %d", len(msgs))
	}

	if msgs[0].MessageReference != "original-msg-99" {
		t.Errorf("QueuedMessage.MessageReference = %q, want %q", msgs[0].MessageReference, "original-msg-99")
	}
}

func Test_onMessageCreate_EmptyContent_StillEnqueued(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-7",
			ChannelID: "chan-1",
			GuildID:   "guild-1",
			Content:   "",
			Author: &discordgo.User{
				ID:       "user-4",
				Username: "Dave",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 1 {
		t.Fatalf("expected queue Len() = 1 for empty-content message, got %d", q.Len())
	}

	msgs := drainQueue(q, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message from Poll, got %d", len(msgs))
	}

	if msgs[0].Content != "" {
		t.Errorf("QueuedMessage.Content = %q, want empty string", msgs[0].Content)
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate - table-driven filtering tests
// ---------------------------------------------------------------------------

func Test_onMessageCreate_FilteringCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		guildID    string
		filter     *safety.Filter
		event      *discordgo.MessageCreate
		wantQueued bool
	}{
		{
			name:    "bot author is filtered out",
			guildID: "guild-1",
			filter:  nil,
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-1", ChannelID: "c-1", GuildID: "guild-1",
					Content: "bot says hi",
					Author:  &discordgo.User{ID: "u-1", Username: "Bot", Bot: true},
				},
			},
			wantQueued: false,
		},
		{
			name:    "wrong guild is filtered out",
			guildID: "guild-1",
			filter:  nil,
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-2", ChannelID: "c-1", GuildID: "guild-99",
					Content: "wrong guild",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: false,
		},
		{
			name:    "nil author is filtered out",
			guildID: "guild-1",
			filter:  nil,
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-3", ChannelID: "c-1", GuildID: "guild-1",
					Content: "ghost message",
					Author:  nil,
				},
			},
			wantQueued: false,
		},
		{
			name:    "denied channel is filtered out",
			guildID: "guild-1",
			filter:  safety.NewFilter(nil, []string{"banned-chan"}),
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-4", ChannelID: "banned-chan", GuildID: "guild-1",
					Content: "nope",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: false,
		},
		{
			name:    "allowlist-only filter blocks unlisted channels",
			guildID: "guild-1",
			filter:  safety.NewFilter([]string{"allowed-chan"}, nil),
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-5", ChannelID: "other-chan", GuildID: "guild-1",
					Content: "not on the list",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: false,
		},
		{
			name:    "allowlist-only filter passes listed channels",
			guildID: "guild-1",
			filter:  safety.NewFilter([]string{"allowed-chan"}, nil),
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-6", ChannelID: "allowed-chan", GuildID: "guild-1",
					Content: "on the list",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: true,
		},
		{
			name:    "nil filter allows all channels",
			guildID: "guild-1",
			filter:  nil,
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-7", ChannelID: "any-chan", GuildID: "guild-1",
					Content: "goes through",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: true,
		},
		{
			name:    "empty guild ID never matches configured guild",
			guildID: "guild-1",
			filter:  nil,
			event: &discordgo.MessageCreate{
				Message: &discordgo.Message{
					ID: "t-8", ChannelID: "c-1", GuildID: "",
					Content: "DM or no guild",
					Author:  &discordgo.User{ID: "u-1", Username: "Alice", Bot: false},
				},
			},
			wantQueued: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s, q := newTestSession(t, tt.guildID, tt.filter)
			s.onMessageCreate(s.dg, tt.event)

			gotLen := q.Len()
			if tt.wantQueued && gotLen != 1 {
				t.Errorf("expected message to be enqueued (Len=1), got Len=%d", gotLen)
			}
			if !tt.wantQueued && gotLen != 0 {
				t.Errorf("expected message to be filtered out (Len=0), got Len=%d", gotLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate - field mapping verification
// ---------------------------------------------------------------------------

func Test_onMessageCreate_FieldMapping(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	ts := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "field-msg-1",
			ChannelID: "field-chan-1",
			GuildID:   "guild-1",
			Content:   "full field test",
			Timestamp: ts,
			Author: &discordgo.User{
				ID:       "field-user-1",
				Username: "FieldTester",
				Bot:      false,
			},
			MessageReference: &discordgo.MessageReference{
				MessageID: "ref-msg-42",
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 1 {
		t.Fatalf("expected queue Len() = 1, got %d", q.Len())
	}

	msgs := drainQueue(q, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	msg := msgs[0]

	// Verify every field is correctly mapped.
	checks := []struct {
		field string
		got   string
		want  string
	}{
		{"ID", msg.ID, "field-msg-1"},
		{"ChannelID", msg.ChannelID, "field-chan-1"},
		{"ChannelName", msg.ChannelName, "field-chan-1"}, // resolver cache empty, falls back to ID
		{"AuthorID", msg.AuthorID, "field-user-1"},
		{"AuthorUsername", msg.AuthorUsername, "FieldTester"},
		{"Content", msg.Content, "full field test"},
		{"MessageReference", msg.MessageReference, "ref-msg-42"},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("QueuedMessage.%s = %q, want %q", c.field, c.got, c.want)
		}
	}

	if !msg.Timestamp.Equal(ts) {
		t.Errorf("QueuedMessage.Timestamp = %v, want %v", msg.Timestamp, ts)
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate - multiple messages accumulate in queue
// ---------------------------------------------------------------------------

func Test_onMessageCreate_MultipleMessages_AllEnqueued(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	for i := 0; i < 5; i++ {
		event := &discordgo.MessageCreate{
			Message: &discordgo.Message{
				ID:        "multi-" + string(rune('A'+i)),
				ChannelID: "chan-1",
				GuildID:   "guild-1",
				Content:   "message",
				Author: &discordgo.User{
					ID:       "user-1",
					Username: "Alice",
					Bot:      false,
				},
			},
		}
		s.onMessageCreate(s.dg, event)
	}

	if q.Len() != 5 {
		t.Errorf("expected queue Len() = 5 after 5 messages, got %d", q.Len())
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate - nil MessageReference leaves field empty
// ---------------------------------------------------------------------------

func Test_onMessageCreate_NilMessageReference_EmptyField(t *testing.T) {
	t.Parallel()

	s, q := newTestSession(t, "guild-1", nil)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:               "msg-no-ref",
			ChannelID:        "chan-1",
			GuildID:          "guild-1",
			Content:          "not a reply",
			MessageReference: nil,
			Author: &discordgo.User{
				ID:       "user-1",
				Username: "Alice",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	msgs := drainQueue(q, 1)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].MessageReference != "" {
		t.Errorf("QueuedMessage.MessageReference = %q, want empty for nil MessageReference", msgs[0].MessageReference)
	}
}

// ---------------------------------------------------------------------------
// onReady
// ---------------------------------------------------------------------------

func Test_onReady_NoPanic(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t, "guild-1", nil)

	event := &discordgo.Ready{
		User: &discordgo.User{
			Username:      "TestBot",
			Discriminator: "1234",
		},
	}

	// onReady calls s.resolver.Refresh() which will fail because the discordgo
	// session is not actually connected. The handler should log the error but
	// not panic.
	s.onReady(s.dg, event)
}

func Test_onReady_WithValidUser_NoPanic(t *testing.T) {
	t.Parallel()

	s, _ := newTestSession(t, "guild-1", nil)

	event := &discordgo.Ready{
		User: &discordgo.User{
			Username:      "ClaudeBot",
			Discriminator: "0001",
			ID:            "bot-id-123",
		},
	}

	// Must not panic even though the underlying session cannot reach Discord.
	s.onReady(s.dg, event)
}

// ---------------------------------------------------------------------------
// onMessageCreate - denylist with glob pattern
// ---------------------------------------------------------------------------

func Test_onMessageCreate_DenylistGlobPattern_NotEnqueued(t *testing.T) {
	t.Parallel()

	// Deny all channels matching "admin-*".
	filter := safety.NewFilter(nil, []string{"admin-*"})
	s, q := newTestSession(t, "guild-1", filter)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-glob",
			ChannelID: "admin-logs",
			GuildID:   "guild-1",
			Content:   "should be denied by glob",
			Author: &discordgo.User{
				ID:       "user-1",
				Username: "Alice",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected queue to be empty for glob-denied channel, got Len() = %d", q.Len())
	}
}

// ---------------------------------------------------------------------------
// onMessageCreate - denylist takes priority over allowlist
// ---------------------------------------------------------------------------

func Test_onMessageCreate_DenylistOverridesAllowlist(t *testing.T) {
	t.Parallel()

	// Allow "general" but also deny it. Denylist wins.
	filter := safety.NewFilter([]string{"general"}, []string{"general"})
	s, q := newTestSession(t, "guild-1", filter)

	event := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-priority",
			ChannelID: "general",
			GuildID:   "guild-1",
			Content:   "denied despite allowlist",
			Author: &discordgo.User{
				ID:       "user-1",
				Username: "Alice",
				Bot:      false,
			},
		},
	}

	s.onMessageCreate(s.dg, event)

	if q.Len() != 0 {
		t.Errorf("expected denylist to override allowlist, got Len() = %d", q.Len())
	}
}
