// Package testutil provides shared test infrastructure for claudebot-mcp tool tests.
//
// The primary helper is NewMockDiscordSession, which starts an httptest.Server
// that simulates key Discord REST API endpoints and returns a *discordgo.Session
// pointing to it.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockDiscord bundles the test server and discordgo session together so callers
// can register additional handlers or inspect request state.
type MockDiscord struct {
	Server  *httptest.Server
	Session *discordgo.Session
	Mux     *http.ServeMux
}

// Close shuts down the test server. It should be called via t.Cleanup.
func (m *MockDiscord) Close() {
	m.Server.Close()
}

// NewMockDiscordSession starts an httptest.Server with handlers that simulate
// Discord's REST API and returns a MockDiscord that wraps both the server and
// a discordgo.Session pointed at it.
//
// The returned MockDiscord should be cleaned up via t.Cleanup:
//
//	md := testutil.NewMockDiscordSession(t)
//	t.Cleanup(md.Close)
func NewMockDiscordSession(t *testing.T) *MockDiscord {
	t.Helper()

	mux := http.NewServeMux()

	// --- Channel messages (send) ---
	// POST /api/v9/channels/{cID}/messages
	mux.HandleFunc("/api/v9/channels/", func(w http.ResponseWriter, r *http.Request) {
		// Parse the path to figure out which sub-resource is being accessed.
		path := strings.TrimPrefix(r.URL.Path, "/api/v9/channels/")
		parts := strings.Split(path, "/")

		if len(parts) < 1 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}

		channelID := parts[0]

		switch {
		// POST /channels/{id}/messages — send message
		case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "messages":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			resp := &discordgo.Message{
				ID:        "mock-msg-001",
				ChannelID: channelID,
				Content:   stringFromAny(body["content"]),
			}
			writeJSON(w, resp)

		// GET /channels/{id}/messages — get messages
		case r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "messages":
			msgs := []*discordgo.Message{
				{
					ID:        "msg-100",
					ChannelID: channelID,
					Content:   "Hello from mock",
					Author: &discordgo.User{
						ID:       "user-1",
						Username: "tester",
					},
				},
			}
			writeJSON(w, msgs)

		// PATCH /channels/{id}/messages/{mID} — edit message
		case r.Method == http.MethodPatch && len(parts) == 3 && parts[1] == "messages":
			msgID := parts[2]
			resp := &discordgo.Message{
				ID:        msgID,
				ChannelID: channelID,
				Content:   "edited content",
			}
			writeJSON(w, resp)

		// DELETE /channels/{id}/messages/{mID} — delete message
		case r.Method == http.MethodDelete && len(parts) == 3 && parts[1] == "messages":
			w.WriteHeader(http.StatusNoContent)

		// PUT /channels/{id}/messages/{mID}/reactions/{emoji}/@me — add reaction
		case r.Method == http.MethodPut && len(parts) >= 5 && parts[1] == "messages" && parts[3] == "reactions":
			w.WriteHeader(http.StatusNoContent)

		// DELETE /channels/{id}/messages/{mID}/reactions/{emoji}/@me — remove reaction
		case r.Method == http.MethodDelete && len(parts) >= 5 && parts[1] == "messages" && parts[3] == "reactions":
			w.WriteHeader(http.StatusNoContent)

		// POST /channels/{id}/typing — typing indicator
		case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "typing":
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// --- Guild channels ---
	// GET /api/v9/guilds/{gID}/channels
	mux.HandleFunc("/api/v9/guilds/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v9/guilds/")
		parts := strings.Split(path, "/")

		if len(parts) < 1 {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}

		guildID := parts[0]

		switch {
		// GET /guilds/{id}/channels
		case r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "channels":
			channels := []*discordgo.Channel{
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
			}
			writeJSON(w, channels)

		// GET /guilds/{id} — get guild info
		case r.Method == http.MethodGet && len(parts) == 1:
			guild := &discordgo.Guild{
				ID:          guildID,
				Name:        "Test Guild",
				MemberCount: 42,
			}
			writeJSON(w, guild)

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	})

	// --- Users ---
	// GET /api/v9/users/{uID}
	mux.HandleFunc("/api/v9/users/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v9/users/")
		userID := strings.TrimSuffix(path, "/")

		if r.Method == http.MethodGet {
			user := &discordgo.User{
				ID:       userID,
				Username: "mockuser",
			}
			writeJSON(w, user)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	ts := httptest.NewServer(mux)

	// Override discordgo's endpoint variables so the session talks to our mock.
	discordgo.EndpointDiscord = ts.URL + "/"
	discordgo.EndpointAPI = discordgo.EndpointDiscord + "api/v" + discordgo.APIVersion + "/"
	discordgo.EndpointGuilds = discordgo.EndpointAPI + "guilds/"
	discordgo.EndpointChannels = discordgo.EndpointAPI + "channels/"
	discordgo.EndpointUsers = discordgo.EndpointAPI + "users/"

	dg, err := discordgo.New("Bot test-token")
	if err != nil {
		t.Fatalf("testutil: discordgo.New failed: %v", err)
	}

	return &MockDiscord{
		Server:  ts,
		Session: dg,
		Mux:     mux,
	}
}

// writeJSON marshals v as JSON and writes it to w with 200 OK.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// stringFromAny safely converts an interface{} to string.
func stringFromAny(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
