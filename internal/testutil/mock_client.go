package testutil

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jamesprial/claudebot-mcp/internal/discord"
)

// Compile-time assertion: *MockDiscordClient satisfies discord.DiscordClient.
var _ discord.DiscordClient = (*MockDiscordClient)(nil)

// MockDiscordClient implements discord.DiscordClient using configurable function
// fields. Each method delegates to its corresponding func field; when the field
// is nil the method returns a sensible default that matches the responses
// produced by NewMockDiscordSession's HTTP handlers.
type MockDiscordClient struct {
	ChannelMessageSendComplexFunc func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessagesFunc           func(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error)
	ChannelMessageEditFunc        func(channelID, messageID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageDeleteFunc      func(channelID, messageID string, options ...discordgo.RequestOption) error
	MessageReactionAddFunc        func(channelID, messageID, emojiID string, options ...discordgo.RequestOption) error
	MessageReactionRemoveFunc     func(channelID, messageID, emojiID, userID string, options ...discordgo.RequestOption) error
	GuildChannelsFunc             func(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildFunc                     func(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error)
	ChannelTypingFunc             func(channelID string, options ...discordgo.RequestOption) error
	UserFunc                      func(userID string, options ...discordgo.RequestOption) (*discordgo.User, error)
}

func (m *MockDiscordClient) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.ChannelMessageSendComplexFunc != nil {
		return m.ChannelMessageSendComplexFunc(channelID, data, options...)
	}
	return &discordgo.Message{
		ID:        "mock-msg-001",
		ChannelID: channelID,
	}, nil
}

func (m *MockDiscordClient) ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error) {
	if m.ChannelMessagesFunc != nil {
		return m.ChannelMessagesFunc(channelID, limit, beforeID, afterID, aroundID, options...)
	}
	return []*discordgo.Message{
		{
			ID:      "mock-msg-001",
			Content: "Hello from mock",
			Author: &discordgo.User{
				ID:       "user-001",
				Username: "mockuser",
			},
			Timestamp: time.Now(),
		},
	}, nil
}

func (m *MockDiscordClient) ChannelMessageEdit(channelID, messageID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.ChannelMessageEditFunc != nil {
		return m.ChannelMessageEditFunc(channelID, messageID, content, options...)
	}
	return &discordgo.Message{
		ID:        messageID,
		ChannelID: channelID,
		Content:   "edited content",
	}, nil
}

func (m *MockDiscordClient) ChannelMessageDelete(channelID, messageID string, options ...discordgo.RequestOption) error {
	if m.ChannelMessageDeleteFunc != nil {
		return m.ChannelMessageDeleteFunc(channelID, messageID, options...)
	}
	return nil
}

func (m *MockDiscordClient) MessageReactionAdd(channelID, messageID, emojiID string, options ...discordgo.RequestOption) error {
	if m.MessageReactionAddFunc != nil {
		return m.MessageReactionAddFunc(channelID, messageID, emojiID, options...)
	}
	return nil
}

func (m *MockDiscordClient) MessageReactionRemove(channelID, messageID, emojiID, userID string, options ...discordgo.RequestOption) error {
	if m.MessageReactionRemoveFunc != nil {
		return m.MessageReactionRemoveFunc(channelID, messageID, emojiID, userID, options...)
	}
	return nil
}

func (m *MockDiscordClient) GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error) {
	if m.GuildChannelsFunc != nil {
		return m.GuildChannelsFunc(guildID, options...)
	}
	return []*discordgo.Channel{
		{
			ID:   "ch-001",
			Name: "general",
			Type: discordgo.ChannelTypeGuildText,
		},
		{
			ID:   "ch-002",
			Name: "random",
			Type: discordgo.ChannelTypeGuildText,
		},
	}, nil
}

func (m *MockDiscordClient) Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error) {
	if m.GuildFunc != nil {
		return m.GuildFunc(guildID, options...)
	}
	return &discordgo.Guild{
		ID:          guildID,
		Name:        "Test Guild",
		MemberCount: 42,
	}, nil
}

func (m *MockDiscordClient) ChannelTyping(channelID string, options ...discordgo.RequestOption) error {
	if m.ChannelTypingFunc != nil {
		return m.ChannelTypingFunc(channelID, options...)
	}
	return nil
}

func (m *MockDiscordClient) User(userID string, options ...discordgo.RequestOption) (*discordgo.User, error) {
	if m.UserFunc != nil {
		return m.UserFunc(userID, options...)
	}
	return &discordgo.User{
		ID:       userID,
		Username: "mockuser",
	}, nil
}
