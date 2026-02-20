package testutil

import (
	"fmt"
	"strings"

	"github.com/jamesprial/claudebot-mcp/internal/resolve"
)

// Compile-time assertion.
var _ resolve.ChannelResolver = (*MockChannelResolver)(nil)

// MockChannelResolver implements resolve.ChannelResolver using in-memory maps.
// It is pre-populated with standard test channels by NewMockChannelResolver.
type MockChannelResolver struct {
	IDToName map[string]string // channel ID -> name
	NameToID map[string]string // channel name -> ID
}

// NewMockChannelResolver returns a MockChannelResolver pre-loaded with the
// standard test channels: ch-001/general and ch-002/random.
func NewMockChannelResolver() *MockChannelResolver {
	return &MockChannelResolver{
		IDToName: map[string]string{"ch-001": "general", "ch-002": "random"},
		NameToID: map[string]string{"general": "ch-001", "random": "ch-002"},
	}
}

// ChannelName returns the name for the given channel ID. If the ID is not
// found, the ID itself is returned (matching *resolve.Resolver behavior).
func (m *MockChannelResolver) ChannelName(id string) string {
	if name, ok := m.IDToName[id]; ok {
		return name
	}
	return id
}

// ChannelID returns the ID for the given channel name. A leading "#" is
// stripped. If the name is not found, an error is returned (matching
// *resolve.Resolver behavior).
func (m *MockChannelResolver) ChannelID(name string) (string, error) {
	name = strings.TrimPrefix(name, "#")
	if id, ok := m.NameToID[name]; ok {
		return id, nil
	}
	return "", fmt.Errorf("resolve: channel %q not found", name)
}
