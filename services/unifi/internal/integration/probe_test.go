package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestProbeWithClientSummarizesUniFiNetwork(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			var body string
			switch req.URL.Path {
			case "/proxy/network/api/s/default/stat/device":
				body = `{"data":[{"type":"uap","state":1},{"type":"usw","state":0,"upgradable":true}]}`
			case "/proxy/network/api/s/default/stat/sta":
				body = `{"data":[{"is_wired":false},{"is_wired":true,"is_guest":true}]}`
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":"not found"}`)),
					Request:    req,
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	}

	snapshot := ProbeWithClient(context.Background(), client, Config{
		BaseURL: "https://unifi.example",
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
