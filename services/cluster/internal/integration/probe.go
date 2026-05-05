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
	"strconv"
	"strings"
	"time"
)

type Snapshot struct {
	Configured         bool      `json:"configured"`
	Reachable          bool      `json:"reachable"`
	Endpoint           string    `json:"endpoint,omitempty"`
	Message            string    `json:"message"`
	HTTPStatus         int       `json:"httpStatus,omitempty"`
	LatencyMs          int64     `json:"latencyMs,omitempty"`
	ObservedAt         string    `json:"observedAt"`
	Severity           string    `json:"severity"`
	DegradedReasons    []string  `json:"degradedReasons,omitempty"`
	RecommendedActions []string  `json:"recommendedActions,omitempty"`
	AuthoritativeLinks []Link    `json:"authoritativeLinks,omitempty"`
	Controls           []Control `json:"controls,omitempty"`
	Metrics            any       `json:"metrics,omitempty"`
	Summary            Summary   `json:"summary,omitempty"`
}

type Summary struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Stats       []Stat   `json:"stats,omitempty"`
	Highlights  []string `json:"highlights,omitempty"`
}

type Stat struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit,omitempty"`
	State string `json:"state,omitempty"`
}

type Link struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Control struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	Description      string `json:"description"`
	Risk             string `json:"risk"`
	Enabled          bool   `json:"enabled"`
	RequiresApproval bool   `json:"requiresApproval"`
	AuditEvent       string `json:"auditEvent"`
}

type Config struct {
	Endpoint   string
	Token      string
	TokenPath  string
	CAPath     string
	ConsoleURL string
	ArgoCDURL  string
}

type versionPayload struct {
	GitVersion string `json:"gitVersion"`
}

type listPayload struct {
	Items []json.RawMessage `json:"items"`
}

type nodeListPayload struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Conditions []struct {
				Type   string `json:"type"`
				Status string `json:"status"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type deploymentListPayload struct {
	Items []struct {
		Status struct {
			Replicas          int `json:"replicas"`
			AvailableReplicas int `json:"availableReplicas"`
		} `json:"status"`
	} `json:"items"`
}

type podListPayload struct {
	Items []struct {
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	} `json:"items"`
}

type clusterVersionListPayload struct {
	Items []struct {
		Status struct {
			Desired struct {
				Version string `json:"version"`
			} `json:"desired"`
			Conditions []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type clusterOperatorListPayload struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Conditions []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	} `json:"items"`
}

type argoApplicationListPayload struct {
	Items []struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Status struct {
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
		} `json:"status"`
	} `json:"items"`
}

func Probe(ctx context.Context) Snapshot {
	return ProbeWithConfig(ctx, ConfigFromEnv())
}

func ConfigFromEnv() Config {
	return Config{
		Endpoint:   strings.TrimSpace(os.Getenv("CLUSTER_API_URL")),
		TokenPath:  firstNonEmpty(os.Getenv("CLUSTER_SERVICEACCOUNT_TOKEN_PATH"), "/var/run/secrets/kubernetes.io/serviceaccount/token"),
		CAPath:     firstNonEmpty(os.Getenv("CLUSTER_SERVICEACCOUNT_CA_PATH"), "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"),
		ConsoleURL: strings.TrimSpace(os.Getenv("CLUSTER_CONSOLE_URL")),
		ArgoCDURL:  strings.TrimSpace(os.Getenv("ARGOCD_CONSOLE_URL")),
	}
}

func ProbeWithConfig(ctx context.Context, cfg Config) Snapshot {
	observedAt := time.Now().UTC().Format(time.RFC3339)
	if cfg.Endpoint == "" {
		return Snapshot{
			Configured:         false,
			Reachable:          false,
			Message:            "OpenShift API monitoring is not configured yet.",
			ObservedAt:         observedAt,
			Severity:           "unknown",
			RecommendedActions: []string{"Set CLUSTER_API_URL to the in-cluster Kubernetes API version endpoint."},
			Controls:           Actions(),
			Summary: Summary{
				Title:       "OpenShift needs connection details",
				Description: "Asterism can explain cluster and GitOps health after the API endpoint is configured.",
			},
		}
	}

	apiRoot, versionURL, parseErr := deriveURLs(cfg.Endpoint)
	if parseErr != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           cfg.Endpoint,
			Message:            fmt.Sprintf("Invalid OpenShift endpoint: %v", parseErr),
			ObservedAt:         observedAt,
			Severity:           "critical",
			RecommendedActions: []string{"Use a full https URL for CLUSTER_API_URL."},
			Controls:           Actions(),
		}
	}

	tokenBytes, tokenErr := tokenBytes(cfg)
	if tokenErr != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           cfg.Endpoint,
			Message:            fmt.Sprintf("Missing service account token: %v", tokenErr),
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"The Cluster service cannot authenticate to the Kubernetes API."},
			RecommendedActions: []string{"Verify the dedicated cluster service account is mounted into the pod."},
			Controls:           Actions(),
		}
	}

	caPool, caErr := readClusterCA(cfg)
	if caErr != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           cfg.Endpoint,
			Message:            fmt.Sprintf("Invalid OpenShift API CA: %v", caErr),
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"The Cluster service cannot validate the Kubernetes API certificate."},
			RecommendedActions: []string{"Verify the service account CA bundle is mounted into the pod."},
			Controls:           Actions(),
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
			Configured:         true,
			Reachable:          false,
			Endpoint:           cfg.Endpoint,
			Message:            fmt.Sprintf("OpenShift API request failed: %v", requestErr),
			HTTPStatus:         statusCode,
			LatencyMs:          latencyMs,
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"Asterism could not read the OpenShift API version."},
			RecommendedActions: []string{"Confirm the in-cluster API endpoint and service account RBAC."},
			AuthoritativeLinks: authoritativeLinks(cfg),
			Controls:           Actions(),
			Summary: Summary{
				Title:       "OpenShift API is unreachable",
				Description: "Cluster status is unavailable until the Kubernetes API responds.",
			},
		}
	}

	nodeResp, _, nodesErr := getJSON[nodeListPayload](ctx, client, apiRoot+"/api/v1/nodes", tokenBytes)
	podResp, _, podsErr := getJSON[podListPayload](ctx, client, apiRoot+"/api/v1/pods", tokenBytes)
	namespaceResp, _, namespaceErr := getJSON[listPayload](ctx, client, apiRoot+"/api/v1/namespaces", tokenBytes)
	deploymentResp, _, deploymentsErr := getJSON[deploymentListPayload](ctx, client, apiRoot+"/apis/apps/v1/deployments", tokenBytes)
	clusterVersionResp, _, clusterVersionErr := getJSON[clusterVersionListPayload](ctx, client, apiRoot+"/apis/config.openshift.io/v1/clusterversions", tokenBytes)
	operatorResp, _, operatorsErr := getJSON[clusterOperatorListPayload](ctx, client, apiRoot+"/apis/config.openshift.io/v1/clusteroperators", tokenBytes)
	appResp, _, appsErr := getJSON[argoApplicationListPayload](ctx, client, apiRoot+"/apis/argoproj.io/v1alpha1/namespaces/openshift-gitops/applications", tokenBytes)

	healthyNodes := 0
	for _, node := range nodeResp.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				healthyNodes++
				break
			}
		}
	}
	runningPods, unhealthyPods := podPhaseCounts(podResp)
	availableDeployments, totalDeploymentReplicas := deploymentCounts(deploymentResp)
	operatorIssues := operatorIssueCount(operatorResp)
	appIssues := applicationIssueCount(appResp)
	clusterVersion := versionResp.GitVersion
	if len(clusterVersionResp.Items) > 0 && clusterVersionResp.Items[0].Status.Desired.Version != "" {
		clusterVersion = clusterVersionResp.Items[0].Status.Desired.Version
	}
	reasons := degradedReasons(nodeResp, healthyNodes, unhealthyPods, availableDeployments, totalDeploymentReplicas, deploymentsErr, operatorIssues, appsErr, appIssues, nodesErr, podsErr, operatorsErr)
	severity := "ok"
	message := "OpenShift API is reachable."
	if len(reasons) > 0 {
		severity = "warning"
		message = "OpenShift API is reachable with items to review."
	}

	metrics := map[string]any{
		"clusterVersion":               clusterVersion,
		"nodeCount":                    len(nodeResp.Items),
		"readyNodeCount":               healthyNodes,
		"namespaceCount":               len(namespaceResp.Items),
		"podCount":                     len(podResp.Items),
		"runningPodCount":              runningPods,
		"unhealthyPodCount":            unhealthyPods,
		"deploymentCount":              len(deploymentResp.Items),
		"availableDeploymentReplicas":  availableDeployments,
		"desiredDeploymentReplicas":    totalDeploymentReplicas,
		"clusterOperatorCount":         len(operatorResp.Items),
		"clusterOperatorIssueCount":    operatorIssues,
		"argoApplicationCount":         len(appResp.Items),
		"argoApplicationIssueCount":    appIssues,
		"nodesFetchSuccess":            nodesErr == nil,
		"podsFetchSuccess":             podsErr == nil,
		"namespacesFetchSuccess":       namespaceErr == nil,
		"deploymentsFetchSuccess":      deploymentsErr == nil,
		"clusterVersionFetchSuccess":   clusterVersionErr == nil,
		"operatorsFetchSuccess":        operatorsErr == nil,
		"argoApplicationsFetchSuccess": appsErr == nil,
	}

	if statusCode >= 200 && statusCode < 400 {
		return Snapshot{
			Configured:         true,
			Reachable:          true,
			Endpoint:           cfg.Endpoint,
			Message:            message,
			HTTPStatus:         statusCode,
			LatencyMs:          latencyMs,
			ObservedAt:         observedAt,
			Severity:           severity,
			DegradedReasons:    reasons,
			RecommendedActions: recommendedActions(reasons),
			AuthoritativeLinks: authoritativeLinks(cfg),
			Controls:           Actions(),
			Metrics:            metrics,
			Summary: Summary{
				Title:       "OpenShift and GitOps overview",
				Description: "Read-only status from the Kubernetes, OpenShift, and Argo CD APIs.",
				Stats: []Stat{
					{Label: "Nodes ready", Value: fmt.Sprintf("%d/%d", healthyNodes, len(nodeResp.Items)), State: stateForEquality(healthyNodes, len(nodeResp.Items))},
					{Label: "Pods running", Value: fmt.Sprintf("%d/%d", runningPods, len(podResp.Items)), State: stateForZero(unhealthyPods)},
					{Label: "Operators to review", Value: strconv.Itoa(operatorIssues), State: stateForZero(operatorIssues)},
					{Label: "GitOps apps to review", Value: strconv.Itoa(appIssues), State: stateForZero(appIssues)},
				},
				Highlights: highlights(clusterVersion, healthyNodes, len(nodeResp.Items), appResp, appsErr),
			},
		}
	}

	return Snapshot{
		Configured:         true,
		Reachable:          false,
		Endpoint:           cfg.Endpoint,
		Message:            fmt.Sprintf("OpenShift API returned status %d", statusCode),
		HTTPStatus:         statusCode,
		LatencyMs:          latencyMs,
		ObservedAt:         observedAt,
		Severity:           "critical",
		DegradedReasons:    []string{"The OpenShift API version endpoint returned an unsuccessful response."},
		RecommendedActions: []string{"Confirm the API endpoint and service account permissions."},
		AuthoritativeLinks: authoritativeLinks(cfg),
		Controls:           Actions(),
		Metrics:            metrics,
	}
}

func Actions() []Control {
	return []Control{
		{
			ID:               "cluster.sync-argocd-application",
			Label:            "Sync Argo CD application",
			Description:      "Future guided action that would ask Argo CD to sync a selected application after confirmation.",
			Risk:             "medium",
			Enabled:          false,
			RequiresApproval: true,
			AuditEvent:       "cluster.guided_action.sync_argocd_application",
		},
		{
			ID:               "cluster.restart-workload",
			Label:            "Restart workload",
			Description:      "Future guided action for an approved workload, routed through Kubernetes with audit logging.",
			Risk:             "medium",
			Enabled:          false,
			RequiresApproval: true,
			AuditEvent:       "cluster.guided_action.restart_workload",
		},
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

func tokenBytes(cfg Config) ([]byte, error) {
	if cfg.Token != "" {
		return []byte(cfg.Token), nil
	}
	return os.ReadFile(cfg.TokenPath)
}

func readClusterCA(cfg Config) (*x509.CertPool, error) {
	if cfg.CAPath == "" {
		return nil, nil
	}
	caBytes, err := os.ReadFile(cfg.CAPath)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caBytes) {
		return nil, fmt.Errorf("unable to parse certificate")
	}

	return pool, nil
}

func podPhaseCounts(payload podListPayload) (running int, unhealthy int) {
	for _, pod := range payload.Items {
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" {
			running++
		} else {
			unhealthy++
		}
	}
	return running, unhealthy
}

func deploymentCounts(payload deploymentListPayload) (available int, desired int) {
	for _, deployment := range payload.Items {
		available += deployment.Status.AvailableReplicas
		desired += deployment.Status.Replicas
	}
	return available, desired
}

func operatorIssueCount(payload clusterOperatorListPayload) int {
	issues := 0
	for _, operator := range payload.Items {
		if conditionStatus(operator.Status.Conditions, "Available") != "True" ||
			conditionStatus(operator.Status.Conditions, "Progressing") == "True" ||
			conditionStatus(operator.Status.Conditions, "Degraded") == "True" {
			issues++
		}
	}
	return issues
}

func applicationIssueCount(payload argoApplicationListPayload) int {
	issues := 0
	for _, app := range payload.Items {
		if app.Status.Health.Status != "Healthy" || app.Status.Sync.Status != "Synced" {
			issues++
		}
	}
	return issues
}

func conditionStatus(conditions []struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return ""
}

func degradedReasons(nodes nodeListPayload, readyNodes int, unhealthyPods int, availableDeployments int, desiredDeployments int, deploymentsErr error, operatorIssues int, appsErr error, appIssues int, nodesErr error, podsErr error, operatorsErr error) []string {
	reasons := make([]string, 0)
	if nodesErr != nil {
		reasons = append(reasons, "Node health could not be read.")
	} else if readyNodes < len(nodes.Items) {
		reasons = append(reasons, fmt.Sprintf("%d node(s) are not Ready.", len(nodes.Items)-readyNodes))
	}
	if podsErr != nil {
		reasons = append(reasons, "Pod health could not be read.")
	} else if unhealthyPods > 0 {
		reasons = append(reasons, fmt.Sprintf("%d pod(s) are not Running or Succeeded.", unhealthyPods))
	}
	if deploymentsErr != nil {
		reasons = append(reasons, "Deployment health could not be read.")
	} else if availableDeployments < desiredDeployments {
		reasons = append(reasons, fmt.Sprintf("%d deployment replica(s) are unavailable.", desiredDeployments-availableDeployments))
	}
	if operatorsErr != nil {
		reasons = append(reasons, "OpenShift operator health could not be read.")
	} else if operatorIssues > 0 {
		reasons = append(reasons, fmt.Sprintf("%d OpenShift operator(s) need attention.", operatorIssues))
	}
	if appsErr != nil {
		reasons = append(reasons, "Argo CD application status could not be read.")
	} else if appIssues > 0 {
		reasons = append(reasons, fmt.Sprintf("%d Argo CD application(s) are not healthy and synced.", appIssues))
	}
	return reasons
}

func recommendedActions(reasons []string) []string {
	if len(reasons) == 0 {
		return []string{"No operator action is needed right now."}
	}
	return []string{"Open OpenShift or Argo CD for the affected resource before making changes."}
}

func highlights(version string, readyNodes int, nodeCount int, apps argoApplicationListPayload, appsErr error) []string {
	items := []string{
		fmt.Sprintf("OpenShift %s is reachable.", version),
		fmt.Sprintf("%d of %d node(s) are Ready.", readyNodes, nodeCount),
	}
	if appsErr == nil {
		items = append(items, fmt.Sprintf("%d Argo CD application(s) are visible to Asterism.", len(apps.Items)))
	}
	return items
}

func authoritativeLinks(cfg Config) []Link {
	links := make([]Link, 0, 2)
	if cfg.ConsoleURL != "" {
		links = append(links, Link{Label: "Open OpenShift Console", Href: cfg.ConsoleURL})
	}
	if cfg.ArgoCDURL != "" {
		links = append(links, Link{Label: "Open Argo CD", Href: cfg.ArgoCDURL})
	}
	return links
}

func stateForZero(value int) string {
	if value == 0 {
		return "ok"
	}
	return "warning"
}

func stateForEquality(left int, right int) string {
	if left == right {
		return "ok"
	}
	return "warning"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
