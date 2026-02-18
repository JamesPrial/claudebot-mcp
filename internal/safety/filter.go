// Package safety provides filtering, confirmation, and audit logging for
// destructive or sensitive claudebot-mcp operations.
package safety

import "path/filepath"

// Filter controls access to named resources using an allowlist and a denylist.
// Glob patterns (as understood by filepath.Match) are supported in both lists.
//
// Rules:
//   - If both lists are empty (or nil), every resource is allowed.
//   - Denylist always takes priority over the allowlist.
//   - If a non-empty allowlist is present, a resource must match at least one
//     allowlist pattern to be permitted (after the denylist check).
type Filter struct {
	allowlist []string
	denylist  []string
}

// NewFilter constructs a Filter from the provided allowlist and denylist
// pattern slices. Either or both may be nil or empty.
func NewFilter(allowlist, denylist []string) *Filter {
	return &Filter{
		allowlist: allowlist,
		denylist:  denylist,
	}
}

// IsAllowed reports whether name is permitted by this filter.
func (f *Filter) IsAllowed(name string) bool {
	// Denylist wins first.
	for _, pattern := range f.denylist {
		if matchGlob(pattern, name) {
			return false
		}
	}

	// If the allowlist is empty (or nil), everything not denied is allowed.
	if len(f.allowlist) == 0 {
		return true
	}

	// Resource must match at least one allowlist pattern.
	for _, pattern := range f.allowlist {
		if matchGlob(pattern, name) {
			return true
		}
	}

	return false
}

// matchGlob returns true when name matches the given glob pattern.
// filepath.Match errors (malformed patterns) are treated as non-matching.
func matchGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}
