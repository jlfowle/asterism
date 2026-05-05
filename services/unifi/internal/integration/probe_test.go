package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeWithClientSummarizesUniFiNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/proxy/network/api/s/default/stat/device":
			_, _ = fmt.Fprint(w, `{"data":[{"type":"uap","state":1},{"type":"usw","state":0,"upgradable":true}]}`)
		case "/proxy/network/api/s/default/stat/sta":
			_, _ = fmt.Fprint(w, `{"data":[{"is_wired":false},{"is_wired":true,"is_guest":true}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot := ProbeWithClient(context.Background(), server.Client(), Config{
		BaseURL: server.URL,
		Site:    "default",
	})

	if !snapshot.Configured || !snapshot.Reachable {
		t.Fatalf("expected reachable configured snapshot, got %#v", snapshot)
	}
	if snapshot.Severity != "warning" {
		t.Fatalf("expected warning severity for offline/update device, got %q", snapshot.Severity)
	}

	metrics := snapshot.Metrics.(map[string]any)
	if metrics["deviceCount"] != 2 {
		t.Fatalf("expected two devices, got %#v", metrics["deviceCount"])
	}
	if metrics["clientCount"] != 2 {
		t.Fatalf("expected two clients, got %#v", metrics["clientCount"])
	}
	if len(snapshot.Controls) == 0 {
		t.Fatalf("expected guided action contracts")
	}
}

func TestProbeWithClientReportsUnconfiguredUniFi(t *testing.T) {
	snapshot := ProbeWithClient(context.Background(), http.DefaultClient, Config{})

	if snapshot.Configured {
		t.Fatalf("expected unconfigured snapshot")
	}
	if snapshot.Severity != "unknown" {
		t.Fatalf("expected unknown severity, got %q", snapshot.Severity)
	}
}
