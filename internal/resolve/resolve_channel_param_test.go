package resolve_test

import (
	"testing"

	"github.com/jamesprial/claudebot-mcp/internal/resolve"
	"github.com/jamesprial/claudebot-mcp/internal/testutil"
)

// ---------------------------------------------------------------------------
// ResolveChannelParam (exported helper in resolve package)
// ---------------------------------------------------------------------------

func Test_ResolveChannelParam_Cases(t *testing.T) {
	t.Parallel()

	md := testutil.NewMockDiscordSession(t)
	t.Cleanup(md.Close)

	r := resolve.New(md.Session, "guild-1")
	// Refresh to populate the cache from mock (returns general and random).
	if err := r.Refresh(); err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantID  string
		wantErr bool
	}{
		{
			name:    "all digits treated as ID",
			input:   "123456789012345678",
			wantID:  "123456789012345678",
			wantErr: false,
		},
		{
			name:    "channel name resolved to ID",
			input:   "general",
			wantID:  "ch-001",
			wantErr: false,
		},
		{
			name:    "hash-prefixed channel name",
			input:   "#general",
			wantID:  "ch-001",
			wantErr: false,
		},
		{
			name:    "empty input returns error",
			input:   "",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "unknown channel name returns error",
			input:   "nonexistent",
			wantID:  "",
			wantErr: true,
		},
		{
			name:    "mixed alphanumeric treated as name lookup",
			input:   "dev-chat",
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			id, err := resolve.ResolveChannelParam(r, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveChannelParam(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && id != tt.wantID {
				t.Errorf("ResolveChannelParam(%q) = %q, want %q", tt.input, id, tt.wantID)
			}
		})
	}
}
