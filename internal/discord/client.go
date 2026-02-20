package discord

import "github.com/bwmarrin/discordgo"

// DiscordClient defines the subset of the Discord REST API used by MCP tool
// handlers. The concrete *discordgo.Session type satisfies this interface.
type DiscordClient interface {
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessages(channelID string, limit int, beforeID, afterID, aroundID string, options ...discordgo.RequestOption) ([]*discordgo.Message, error)
	ChannelMessageEdit(channelID, messageID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageDelete(channelID, messageID string, options ...discordgo.RequestOption) error
	MessageReactionAdd(channelID, messageID, emojiID string, options ...discordgo.RequestOption) error
	MessageReactionRemove(channelID, messageID, emojiID, userID string, options ...discordgo.RequestOption) error
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error)
	ChannelTyping(channelID string, options ...discordgo.RequestOption) error
	User(userID string, options ...discordgo.RequestOption) (*discordgo.User, error)
}

// Compile-time assertion: *discordgo.Session satisfies DiscordClient.
var _ DiscordClient = (*discordgo.Session)(nil)
