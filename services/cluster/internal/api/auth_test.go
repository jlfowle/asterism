package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddlewareDisabled(t *testing.T) {
	middleware := AuthMiddleware{
		mode:          "disabled",
		requiredGroup: "",
	}

	handler := middleware.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareEnforcedMissingPrincipal(t *testing.T) {
	middleware := AuthMiddleware{
		mode:          "enforced",
		requiredGroup: "",
	}

	handler := middleware.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}

func TestAuthMiddlewareEnforcedWithPrincipal(t *testing.T) {
	middleware := AuthMiddleware{
		mode:          "enforced",
		requiredGroup: "",
	}

	handler := middleware.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal := principalFromContext(r.Context())
		if principal != "user@example.com" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Forwarded-Email", "user@example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareReadPrincipalPriority(t *testing.T) {
	middleware := AuthMiddleware{
		mode:          "enforced",
		requiredGroup: "",
	}

	tests := []struct {
		name      string
		headers   map[string]string
		expected  string
	}{
		{
			name: "X-Asterism-Principal takes priority",
			headers: map[string]string{
				"X-Asterism-Principal": "principal@example.com",
				"X-Forwarded-User":      "user@example.com",
				"X-Forwarded-Email":     "email@example.com",
			},
			expected: "principal@example.com",
		},
		{
			name: "X-Forwarded-User fallback",
			headers: map[string]string{
				"X-Forwarded-User":  "user@example.com",
				"X-Forwarded-Email": "email@example.com",
			},
			expected: "user@example.com",
		},
		{
			name: "X-Forwarded-Email fallback",
			headers: map[string]string{
				"X-Forwarded-Email": "email@example.com",
			},
			expected: "email@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.Protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				principal := principalFromContext(r.Context())
				if principal != tt.expected {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte(principal))
					return
				}
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/api/test", nil)
			for key, val := range tt.headers {
				req.Header.Set(key, val)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestAuthMiddlewareReadGroups(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected []string
	}{
		{
			name:     "single group",
			header:   "admin",
			expected: []string{"admin"},
		},
		{
			name:     "multiple groups",
			header:   "admin,users,viewers",
			expected: []string{"admin", "users", "viewers"},
		},
		{
			name:     "groups with whitespace",
			header:   "admin, users , viewers",
			expected: []string{"admin", "users", "viewers"},
		},
		{
			name:     "empty header",
			header:   "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a custom middleware for testing group extraction
			m := &AuthMiddleware{mode: "disabled"}

			req := httptest.NewRequest("GET", "/api/test", nil)
			if tt.header != "" {
				req.Header.Set("X-Asterism-Groups", tt.header)
			}

			groups := m.readGroups(req)

			if len(groups) != len(tt.expected) {
				t.Errorf("expected %d groups, got %d: %v", len(tt.expected), len(groups), groups)
				return
			}

			for i, g := range groups {
				if g != tt.expected[i] {
					t.Errorf("group %d: expected %q, got %q", i, tt.expected[i], g)
				}
			}
		})
	}
}
