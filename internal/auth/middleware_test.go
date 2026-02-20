package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_NewAuthMiddleware_Cases(t *testing.T) {
	t.Parallel()

	// successHandler is the inner handler that returns 200 OK.
	successHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	tests := []struct {
		name           string
		configToken    string
		authHeader     string
		wantStatusCode int
	}{
		{
			name:           "correct token returns 200",
			configToken:    "correct-token",
			authHeader:     "Bearer correct-token",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "missing header returns 401",
			configToken:    "correct-token",
			authHeader:     "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "wrong token returns 401",
			configToken:    "correct-token",
			authHeader:     "Bearer wrong-token",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "non-Bearer scheme returns 401",
			configToken:    "correct-token",
			authHeader:     "NotBearer token",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "empty config token with no header returns 200 (auth disabled)",
			configToken:    "",
			authHeader:     "",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "empty config token with Bearer header returns 200 (auth disabled)",
			configToken:    "",
			authHeader:     "Bearer anything",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Bearer prefix only returns 401",
			configToken:    "correct-token",
			authHeader:     "Bearer ",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "lowercase bearer returns 401",
			configToken:    "correct-token",
			authHeader:     "bearer correct-token",
			wantStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			middleware := NewAuthMiddleware(tt.configToken, nil)
			handler := middleware(successHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatusCode)
			}
		})
	}
}

func Test_NewAuthMiddleware_PassesThroughToHandler(t *testing.T) {
	t.Parallel()

	// Verify that on success, the inner handler actually executes.
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := NewAuthMiddleware("my-token", nil)
	handler := middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer my-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called on valid auth")
	}
}

func Test_NewAuthMiddleware_BlocksInnerHandler(t *testing.T) {
	t.Parallel()

	// Verify that on failure, the inner handler does NOT execute.
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	middleware := NewAuthMiddleware("my-token", nil)
	handler := middleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("inner handler was called despite invalid auth")
	}
}
