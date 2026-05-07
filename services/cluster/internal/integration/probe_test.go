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

func TestProbeWithConfigSummarizesClusterAndGitOps(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Authorization") != "Bearer test-token" {
				return &http.Response{
					StatusCode: http.StatusUnauthorized,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":"missing token"}`)),
					Request:    req,
				}, nil
			}

			var body string
			switch req.URL.Path {
			case "/version":
				body = `{"gitVersion":"v1.34.6"}`
			case "/api/v1/nodes":
				body = `{"items":[{"metadata":{"name":"node-a"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`
			case "/api/v1/pods":
				body = `{"items":[{"status":{"phase":"Running"}},{"status":{"phase":"Failed"}}]}`
			case "/api/v1/namespaces":
				body = `{"items":[{},{}]}`
			case "/apis/apps/v1/deployments":
				body = `{"items":[{"status":{"replicas":2,"availableReplicas":1}}]}`
			case "/apis/config.openshift.io/v1/clusterversions":
				body = `{"items":[{"status":{"desired":{"version":"4.21.11"}}}]}`
			case "/apis/config.openshift.io/v1/clusteroperators":
				body = `{"items":[{"metadata":{"name":"ingress"},"status":{"conditions":[{"type":"Available","status":"True"},{"type":"Progressing","status":"False"},{"type":"Degraded","status":"False"}]}}]}`
			case "/apis/argoproj.io/v1alpha1/namespaces/openshift-gitops/applications":
				body = `{"items":[{"metadata":{"name":"app-asterism"},"status":{"health":{"status":"Healthy"},"sync":{"status":"Synced"}}}]}`
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
		Endpoint: "https://cluster.example/version",
		Token:    "test-token",
	})

	if !snapshot.Configured || !snapshot.Reachable {
		t.Fatalf("expected reachable configured snapshot, got %#v", snapshot)
	}
	if snapshot.Severity != "warning" {
		t.Fatalf("expected warning severity for failed pod, got %q", snapshot.Severity)
	}

	metrics := snapshot.Metrics.(map[string]any)
	if metrics["clusterVersion"] != "4.21.11" {
		t.Fatalf("expected OpenShift version metric, got %#v", metrics["clusterVersion"])
	}
	if metrics["unhealthyPodCount"] != 1 {
		t.Fatalf("expected one unhealthy pod, got %#v", metrics["unhealthyPodCount"])
	}
}

func TestProbeWithConfigReportsUnconfiguredCluster(t *testing.T) {
	snapshot := ProbeWithConfig(context.Background(), Config{})

	if snapshot.Configured {
		t.Fatalf("expected unconfigured snapshot")
	}
	if snapshot.Severity != "unknown" {
		t.Fatalf("expected unknown severity, got %q", snapshot.Severity)
	}
}
