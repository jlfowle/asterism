package integration

import (
	"encoding/json"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Snapshot struct {
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	Endpoint   string `json:"endpoint,omitempty"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	LatencyMs  int64  `json:"latencyMs,omitempty"`
	Metrics    any    `json:"metrics,omitempty"`
}

func Probe(ctx context.Context) Snapshot {
	endpoint := os.Getenv("PFSENSE_API_URL")
	token := os.Getenv("PFSENSE_API_TOKEN")

	if endpoint == "" {
		return Snapshot{
			Configured: false,
			Reachable:  false,
			Message:    "integration not configured",
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("failed to create request: %v", err),
		}
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	startedAt := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	latencyMs := time.Since(startedAt).Milliseconds()
	metrics := map[string]any{
		"statusClass": fmt.Sprintf("%dxx", resp.StatusCode/100),
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr == nil {
			var payload map[string]any
			if unmarshalErr := json.Unmarshal(bodyBytes, &payload); unmarshalErr == nil {
				for _, key := range []string{"name", "version", "status", "uptime"} {
					if value, exists := payload[key]; exists {
						metrics[key] = value
					}
				}
			}
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return Snapshot{
			Configured: true,
			Reachable:  true,
			Endpoint:   endpoint,
			Message:    "upstream API reachable",
			HTTPStatus: resp.StatusCode,
			LatencyMs:  latencyMs,
			Metrics:    metrics,
		}
	}

	return Snapshot{
		Configured: true,
		Reachable:  false,
		Endpoint:   endpoint,
		Message:    fmt.Sprintf("upstream returned status %d", resp.StatusCode),
		HTTPStatus: resp.StatusCode,
		LatencyMs:  latencyMs,
		Metrics:    metrics,
	}
}
