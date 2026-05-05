package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	BaseURL    string
	Token      string
	Site       string
	ConsoleURL string
	APIPrefix  string
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type unifiEnvelope struct {
	Data []map[string]any `json:"data"`
}

func Probe(ctx context.Context) Snapshot {
	return ProbeWithClient(ctx, http.DefaultClient, ConfigFromEnv())
}

func ConfigFromEnv() Config {
	baseURL := firstNonEmpty(os.Getenv("UNIFI_API_BASE_URL"), os.Getenv("UNIFI_API_URL"))
	site := strings.TrimSpace(os.Getenv("UNIFI_SITE_ID"))
	if site == "" {
		site = "default"
	}

	return Config{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:      strings.TrimSpace(os.Getenv("UNIFI_API_TOKEN")),
		Site:       site,
		ConsoleURL: strings.TrimSpace(os.Getenv("UNIFI_CONSOLE_URL")),
		APIPrefix:  strings.TrimSpace(os.Getenv("UNIFI_NETWORK_API_PREFIX")),
	}
}

func ProbeWithClient(ctx context.Context, client HTTPDoer, cfg Config) Snapshot {
	observedAt := time.Now().UTC().Format(time.RFC3339)

	if cfg.BaseURL == "" {
		return Snapshot{
			Configured:         false,
			Reachable:          false,
			Message:            "UniFi is not connected to Asterism yet.",
			ObservedAt:         observedAt,
			Severity:           "unknown",
			RecommendedActions: []string{"Add the local UniFi Network API base URL and read-only token through External Secrets."},
			Controls:           Actions(),
			Summary: Summary{
				Title:       "UniFi needs connection details",
				Description: "Asterism can explain wireless and client status after the local UniFi Network API is configured.",
			},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	startedAt := time.Now()
	deviceItems, deviceStatus, deviceErr := getUniFiItems(ctx, client, cfg, "stat/device")
	clientItems, _, clientErr := getUniFiItems(ctx, client, cfg, "stat/sta")
	latencyMs := time.Since(startedAt).Milliseconds()
	if deviceErr != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           cfg.BaseURL,
			Message:            fmt.Sprintf("UniFi Network API request failed: %v", deviceErr),
			HTTPStatus:         deviceStatus,
			LatencyMs:          latencyMs,
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"Asterism could not read UniFi device status."},
			RecommendedActions: []string{"Confirm the UniFi Network API base URL, token, and site identifier."},
			AuthoritativeLinks: authoritativeLinks(cfg),
			Controls:           Actions(),
			Summary: Summary{
				Title:       "UniFi is unreachable",
				Description: "Wireless status is unavailable until the local UniFi Network API responds.",
			},
		}
	}

	deviceSummary := summarizeDevices(deviceItems)
	clientSummary := summarizeClients(clientItems)
	reasons := degradedReasons(deviceSummary, clientErr)
	severity := "ok"
	message := "UniFi Network is reachable."
	if len(reasons) > 0 {
		severity = "warning"
		message = "UniFi Network is reachable with items to review."
	}

	metrics := map[string]any{
		"site":                cfg.Site,
		"deviceCount":         deviceSummary.Total,
		"onlineDeviceCount":   deviceSummary.Online,
		"offlineDeviceCount":  deviceSummary.Offline,
		"updateDeviceCount":   deviceSummary.PendingUpdates,
		"accessPointCount":    deviceSummary.AccessPoints,
		"switchCount":         deviceSummary.Switches,
		"gatewayCount":        deviceSummary.Gateways,
		"clientCount":         clientSummary.Total,
		"wifiClientCount":     clientSummary.Wifi,
		"wiredClientCount":    clientSummary.Wired,
		"guestClientCount":    clientSummary.Guests,
		"clientsFetchSuccess": clientErr == nil,
		"statusClass":         fmt.Sprintf("%dxx", deviceStatus/100),
	}

	return Snapshot{
		Configured:         true,
		Reachable:          true,
		Endpoint:           cfg.BaseURL,
		Message:            message,
		HTTPStatus:         deviceStatus,
		LatencyMs:          latencyMs,
		ObservedAt:         observedAt,
		Severity:           severity,
		DegradedReasons:    reasons,
		RecommendedActions: recommendedActions(reasons),
		AuthoritativeLinks: authoritativeLinks(cfg),
		Controls:           Actions(),
		Metrics:            metrics,
		Summary: Summary{
			Title:       "UniFi Network overview",
			Description: "Read-only status from the local UniFi Network API.",
			Stats: []Stat{
				{Label: "Devices online", Value: fmt.Sprintf("%d/%d", deviceSummary.Online, deviceSummary.Total), State: stateForZero(deviceSummary.Offline)},
				{Label: "Clients", Value: strconv.Itoa(clientSummary.Total), State: "ok"},
				{Label: "Wi-Fi clients", Value: strconv.Itoa(clientSummary.Wifi), State: "ok"},
				{Label: "Pending updates", Value: strconv.Itoa(deviceSummary.PendingUpdates), State: stateForZero(deviceSummary.PendingUpdates)},
			},
			Highlights: highlights(deviceSummary, clientSummary, clientErr),
		},
	}
}

func Actions() []Control {
	return []Control{
		{
			ID:               "unifi.pause-guest-wifi",
			Label:            "Pause guest Wi-Fi",
			Description:      "Future guided action that would ask UniFi to pause a selected guest WLAN through its own API.",
			Risk:             "medium",
			Enabled:          false,
			RequiresApproval: true,
			AuditEvent:       "unifi.guided_action.pause_guest_wifi",
		},
		{
			ID:               "unifi.restart-access-point",
			Label:            "Restart access point",
			Description:      "Future guided action for a selected AP, with confirmation and audit logging.",
			Risk:             "medium",
			Enabled:          false,
			RequiresApproval: true,
			AuditEvent:       "unifi.guided_action.restart_access_point",
		},
	}
}

func getUniFiItems(ctx context.Context, client HTTPDoer, cfg Config, resource string) ([]map[string]any, int, error) {
	var lastStatus int
	var lastErr error
	for _, prefix := range apiPrefixes(cfg) {
		requestURL, err := joinUniFiURL(cfg.BaseURL, prefix, cfg.Site, resource)
		if err != nil {
			return nil, 0, err
		}

		items, status, err := requestItems(ctx, client, requestURL, cfg.Token)
		lastStatus = status
		if err == nil {
			return items, status, nil
		}
		lastErr = err
	}

	return nil, lastStatus, lastErr
}

func requestItems(ctx context.Context, client HTTPDoer, requestURL string, token string) ([]map[string]any, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return nil, resp.StatusCode, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}

	var envelope unifiEnvelope
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && envelope.Data != nil {
		return envelope.Data, resp.StatusCode, nil
	}

	var items []map[string]any
	if err := json.Unmarshal(bodyBytes, &items); err == nil {
		return items, resp.StatusCode, nil
	}

	return nil, resp.StatusCode, fmt.Errorf("unexpected UniFi response shape")
}

type deviceCounts struct {
	Total          int
	Online         int
	Offline        int
	PendingUpdates int
	AccessPoints   int
	Switches       int
	Gateways       int
}

func summarizeDevices(items []map[string]any) deviceCounts {
	var counts deviceCounts
	for _, item := range items {
		counts.Total++
		if intField(item, "state") == 1 || stringField(item, "state") == "CONNECTED" {
			counts.Online++
		} else {
			counts.Offline++
		}
		if boolField(item, "upgradable") || boolField(item, "upgradeable") {
			counts.PendingUpdates++
		}
		switch strings.ToLower(stringField(item, "type")) {
		case "uap":
			counts.AccessPoints++
		case "usw":
			counts.Switches++
		case "ugw", "uxg":
			counts.Gateways++
		}
	}
	return counts
}

type clientCounts struct {
	Total  int
	Wifi   int
	Wired  int
	Guests int
}

func summarizeClients(items []map[string]any) clientCounts {
	var counts clientCounts
	for _, item := range items {
		counts.Total++
		if boolField(item, "is_wired") {
			counts.Wired++
		} else {
			counts.Wifi++
		}
		if boolField(item, "is_guest") {
			counts.Guests++
		}
	}
	return counts
}

func degradedReasons(devices deviceCounts, clientErr error) []string {
	reasons := make([]string, 0)
	if devices.Offline > 0 {
		reasons = append(reasons, fmt.Sprintf("%d UniFi device(s) appear offline.", devices.Offline))
	}
	if devices.PendingUpdates > 0 {
		reasons = append(reasons, fmt.Sprintf("%d UniFi device(s) have updates available.", devices.PendingUpdates))
	}
	if clientErr != nil {
		reasons = append(reasons, "Client telemetry could not be read.")
	}
	return reasons
}

func recommendedActions(reasons []string) []string {
	if len(reasons) == 0 {
		return []string{"No operator action is needed right now."}
	}
	return []string{"Open UniFi Network to inspect affected devices before making changes."}
}

func highlights(devices deviceCounts, clients clientCounts, clientErr error) []string {
	items := []string{
		fmt.Sprintf("%d of %d UniFi devices are online.", devices.Online, devices.Total),
		fmt.Sprintf("%d active clients are visible to Asterism.", clients.Total),
	}
	if clientErr != nil {
		items = append(items, "Client details are stale or unavailable.")
	}
	return items
}

func authoritativeLinks(cfg Config) []Link {
	target := cfg.ConsoleURL
	if target == "" {
		target = cfg.BaseURL
	}
	if target == "" {
		return nil
	}
	return []Link{{Label: "Open UniFi Network", Href: target}}
}

func apiPrefixes(cfg Config) []string {
	if cfg.APIPrefix != "" {
		return []string{cfg.APIPrefix}
	}
	return []string{"/proxy/network/api", "/api"}
}

func joinUniFiURL(baseURL string, prefix string, site string, resource string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("base URL must include scheme and host")
	}

	prefix = strings.Trim(prefix, "/")
	resource = strings.Trim(resource, "/")
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + prefix + "/s/" + url.PathEscape(site) + "/" + resource
	return parsed.String(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func stringField(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func boolField(item map[string]any, key string) bool {
	value, ok := item[key]
	if !ok || value == nil {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true" || typed == "1"
	default:
		return fmt.Sprintf("%v", typed) == "1"
	}
}

func intField(item map[string]any, key string) int {
	value, ok := item[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case int:
		return typed
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func stateForZero(value int) string {
	if value == 0 {
		return "ok"
	}
	return "warning"
}
