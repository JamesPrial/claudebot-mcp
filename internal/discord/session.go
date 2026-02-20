// Package discord wraps a discordgo.Session with queue, resolver, and filter
// integration to provide a ready-to-use Discord ingestion layer for claudebot-mcp.
package discord

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/queue"
	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/safety"
)

// Session wraps a discordgo.Session and routes incoming guild messages through
// the safety filter before pushing them onto the message queue.
type Session struct {
	dg       *discordgo.Session
	guildID  string
	queue    *queue.Queue
	resolver *resolve.Resolver
	// filter applies channel filtering at the ingestion level, preventing
	// messages from denied channels from entering the queue. When nil, all
	// messages from the configured guild are enqueued. Currently,
	// NewFromSession passes nil for this field; channel filtering is
	// enforced at the tool handler level instead. The field is exercised
	// by tests via the internal newFromSessionFull constructor.
	filter *safety.Filter
	logger *slog.Logger
}

// NewFromSession wraps an existing *discordgo.Session, registering message and
// ready event handlers and configuring the required gateway intents. The guild
// ID is read from the resolver. A nil filter allows all channels; a nil logger
// defaults to slog.Default().
//
// Intents enabled:
//   - IntentGuilds
//   - IntentGuildMessages
//   - IntentMessageContent
//   - IntentGuildMessageReactions
func NewFromSession(
	dg *discordgo.Session,
	q *queue.Queue,
	r *resolve.Resolver,
	logger *slog.Logger,
) *Session {
	return newFromSessionFull(dg, q, r, nil, logger)
}

// newFromSessionFull is the internal constructor used by NewFromSession and by
// callers that need to supply a custom filter and/or logger. A nil logger
// defaults to slog.Default().
func newFromSessionFull(
	dg *discordgo.Session,
	q *queue.Queue,
	r *resolve.Resolver,
	filter *safety.Filter,
	logger *slog.Logger,
) *Session {
	if logger == nil {
		logger = slog.Default()
	}

	s := &Session{
		dg:       dg,
		guildID:  r.GuildID(),
		queue:    q,
		resolver: r,
		filter:   filter,
		logger:   logger,
	}

	dg.Identify.Intents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentMessageContent |
		discordgo.IntentGuildMessageReactions

	dg.AddHandler(s.onReady)
	dg.AddHandler(s.onMessageCreate)

	return s
}

// Open establishes the WebSocket connection to the Discord gateway.
// It must be called after NewFromSession to begin receiving events.
func (s *Session) Open() error {
	return s.dg.Open()
}

// Close gracefully closes the WebSocket connection to the Discord gateway.
// It is safe to call Close multiple times.
func (s *Session) Close() error {
	return s.dg.Close()
}

// DiscordSession returns the underlying *discordgo.Session for callers that
// need direct access to the Discord API.
func (s *Session) DiscordSession() *discordgo.Session {
	return s.dg
}

// onReady is called when the Discord gateway confirms the bot is connected.
// It logs the bot's username and triggers an initial channel cache refresh.
func (s *Session) onReady(dg *discordgo.Session, event *discordgo.Ready) {
	s.logger.Info("discord connected",
		"username", event.User.Username,
		"discriminator", event.User.Discriminator,
	)
	if err := s.resolver.Refresh(); err != nil {
		s.logger.Warn("channel cache refresh failed", "error", err)
	}
}

// onMessageCreate handles incoming Discord message events. It filters out bot
// messages, messages from other guilds, and messages in denied channels before
// resolving the channel name and enqueueing the message.
func (s *Session) onMessageCreate(dg *discordgo.Session, event *discordgo.MessageCreate) {
	if event.Author == nil {
		return
	}

	// Ignore messages from bots (including ourselves).
	if event.Author.Bot {
		return
	}

	// Ignore messages that belong to a different guild.
	if event.GuildID != s.guildID {
		return
	}

	// Resolve the channel name for filter and display purposes.
	channelName := s.resolver.ChannelName(event.ChannelID)

	// Apply channel filter using the resolved name.
	if s.filter != nil && !s.filter.IsAllowed(channelName) {
		s.logger.Debug("message filtered by channel deny", "channel", channelName, "author", event.Author.Username)
		return
	}

	// Build the message reference string if this is a reply.
	var msgRef string
	if event.MessageReference != nil {
		msgRef = event.MessageReference.MessageID
	}

	msg := queue.QueuedMessage{
		ID:               event.ID,
		ChannelID:        event.ChannelID,
		ChannelName:      channelName,
		AuthorID:         event.Author.ID,
		AuthorUsername:   event.Author.Username,
		Content:          event.Content,
		Timestamp:        event.Timestamp,
		MessageReference: msgRef,
	}

	s.queue.Enqueue(msg)
	s.logger.Debug("message enqueued", "id", event.ID, "channel", channelName, "author", event.Author.Username)
}
