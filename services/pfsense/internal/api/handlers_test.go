package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type statusTestResponse struct {
	Service   string `json:"service"`
	Principal string `json:"principal"`
}

func TestHealthz(t *testing.T) {
	h := NewHandler("pfsense")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestServiceUIAssets(t *testing.T) {
	h := NewHandler("pfsense")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	moduleReq := httptest.NewRequest(http.MethodGet, "/ui/module.json", nil)
	moduleRes := httptest.NewRecorder()
	mux.ServeHTTP(moduleRes, moduleReq)

	if moduleRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for module manifest, got %d", moduleRes.Code)
	}

	if !strings.Contains(moduleRes.Body.String(), "\"service\": \"pfsense\"") {
		t.Fatalf("expected service module manifest to reference pfsense, got %q", moduleRes.Body.String())
	}

	cardReq := httptest.NewRequest(http.MethodGet, "/ui/dashboard-card.js", nil)
	cardRes := httptest.NewRecorder()
	mux.ServeHTTP(cardRes, cardReq)

	if cardRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for dashboard card module, got %d", cardRes.Code)
	}

	if !strings.Contains(cardRes.Body.String(), "createDashboardCard") {
		t.Fatalf("expected dashboard card module export, got %q", cardRes.Body.String())
	}
}

func TestStatusUnauthorizedWhenMissingPrincipal(t *testing.T) {
	t.Setenv("AUTH_MODE", "enforced")
	t.Setenv("AUTH_REQUIRED_GROUP", "")

	h := NewHandler("pfsense")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
}

func TestStatusAuthorizedWithPrincipal(t *testing.T) {
	t.Setenv("AUTH_MODE", "enforced")
	t.Setenv("AUTH_REQUIRED_GROUP", "")
	t.Setenv("PFSENSE_API_URL", "")

	h := NewHandler("pfsense")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req.Header.Set("X-Asterism-Principal", "test-user")

	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload statusTestResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal status response: %v", err)
	}

	if payload.Service != "pfsense" {
		t.Fatalf("expected service %q, got %q", "pfsense", payload.Service)
	}

	if payload.Principal != "test-user" {
		t.Fatalf("expected principal %q, got %q", "test-user", payload.Principal)
	}
}
