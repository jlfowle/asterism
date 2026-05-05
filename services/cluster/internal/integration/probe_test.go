package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProbeWithConfigSummarizesClusterAndGitOps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/version":
			_, _ = fmt.Fprint(w, `{"gitVersion":"v1.34.6"}`)
		case "/api/v1/nodes":
			_, _ = fmt.Fprint(w, `{"items":[{"metadata":{"name":"node-a"},"status":{"conditions":[{"type":"Ready","status":"True"}]}}]}`)
		case "/api/v1/pods":
			_, _ = fmt.Fprint(w, `{"items":[{"status":{"phase":"Running"}},{"status":{"phase":"Failed"}}]}`)
		case "/api/v1/namespaces":
			_, _ = fmt.Fprint(w, `{"items":[{},{}]}`)
		case "/apis/apps/v1/deployments":
			_, _ = fmt.Fprint(w, `{"items":[{"status":{"replicas":2,"availableReplicas":1}}]}`)
		case "/apis/config.openshift.io/v1/clusterversions":
			_, _ = fmt.Fprint(w, `{"items":[{"status":{"desired":{"version":"4.21.11"}}}]}`)
		case "/apis/config.openshift.io/v1/clusteroperators":
			_, _ = fmt.Fprint(w, `{"items":[{"metadata":{"name":"ingress"},"status":{"conditions":[{"type":"Available","status":"True"},{"type":"Progressing","status":"False"},{"type":"Degraded","status":"False"}]}}]}`)
		case "/apis/argoproj.io/v1alpha1/namespaces/openshift-gitops/applications":
			_, _ = fmt.Fprint(w, `{"items":[{"metadata":{"name":"app-asterism"},"status":{"health":{"status":"Healthy"},"sync":{"status":"Synced"}}}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	snapshot := ProbeWithConfig(context.Background(), Config{
		Endpoint: server.URL + "/version",
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
