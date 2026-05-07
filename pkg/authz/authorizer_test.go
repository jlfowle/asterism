package authz

import (
	"context"
	"testing"
)

func TestNewAuthorizer(t *testing.T) {
	tests := []struct {
		name          string
		requiredGroup string
		opaURL        string
	}{
		{
			name:          "empty required group, no OPA",
			requiredGroup: "",
			opaURL:        "",
		},
		{
			name:          "with required group and no OPA",
			requiredGroup: "admin",
			opaURL:        "",
		},
		{
			name:          "with required group and OPA URL",
			requiredGroup: "admin",
			opaURL:        "http://localhost:8181",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.opaURL != "" {
				t.Setenv("AUTHZ_OPA_URL", tt.opaURL)
			} else {
				t.Setenv("AUTHZ_OPA_URL", "")
			}

			auth := NewAuthorizer(tt.requiredGroup)
			if auth == nil {
				t.Fatal("expected non-nil authorizer")
			}

			if tt.opaURL != "" {
				if _, ok := auth.(*OPAAuthorizer); !ok {
					t.Fatalf("expected OPA-backed authorizer when AUTHZ_OPA_URL is set, got %T", auth)
				}
			}
		})
	}
}

func TestLocalAuthorizerAlwaysAllowsWhenNoRequiredGroup(t *testing.T) {
	auth := &LocalAuthorizer{
		requiredGroup: "",
	}

	allowed, reason, err := auth.IsAllowed(context.Background(), "user1", []string{}, "/api/v1/test", "GET")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected IsAllowed to return true when requiredGroup is empty")
	}
	if reason != "" {
		t.Errorf("expected empty reason, got %q", reason)
	}
}

func TestLocalAuthorizerWithMatch(t *testing.T) {
	auth := &LocalAuthorizer{
		requiredGroup: "admin",
	}

	allowed, reason, err := auth.IsAllowed(context.Background(), "user1", []string{"admin", "users"}, "/api/v1/test", "GET")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected IsAllowed to return true for matching group")
	}
	if reason != "" {
		t.Errorf("expected empty reason for allowed access, got %q", reason)
	}
}

func TestLocalAuthorizerWithoutMatch(t *testing.T) {
	auth := &LocalAuthorizer{
		requiredGroup: "admin",
	}

	allowed, reason, err := auth.IsAllowed(context.Background(), "user1", []string{"users", "viewers"}, "/api/v1/test", "GET")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected IsAllowed to return false for non-matching groups")
	}
	if reason == "" {
		t.Error("expected non-empty denial reason")
	}
}

func TestLocalAuthorizerWithEmptyGroups(t *testing.T) {
	auth := &LocalAuthorizer{
		requiredGroup: "admin",
	}

	allowed, reason, err := auth.IsAllowed(context.Background(), "user1", []string{}, "/api/v1/test", "GET")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected IsAllowed to return false for empty groups")
	}
	if reason == "" {
		t.Error("expected non-empty denial reason")
	}
}

func TestLocalAuthorizerWithNilGroups(t *testing.T) {
	auth := &LocalAuthorizer{
		requiredGroup: "admin",
	}

	allowed, reason, err := auth.IsAllowed(context.Background(), "user1", nil, "/api/v1/test", "GET")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected IsAllowed to return false for nil groups")
	}
	if reason == "" {
		t.Error("expected non-empty denial reason")
	}
}
