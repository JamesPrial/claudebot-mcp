package safety

import (
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// NewConfirmationTracker
// ---------------------------------------------------------------------------

func Test_NewConfirmationTracker_NilTools(t *testing.T) {
	t.Parallel()
	ct := NewConfirmationTracker(nil)
	if ct == nil {
		t.Fatal("NewConfirmationTracker(nil) should return non-nil tracker")
	}
}

func Test_NewConfirmationTracker_EmptyTools(t *testing.T) {
	t.Parallel()
	ct := NewConfirmationTracker([]string{})
	if ct == nil {
		t.Fatal("NewConfirmationTracker([]) should return non-nil tracker")
	}
}

// ---------------------------------------------------------------------------
// NeedsConfirmation
// ---------------------------------------------------------------------------

func Test_NeedsConfirmation_Cases(t *testing.T) {
	t.Parallel()

	destructive := []string{"discord_delete_message", "discord_ban_member"}
	ct := NewConfirmationTracker(destructive)

	tests := []struct {
		name string
		tool string
		want bool
	}{
		{
			name: "registered destructive tool returns true",
			tool: "discord_delete_message",
			want: true,
		},
		{
			name: "another registered tool returns true",
			tool: "discord_ban_member",
			want: true,
		},
		{
			name: "unregistered tool returns false",
			tool: "discord_send_message",
			want: false,
		},
		{
			name: "empty string returns false",
			tool: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ct.NeedsConfirmation(tt.tool)
			if got != tt.want {
				t.Errorf("NeedsConfirmation(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// RequestConfirmation
// ---------------------------------------------------------------------------

func Test_RequestConfirmation_ReturnsNonEmptyToken(t *testing.T) {
	t.Parallel()
	ct := NewConfirmationTracker([]string{"discord_delete_message"})
	token := ct.RequestConfirmation("discord_delete_message", "channel-123", "delete a message")
	if token == "" {
		t.Error("RequestConfirmation should return a non-empty token")
	}
}

func Test_RequestConfirmation_ReturnsUniqueTokens(t *testing.T) {
	t.Parallel()
	ct := NewConfirmationTracker([]string{"discord_delete_message"})

	token1 := ct.RequestConfirmation("discord_delete_message", "ch1", "desc1")
	token2 := ct.RequestConfirmation("discord_delete_message", "ch2", "desc2")

	if token1 == token2 {
		t.Error("sequential RequestConfirmation calls should return unique tokens")
	}
}

// ---------------------------------------------------------------------------
// Confirm
// ---------------------------------------------------------------------------

func Test_Confirm_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(ct *ConfirmationTracker) string // returns token to confirm
		want  bool
	}{
		{
			name: "valid unused token returns true",
			setup: func(ct *ConfirmationTracker) string {
				return ct.RequestConfirmation("tool", "resource", "desc")
			},
			want: true,
		},
		{
			name: "already-used token returns false",
			setup: func(ct *ConfirmationTracker) string {
				token := ct.RequestConfirmation("tool", "resource", "desc")
				ct.Confirm(token) // consume it
				return token
			},
			want: false,
		},
		{
			name: "bogus token returns false",
			setup: func(ct *ConfirmationTracker) string {
				return "completely-bogus-token-12345"
			},
			want: false,
		},
		{
			name: "empty string returns false",
			setup: func(ct *ConfirmationTracker) string {
				return ""
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ct := NewConfirmationTracker([]string{"tool"})
			token := tt.setup(ct)
			got := ct.Confirm(token)
			if got != tt.want {
				t.Errorf("Confirm(%q) = %v, want %v", token, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Concurrency: 100 concurrent RequestConfirmation calls
// ---------------------------------------------------------------------------

func Test_RequestConfirmation_ConcurrentUniqueness(t *testing.T) {
	t.Parallel()

	ct := NewConfirmationTracker([]string{"tool"})
	const numGoroutines = 100

	tokens := make([]string, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			tokens[idx] = ct.RequestConfirmation("tool", "resource", "desc")
		}(i)
	}
	wg.Wait()

	seen := make(map[string]struct{}, numGoroutines)
	for i, tok := range tokens {
		if tok == "" {
			t.Errorf("goroutine %d returned empty token", i)
			continue
		}
		if _, exists := seen[tok]; exists {
			t.Errorf("duplicate token %q from goroutine %d", tok, i)
		}
		seen[tok] = struct{}{}
	}
}

// ---------------------------------------------------------------------------
// Concurrency: Confirm is single-use under concurrent access
// ---------------------------------------------------------------------------

func Test_Confirm_ConcurrentSingleUse(t *testing.T) {
	t.Parallel()

	ct := NewConfirmationTracker([]string{"tool"})
	token := ct.RequestConfirmation("tool", "resource", "desc")

	const numGoroutines = 50
	results := make([]bool, numGoroutines)
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx] = ct.Confirm(token)
		}(i)
	}
	wg.Wait()

	trueCount := 0
	for _, r := range results {
		if r {
			trueCount++
		}
	}
	if trueCount != 1 {
		t.Errorf("exactly 1 Confirm should succeed, got %d", trueCount)
	}
}
