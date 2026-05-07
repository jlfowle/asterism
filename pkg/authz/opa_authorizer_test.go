package authz

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestOPAAuthorizerUsesDefaultV1DataPath(t *testing.T) {
	var (
		mu   sync.Mutex
		seen []string
	)

	auth := NewOPAAuthorizer("http://opa.example", "")
	opa := auth.(*OPAAuthorizer)
	opa.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			mu.Lock()
			seen = append(seen, req.URL.Path)
			mu.Unlock()
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"result":{"allow":true}}`)),
				Request:    req,
			}, nil
		}),
	}

	allowed, reason, err := opa.IsAllowed(context.Background(), "user@example.com", []string{"asterism-users"}, "/api/v1/status", http.MethodGet)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected allow, got reason %q", reason)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 {
		t.Fatalf("expected one OPA request, got %d", len(seen))
	}
	if seen[0] != "/v1/data/asterism/authz/allow" {
		t.Fatalf("expected default OPA path, got %q", seen[0])
	}
}

func TestOPAAuthorizerDeniesWithoutLocalFallback(t *testing.T) {
	auth := NewOPAAuthorizer("http://opa.example", "admin")
	opa := auth.(*OPAAuthorizer)
	opa.client = &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"result":{"allow":false}}`)),
				Request:    req,
			}, nil
		}),
	}

	allowed, reason, err := opa.IsAllowed(context.Background(), "user@example.com", []string{"admin"}, "/api/v1/status", http.MethodGet)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Fatalf("expected deny when OPA says no")
	}
	if reason != "denied" {
		t.Fatalf("expected OPA denial reason, got %q", reason)
	}
}
