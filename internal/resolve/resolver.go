// Package resolve provides a channel name ↔ ID cache for a single Discord guild.
package resolve

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

// Resolver maintains an in-memory bidirectional cache of Discord channel IDs
// and names for a single guild. It is safe for concurrent use.
type Resolver struct {
	session *discordgo.Session
	guildID string
	mu      sync.RWMutex
	byID    map[string]string // channel ID -> name
	byName  map[string]string // channel name -> ID
}

// New constructs a Resolver for the given guild backed by the provided
// discordgo session. The cache is empty until Refresh is called.
func New(session *discordgo.Session, guildID string) *Resolver {
	return &Resolver{
		session: session,
		guildID: guildID,
		byID:    make(map[string]string),
		byName:  make(map[string]string),
	}
}

// GuildID returns the guild ID this Resolver was constructed with.
func (r *Resolver) GuildID() string {
	return r.guildID
}

// ChannelName returns the human-readable name for the channel with the given
// ID. If the ID is not present in the cache, the ID itself is returned so
// callers always receive a non-empty, printable value.
func (r *Resolver) ChannelName(id string) string {
	r.mu.RLock()
	name, ok := r.byID[id]
	r.mu.RUnlock()
	if !ok {
		return id
	}
	return name
}

// ChannelID returns the ID for the channel with the given name. A leading "#"
// is stripped before the lookup. If the name is not present in the cache, an
// error is returned.
func (r *Resolver) ChannelID(name string) (string, error) {
	name = strings.TrimPrefix(name, "#")

	r.mu.RLock()
	id, ok := r.byName[name]
	r.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("resolve: channel %q not found", name)
	}
	return id, nil
}

// Refresh fetches the current channel list for the guild from Discord and
// updates the cache. Only text channels (Type == discordgo.ChannelTypeGuildText,
// numeric value 0) are indexed. A write lock is held only during the map swap,
// so concurrent reads are not blocked during the network call.
func (r *Resolver) Refresh() error {
	channels, err := r.session.GuildChannels(r.guildID)
	if err != nil {
		return fmt.Errorf("resolve: failed to fetch guild channels: %w", err)
	}

	newByID := make(map[string]string, len(channels))
	newByName := make(map[string]string, len(channels))

	for _, ch := range channels {
		// Only cache text channels (Type == 0).
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		newByID[ch.ID] = ch.Name
		newByName[ch.Name] = ch.ID
	}

	r.mu.Lock()
	r.byID = newByID
	r.byName = newByName
	r.mu.Unlock()

	return nil
}

// ResolveChannelParam resolves a channel parameter that may be a name or ID.
// All-digit strings are treated as IDs, otherwise looked up via the Resolver.
// A leading "#" is stripped from names.
func ResolveChannelParam(r ChannelResolver, channel string) (string, error) {
	channel = strings.TrimPrefix(channel, "#")

	// All-digit strings are already IDs.
	allDigits := len(channel) > 0
	for _, c := range channel {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return channel, nil
	}

	return r.ChannelID(channel)
}
