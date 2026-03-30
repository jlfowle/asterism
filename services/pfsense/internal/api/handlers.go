package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jlfowle/asterism/services/pfsense/internal/integration"
	uiassets "github.com/jlfowle/asterism/services/pfsense/ui"
)

type Handler struct {
	serviceName string
	auth        AuthMiddleware
}

type statusResponse struct {
	Service     string               `json:"service"`
	Status      string               `json:"status"`
	Principal   string               `json:"principal,omitempty"`
	Integration integration.Snapshot `json:"integration"`
	Timestamp   string               `json:"timestamp"`
}

func NewHandler(serviceName string) Handler {
	return Handler{
		serviceName: serviceName,
		auth:        NewAuthMiddlewareFromEnv(),
	}
}

func (h Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("/ui/", http.StripPrefix("/ui/", http.FileServer(http.FS(uiassets.Files))))
	mux.HandleFunc("/healthz", h.healthz)
	mux.Handle("/api/v1/status", h.auth.Protect(http.HandlerFunc(h.status)))
}

func (h Handler) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h Handler) status(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{
		Service:     h.serviceName,
		Status:      "ready",
		Principal:   principalFromContext(r.Context()),
		Integration: integration.Probe(r.Context()),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
