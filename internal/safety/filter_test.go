package safety

import "testing"

func Test_Filter_IsAllowed_Cases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		allowlist []string
		denylist  []string
		resource  string
		want      bool
	}{
		{
			name:      "empty lists allow everything",
			allowlist: []string{},
			denylist:  []string{},
			resource:  "anything",
			want:      true,
		},
		{
			name:      "nil lists allow everything",
			allowlist: nil,
			denylist:  nil,
			resource:  "anything",
			want:      true,
		},
		{
			name:      "allowlist match allows resource",
			allowlist: []string{"general"},
			denylist:  []string{},
			resource:  "general",
			want:      true,
		},
		{
			name:      "allowlist miss denies resource",
			allowlist: []string{"general"},
			denylist:  []string{},
			resource:  "random",
			want:      false,
		},
		{
			name:      "denylist match denies resource",
			allowlist: []string{},
			denylist:  []string{"admin"},
			resource:  "admin",
			want:      false,
		},
		{
			name:      "denylist miss allows resource",
			allowlist: []string{},
			denylist:  []string{"admin"},
			resource:  "general",
			want:      true,
		},
		{
			name:      "deny wins over allow",
			allowlist: []string{"general", "admin"},
			denylist:  []string{"admin"},
			resource:  "admin",
			want:      false,
		},
		{
			name:      "glob pattern in allowlist matches",
			allowlist: []string{"bot-*"},
			denylist:  []string{},
			resource:  "bot-commands",
			want:      true,
		},
		{
			name:      "glob pattern in allowlist does not match",
			allowlist: []string{"bot-*"},
			denylist:  []string{},
			resource:  "general",
			want:      false,
		},
		{
			name:      "wildcard allow with specific deny blocks denied",
			allowlist: []string{"*"},
			denylist:  []string{"admin"},
			resource:  "admin",
			want:      false,
		},
		{
			name:      "wildcard allow with specific deny allows others",
			allowlist: []string{"*"},
			denylist:  []string{"admin"},
			resource:  "general",
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := NewFilter(tt.allowlist, tt.denylist)
			got := f.IsAllowed(tt.resource)
			if got != tt.want {
				t.Errorf("NewFilter(%v, %v).IsAllowed(%q) = %v, want %v",
					tt.allowlist, tt.denylist, tt.resource, got, tt.want)
			}
		})
	}
}

func Test_Filter_IsAllowed_EmptyResource(t *testing.T) {
	t.Parallel()
	// Empty resource with empty lists should be allowed
	f := NewFilter(nil, nil)
	if !f.IsAllowed("") {
		t.Error("empty resource with nil lists should be allowed")
	}
}

func Test_Filter_IsAllowed_GlobPatternInDenylist(t *testing.T) {
	t.Parallel()
	f := NewFilter(nil, []string{"secret-*"})
	if f.IsAllowed("secret-channel") {
		t.Error("resource matching deny glob should be denied")
	}
	if !f.IsAllowed("public-channel") {
		t.Error("resource not matching deny glob should be allowed")
	}
}
