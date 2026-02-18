package safety

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const tokenTTL = 5 * time.Minute

// pendingConfirmation holds the metadata for an outstanding confirmation token.
type pendingConfirmation struct {
	tool         string
	resourceName string
	description  string
	createdAt    time.Time
}

// ConfirmationTracker manages single-use, time-limited confirmation tokens for
// destructive tool invocations.
type ConfirmationTracker struct {
	destructive map[string]struct{}

	mu     sync.Mutex
	tokens map[string]*pendingConfirmation
}

// NewConfirmationTracker returns a ConfirmationTracker whose set of tools
// requiring explicit confirmation is defined by destructiveTools. A nil or
// empty slice means no tools require confirmation.
func NewConfirmationTracker(destructiveTools []string) *ConfirmationTracker {
	ct := &ConfirmationTracker{
		destructive: make(map[string]struct{}, len(destructiveTools)),
		tokens:      make(map[string]*pendingConfirmation),
	}
	for _, tool := range destructiveTools {
		ct.destructive[tool] = struct{}{}
	}
	return ct
}

// NeedsConfirmation reports whether tool is in the destructive-tools set.
func (ct *ConfirmationTracker) NeedsConfirmation(tool string) bool {
	_, ok := ct.destructive[tool]
	return ok
}

// sweepExpired removes all tokens whose age exceeds tokenTTL. The caller must
// hold ct.mu.
func (ct *ConfirmationTracker) sweepExpired() {
	for token, pending := range ct.tokens {
		if time.Since(pending.createdAt) > tokenTTL {
			delete(ct.tokens, token)
		}
	}
}

// RequestConfirmation creates a new confirmation token for the given tool,
// resource, and description and returns the opaque token string. Tokens are
// valid for 5 minutes and are single-use.
func (ct *ConfirmationTracker) RequestConfirmation(tool, resourceName, description string) string {
	token := generateToken()

	ct.mu.Lock()
	ct.sweepExpired()
	ct.tokens[token] = &pendingConfirmation{
		tool:         tool,
		resourceName: resourceName,
		description:  description,
		createdAt:    time.Now(),
	}
	ct.mu.Unlock()

	return token
}

// Confirm consumes the given token and returns true if it was valid and
// unexpired. Subsequent calls with the same token return false.
func (ct *ConfirmationTracker) Confirm(token string) bool {
	if token == "" {
		return false
	}

	ct.mu.Lock()
	defer ct.mu.Unlock()

	pending, ok := ct.tokens[token]
	if !ok {
		return false
	}

	// Remove the token immediately (single-use).
	delete(ct.tokens, token)

	// Check expiry.
	if time.Since(pending.createdAt) > tokenTTL {
		return false
	}

	return true
}

// generateToken returns a cryptographically random hex-encoded token string.
func generateToken() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a timestamp-based value if crypto/rand is unavailable.
		// This should never happen in practice.
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(b[:])
}
