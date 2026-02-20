// Package auth provides HTTP middleware for bearer token authentication.
package auth

import (
	"log/slog"
	"net/http"
	"strings"
)

// NewAuthMiddleware returns an HTTP middleware that enforces bearer token
// authentication. If the configured token is empty, authentication is disabled
// and all requests pass through to the next handler unconditionally.
//
// When enabled, the middleware requires the incoming request to carry an
// Authorization header with the exact format:
//
//	Authorization: Bearer <token>
//
// The "Bearer" prefix is case-sensitive and must be followed by exactly one
// space before the token value. Any deviation — missing header, wrong token,
// lowercase prefix, extra spaces, or an empty token value — results in a 401
// Unauthorized response and the next handler is never called.
//
// logger is used to emit DEBUG-level messages on rejected requests. If nil,
// slog.Default() is used.
func NewAuthMiddleware(token string, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Auth disabled when no token is configured.
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")

			// Header must start with exactly "Bearer " (one space).
			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				logger.Debug("auth rejected: missing or malformed Authorization header", "remote", r.RemoteAddr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract the token portion after the prefix.
			provided := authHeader[len(prefix):]

			// The extracted portion must be non-empty and match exactly.
			if provided == "" || provided != token {
				logger.Debug("auth rejected: invalid token", "remote", r.RemoteAddr)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
