package resolve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// testChannels returns a set of mock Discord channels for testing.
// Includes both text and voice channels.
func testChannels() []*discordgo.Channel {
	return []*discordgo.Channel{
		{
			ID:   "111",
			Name: "general",
			Type: discordgo.ChannelTypeGuildText,
		},
		{
			ID:   "222",
			Name: "random",
			Type: discordgo.ChannelTypeGuildText,
		},
		{
			ID:   "333",
			Name: "voice-chat",
			Type: discordgo.ChannelTypeGuildVoice,
		},
		{
			ID:   "444",
			Name: "announcements",
			Type: discordgo.ChannelTypeGuildText,
		},
	}
}

// newTestResolver sets up a mock Discord API server and returns a Resolver
// that uses it, plus a cleanup function.
func newTestResolver(t *testing.T, guildID string, channels []*discordgo.Channel) *Resolver {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v9/guilds/"+guildID+"/channels", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(channels); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Create a discordgo session pointing to our test server.
	session, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("failed to create discordgo session: %v", err)
	}
	// Save original endpoint values before mutating so they can be restored.
	origAPI := discordgo.EndpointAPI
	origGuilds := discordgo.EndpointGuilds

	// Override the API endpoint to point to our test server.
	// discordgo uses session.EndPoint as the base URL.
	// The typical pattern is to set the package-level EndPoint variable.
	discordgo.EndpointAPI = server.URL + "/api/v9/"
	discordgo.EndpointGuilds = discordgo.EndpointAPI + "guilds/"

	t.Cleanup(func() {
		discordgo.EndpointAPI = origAPI
		discordgo.EndpointGuilds = origGuilds
	})

	return New(session, guildID)
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func Test_Refresh_PopulatesCache(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// After refresh, ChannelName should work for text channels.
	name := r.ChannelName("111")
	if name != "general" {
		t.Errorf("ChannelName('111') = %q, want %q", name, "general")
	}

	name = r.ChannelName("222")
	if name != "random" {
		t.Errorf("ChannelName('222') = %q, want %q", name, "random")
	}
}

func Test_Refresh_OnlyTextChannelsCached(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Voice channel "333" should NOT be cached.
	name := r.ChannelName("333")
	// Cache miss should return the ID itself.
	if name != "333" {
		t.Errorf("ChannelName('333') for voice channel = %q, want %q (cache miss returns ID)", name, "333")
	}

	// Text channel "444" should be cached.
	name = r.ChannelName("444")
	if name != "announcements" {
		t.Errorf("ChannelName('444') = %q, want %q", name, "announcements")
	}
}

// ---------------------------------------------------------------------------
// ChannelName
// ---------------------------------------------------------------------------

func Test_ChannelName_CachedEntry(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	got := r.ChannelName("111")
	if got != "general" {
		t.Errorf("ChannelName('111') = %q, want %q", got, "general")
	}
}

func Test_ChannelName_CacheMiss_ReturnsID(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// ID "999" is not in the channel list.
	got := r.ChannelName("999")
	if got != "999" {
		t.Errorf("ChannelName('999') cache miss = %q, want %q (should return ID)", got, "999")
	}
}

func Test_ChannelName_BeforeRefresh_ReturnsID(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	// No Refresh() called — cache is empty.
	got := r.ChannelName("111")
	if got != "111" {
		t.Errorf("ChannelName('111') before refresh = %q, want %q (should return ID)", got, "111")
	}
}

// ---------------------------------------------------------------------------
// ChannelID
// ---------------------------------------------------------------------------

func Test_ChannelID_CachedEntry(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	id, err := r.ChannelID("general")
	if err != nil {
		t.Fatalf("ChannelID('general') error = %v", err)
	}
	if id != "111" {
		t.Errorf("ChannelID('general') = %q, want %q", id, "111")
	}
}

func Test_ChannelID_StripsHashPrefix(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	id, err := r.ChannelID("#general")
	if err != nil {
		t.Fatalf("ChannelID('#general') error = %v", err)
	}
	if id != "111" {
		t.Errorf("ChannelID('#general') = %q, want %q", id, "111")
	}
}

func Test_ChannelID_CacheMiss_ReturnsError(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	_, err := r.ChannelID("nonexistent")
	if err == nil {
		t.Fatal("ChannelID('nonexistent') expected error for cache miss, got nil")
	}
}

func Test_ChannelID_BeforeRefresh_ReturnsError(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	// No Refresh() called.
	_, err := r.ChannelID("general")
	if err == nil {
		t.Fatal("ChannelID('general') before refresh expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func Test_ChannelID_EmptyName_ReturnsError(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	_, err := r.ChannelID("")
	if err == nil {
		t.Fatal("ChannelID('') expected error for empty name, got nil")
	}
}

func Test_ChannelID_HashOnly_ReturnsError(t *testing.T) {
	channels := testChannels()
	r := newTestResolver(t, "guild-1", channels)

	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// "#" should be stripped to "", which should not match anything.
	_, err := r.ChannelID("#")
	if err == nil {
		t.Fatal("ChannelID('#') expected error, got nil")
	}
}

func Test_Refresh_MultipleCallsOverwriteCache(t *testing.T) {
	// Start with one set of channels.
	channels1 := []*discordgo.Channel{
		{ID: "111", Name: "general", Type: discordgo.ChannelTypeGuildText},
	}

	mux := http.NewServeMux()
	callCount := 0
	mux.HandleFunc("/api/v9/guilds/guild-1/channels", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		var channels []*discordgo.Channel
		if callCount == 1 {
			channels = channels1
		} else {
			// Second call returns a different set.
			channels = []*discordgo.Channel{
				{ID: "555", Name: "new-channel", Type: discordgo.ChannelTypeGuildText},
			}
		}
		if err := json.NewEncoder(w).Encode(channels); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	session, err := discordgo.New("Bot fake-token")
	if err != nil {
		t.Fatalf("failed to create discordgo session: %v", err)
	}

	// Save original endpoint values before mutating so they can be restored.
	origAPI := discordgo.EndpointAPI
	origGuilds := discordgo.EndpointGuilds

	discordgo.EndpointAPI = server.URL + "/api/v9/"
	discordgo.EndpointGuilds = discordgo.EndpointAPI + "guilds/"

	t.Cleanup(func() {
		discordgo.EndpointAPI = origAPI
		discordgo.EndpointGuilds = origGuilds
	})

	resolver := New(session, "guild-1")

	// First refresh.
	if err := resolver.Refresh(); err != nil {
		t.Fatalf("first Refresh() error = %v", err)
	}
	if name := resolver.ChannelName("111"); name != "general" {
		t.Errorf("after first refresh: ChannelName('111') = %q, want %q", name, "general")
	}

	// Second refresh — cache should be overwritten.
	if err := resolver.Refresh(); err != nil {
		t.Fatalf("second Refresh() error = %v", err)
	}
	if name := resolver.ChannelName("555"); name != "new-channel" {
		t.Errorf("after second refresh: ChannelName('555') = %q, want %q", name, "new-channel")
	}
	// Old entry should now be a cache miss.
	if name := resolver.ChannelName("111"); name != "111" {
		t.Errorf("after second refresh: ChannelName('111') = %q, want %q (cache miss)", name, "111")
	}
}
