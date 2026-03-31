package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

type versionPayload struct {
	GitVersion string `json:"gitVersion"`
}

type listPayload struct {
	Items []json.RawMessage `json:"items"`
}

type nodeListPayload struct {
	Items []struct {
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

func Probe(ctx context.Context) Snapshot {
	endpoint := strings.TrimSpace(os.Getenv("CLUSTER_API_URL"))
	if endpoint == "" {
		return Snapshot{
			Configured: false,
			Reachable:  false,
			Message:    "integration not configured",
		}
	}

	apiRoot, versionURL, parseErr := deriveURLs(endpoint)
	if parseErr != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("invalid cluster endpoint: %v", parseErr),
		}
	}

	tokenBytes, tokenErr := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if tokenErr != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("missing service account token: %v", tokenErr),
		}
	}

	caPool, caErr := readClusterCA()
	if caErr != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("invalid cluster CA: %v", caErr),
		}
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: caPool},
		},
	}

	startedAt := time.Now()
	versionResp, statusCode, requestErr := getJSON[versionPayload](ctx, client, versionURL, tokenBytes)
	latencyMs := time.Since(startedAt).Milliseconds()
	if requestErr != nil {
		return Snapshot{
			Configured: true,
			Reachable:  false,
			Endpoint:   endpoint,
			Message:    fmt.Sprintf("request failed: %v", requestErr),
			HTTPStatus: statusCode,
			LatencyMs:  latencyMs,
		}
	}

	nodeResp, _, nodesErr := getJSON[nodeListPayload](ctx, client, apiRoot+"/api/v1/nodes", tokenBytes)
	podResp, _, podsErr := getJSON[listPayload](ctx, client, apiRoot+"/api/v1/pods", tokenBytes)

	healthyNodes := 0
	for _, node := range nodeResp.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				healthyNodes++
				break
			}
		}
	}

	metrics := map[string]any{
		"clusterVersion":   versionResp.GitVersion,
		"nodeCount":        len(nodeResp.Items),
		"readyNodeCount":   healthyNodes,
		"podCount":         len(podResp.Items),
		"nodesFetchSuccess": nodesErr == nil,
		"podsFetchSuccess":  podsErr == nil,
	}

	if statusCode >= 200 && statusCode < 400 {
		return Snapshot{
			Configured: true,
			Reachable:  true,
			Endpoint:   endpoint,
			Message:    "cluster API reachable",
			HTTPStatus: statusCode,
			LatencyMs:  latencyMs,
			Metrics:    metrics,
		}
	}

	return Snapshot{
		Configured: true,
		Reachable:  false,
		Endpoint:   endpoint,
		Message:    fmt.Sprintf("cluster API returned status %d", statusCode),
		HTTPStatus: statusCode,
		LatencyMs:  latencyMs,
		Metrics:    metrics,
	}
}

func deriveURLs(endpoint string) (apiRoot string, versionURL string, err error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", "", err
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("endpoint must include scheme and host")
	}

	apiRoot = parsed.Scheme + "://" + parsed.Host
	trimmedPath := strings.TrimSpace(parsed.Path)
	if trimmedPath == "" || trimmedPath == "/" {
		versionURL = apiRoot + "/version"
	} else {
		versionURL = endpoint
	}

	return apiRoot, versionURL, nil
}

func getJSON[T any](ctx context.Context, client *http.Client, requestURL string, token []byte) (T, int, error) {
	var zero T

	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, requestURL, nil)
	if err != nil {
		return zero, 0, err
	}

	req.Header.Set("Authorization", "Bearer "+string(token))
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return zero, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return zero, resp.StatusCode, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var payload T
	if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
		return zero, resp.StatusCode, decodeErr
	}

	return payload, resp.StatusCode, nil
}

func readClusterCA() (*x509.CertPool, error) {
	caBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("unable to parse certificate")
	}

	return pool, nil
}
