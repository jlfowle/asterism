package integration

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
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
	Host                 string
	Port                 uint16
	Community            string
	ConsoleURL           string
	ExpectedUpInterfaces []string
	Timeout              time.Duration
}

type SNMPClient interface {
	Get(oids []string) (*gosnmp.SnmpPacket, error)
	WalkAll(rootOid string) ([]gosnmp.SnmpPDU, error)
}

func Probe(ctx context.Context) Snapshot {
	return ProbeWithClient(ctx, newSNMPClient, ConfigFromEnv())
}

func ConfigFromEnv() Config {
	port := uint16(161)
	if rawPort := strings.TrimSpace(os.Getenv("PFSENSE_SNMP_PORT")); rawPort != "" {
		if parsed, err := strconv.ParseUint(rawPort, 10, 16); err == nil {
			port = uint16(parsed)
		}
	}

	timeout := 5 * time.Second
	if rawTimeout := strings.TrimSpace(os.Getenv("PFSENSE_SNMP_TIMEOUT")); rawTimeout != "" {
		if parsed, err := time.ParseDuration(rawTimeout); err == nil {
			timeout = parsed
		}
	}

	return Config{
		Host:                 strings.TrimSpace(os.Getenv("PFSENSE_SNMP_HOST")),
		Port:                 port,
		Community:            strings.TrimSpace(os.Getenv("PFSENSE_SNMP_COMMUNITY")),
		ConsoleURL:           strings.TrimSpace(os.Getenv("PFSENSE_CONSOLE_URL")),
		ExpectedUpInterfaces: splitCSV(os.Getenv("PFSENSE_EXPECTED_UP_INTERFACES")),
		Timeout:              timeout,
	}
}

func ProbeWithClient(ctx context.Context, clientFactory func(Config) (SNMPClient, func() error, error), cfg Config) Snapshot {
	observedAt := time.Now().UTC().Format(time.RFC3339)

	if cfg.Host == "" || cfg.Community == "" {
		return Snapshot{
			Configured:         false,
			Reachable:          false,
			Message:            "pfSense read-only monitoring is not configured yet.",
			ObservedAt:         observedAt,
			Severity:           "unknown",
			RecommendedActions: []string{"Enable pfSense SNMP with a non-default read community and restrict access to Asterism."},
			Controls:           Actions(),
			Summary: Summary{
				Title:       "pfSense needs SNMP details",
				Description: "Asterism can show firewall and gateway health after read-only SNMP is configured.",
			},
		}
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	startedAt := time.Now()
	client, closeClient, err := clientFactory(cfg)
	if err != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           endpoint(cfg),
			Message:            fmt.Sprintf("pfSense SNMP connection failed: %v", err),
			LatencyMs:          time.Since(startedAt).Milliseconds(),
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"Asterism could not connect to pfSense SNMP."},
			RecommendedActions: []string{"Confirm pfSense SNMP host, community, firewall rules, and interface binding."},
			AuthoritativeLinks: authoritativeLinks(cfg),
			Controls:           Actions(),
			Summary: Summary{
				Title:       "pfSense is unreachable",
				Description: "Gateway status is unavailable until SNMP responds.",
			},
		}
	}
	defer func() { _ = closeClient() }()

	type result struct {
		system     *gosnmp.SnmpPacket
		interfaces []interfaceState
		err        error
	}
	resultCh := make(chan result, 1)
	go func() {
		systemPacket, err := client.Get([]string{
			".1.3.6.1.2.1.1.1.0",
			".1.3.6.1.2.1.1.3.0",
			".1.3.6.1.2.1.1.5.0",
		})
		if err != nil {
			resultCh <- result{err: err}
			return
		}
		interfaces, err := readInterfaces(client)
		resultCh <- result{system: systemPacket, interfaces: interfaces, err: err}
	}()

	var probeResult result
	select {
	case <-ctx.Done():
		probeResult.err = ctx.Err()
	case probeResult = <-resultCh:
	}
	latencyMs := time.Since(startedAt).Milliseconds()

	if probeResult.err != nil {
		return Snapshot{
			Configured:         true,
			Reachable:          false,
			Endpoint:           endpoint(cfg),
			Message:            fmt.Sprintf("pfSense SNMP request failed: %v", probeResult.err),
			LatencyMs:          latencyMs,
			ObservedAt:         observedAt,
			Severity:           "critical",
			DegradedReasons:    []string{"Asterism could not read pfSense SNMP status."},
			RecommendedActions: []string{"Check SNMP service, read community, and pfSense firewall rules."},
			AuthoritativeLinks: authoritativeLinks(cfg),
			Controls:           Actions(),
			Summary: Summary{
				Title:       "pfSense SNMP is not responding",
				Description: "Firewall status is unavailable until the read-only SNMP endpoint responds.",
			},
		}
	}

	system := parseSystem(probeResult.system)
	interfaceSummary := summarizeInterfaces(probeResult.interfaces, cfg.ExpectedUpInterfaces)
	reasons := interfaceSummary.Reasons
	severity := "ok"
	message := "pfSense SNMP is reachable."
	if len(reasons) > 0 {
		severity = "warning"
		message = "pfSense SNMP is reachable with gateway items to review."
	}

	metrics := map[string]any{
		"systemName":                system.Name,
		"systemDescription":         system.Description,
		"uptimeSeconds":             system.UptimeSeconds,
		"interfaceCount":            len(probeResult.interfaces),
		"upInterfaceCount":          interfaceSummary.Up,
		"downInterfaceCount":        interfaceSummary.Down,
		"expectedInterfaceCount":    len(cfg.ExpectedUpInterfaces),
		"expectedInterfacesHealthy": len(interfaceSummary.Reasons) == 0,
	}

	return Snapshot{
		Configured:         true,
		Reachable:          true,
		Endpoint:           endpoint(cfg),
		Message:            message,
		LatencyMs:          latencyMs,
		ObservedAt:         observedAt,
		Severity:           severity,
		DegradedReasons:    reasons,
		RecommendedActions: recommendedActions(reasons),
		AuthoritativeLinks: authoritativeLinks(cfg),
		Controls:           Actions(),
		Metrics:            metrics,
		Summary: Summary{
			Title:       "pfSense gateway overview",
			Description: "Read-only status from pfSense SNMP.",
			Stats: []Stat{
				{Label: "SNMP", Value: "reachable", State: "ok"},
				{Label: "Interfaces up", Value: fmt.Sprintf("%d/%d", interfaceSummary.Up, len(probeResult.interfaces)), State: stateForReasons(reasons)},
				{Label: "Expected links", Value: fmt.Sprintf("%d", len(cfg.ExpectedUpInterfaces)), State: stateForReasons(reasons)},
				{Label: "Uptime", Value: formatDuration(system.UptimeSeconds), State: "ok"},
			},
			Highlights: highlights(system, interfaceSummary, cfg.ExpectedUpInterfaces),
		},
	}
}

func Actions() []Control {
	return []Control{
		{
			ID:               "pfsense.renew-wan",
			Label:            "Renew WAN lease",
			Description:      "Future guided action that would ask pfSense to renew a selected WAN DHCP lease through a backend-owned API.",
			Risk:             "medium",
			Enabled:          false,
			RequiresApproval: true,
			AuditEvent:       "pfsense.guided_action.renew_wan",
		},
		{
			ID:               "pfsense.open-firewall-log",
			Label:            "Open firewall log",
			Description:      "Future guided action that deep-links a non-expert to the relevant pfSense firewall log view.",
			Risk:             "low",
			Enabled:          false,
			RequiresApproval: false,
			AuditEvent:       "pfsense.guided_action.open_firewall_log",
		},
	}
}

func newSNMPClient(cfg Config) (SNMPClient, func() error, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	client := &gosnmp.GoSNMP{
		Target:    cfg.Host,
		Port:      cfg.Port,
		Community: cfg.Community,
		Version:   gosnmp.Version2c,
		Timeout:   timeout,
		Retries:   1,
	}
	if err := client.Connect(); err != nil {
		return nil, func() error { return nil }, err
	}
	return client, client.Conn.Close, nil
}

type interfaceState struct {
	Index int
	Name  string
	Up    bool
}

func readInterfaces(client SNMPClient) ([]interfaceState, error) {
	names, err := client.WalkAll(".1.3.6.1.2.1.2.2.1.2")
	if err != nil {
		return nil, err
	}
	states, err := client.WalkAll(".1.3.6.1.2.1.2.2.1.8")
	if err != nil {
		return nil, err
	}

	byIndex := make(map[int]*interfaceState)
	for _, pdu := range names {
		index := oidIndex(pdu.Name)
		state := byIndex[index]
		if state == nil {
			state = &interfaceState{Index: index}
			byIndex[index] = state
		}
		state.Name = pduString(pdu)
	}
	for _, pdu := range states {
		index := oidIndex(pdu.Name)
		state := byIndex[index]
		if state == nil {
			state = &interfaceState{Index: index}
			byIndex[index] = state
		}
		state.Up = pduInt(pdu) == 1
	}

	interfaces := make([]interfaceState, 0, len(byIndex))
	for _, state := range byIndex {
		interfaces = append(interfaces, *state)
	}
	return interfaces, nil
}

type systemInfo struct {
	Description   string
	Name          string
	UptimeSeconds int64
}

func parseSystem(packet *gosnmp.SnmpPacket) systemInfo {
	var info systemInfo
	if packet == nil {
		return info
	}
	for _, variable := range packet.Variables {
		switch variable.Name {
		case ".1.3.6.1.2.1.1.1.0":
			info.Description = pduString(variable)
		case ".1.3.6.1.2.1.1.3.0":
			info.UptimeSeconds = int64(pduInt(variable)) / 100
		case ".1.3.6.1.2.1.1.5.0":
			info.Name = pduString(variable)
		}
	}
	return info
}

type interfaceSummary struct {
	Up      int
	Down    int
	Reasons []string
}

func summarizeInterfaces(interfaces []interfaceState, expected []string) interfaceSummary {
	summary := interfaceSummary{}
	byName := make(map[string]interfaceState, len(interfaces))
	for _, item := range interfaces {
		if item.Up {
			summary.Up++
		} else {
			summary.Down++
		}
		byName[strings.ToLower(item.Name)] = item
	}

	for _, expectedName := range expected {
		item, exists := byName[strings.ToLower(expectedName)]
		if !exists {
			summary.Reasons = append(summary.Reasons, fmt.Sprintf("Expected pfSense interface %q was not reported by SNMP.", expectedName))
			continue
		}
		if !item.Up {
			summary.Reasons = append(summary.Reasons, fmt.Sprintf("Expected pfSense interface %q is down.", expectedName))
		}
	}
	return summary
}

func recommendedActions(reasons []string) []string {
	if len(reasons) == 0 {
		return []string{"No operator action is needed right now."}
	}
	return []string{"Open pfSense to inspect the affected gateway or interface before making changes."}
}

func highlights(system systemInfo, summary interfaceSummary, expected []string) []string {
	name := system.Name
	if name == "" {
		name = "pfSense"
	}
	items := []string{
		fmt.Sprintf("%s is reachable by read-only SNMP.", name),
		fmt.Sprintf("%d interfaces are up and %d are down.", summary.Up, summary.Down),
	}
	if len(expected) == 0 {
		items = append(items, "No expected-up interface list is configured, so down interfaces are informational.")
	}
	return items
}

func authoritativeLinks(cfg Config) []Link {
	if cfg.ConsoleURL == "" {
		return nil
	}
	return []Link{{Label: "Open pfSense", Href: cfg.ConsoleURL}}
}

func endpoint(cfg Config) string {
	return fmt.Sprintf("snmp://%s:%d", cfg.Host, cfg.Port)
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}

func oidIndex(oid string) int {
	parts := strings.Split(strings.Trim(oid, "."), ".")
	if len(parts) == 0 {
		return 0
	}
	parsed, _ := strconv.Atoi(parts[len(parts)-1])
	return parsed
}

func pduString(pdu gosnmp.SnmpPDU) string {
	switch value := pdu.Value.(type) {
	case string:
		return value
	case []byte:
		return string(value)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func pduInt(pdu gosnmp.SnmpPDU) int {
	switch value := pdu.Value.(type) {
	case int:
		return value
	case uint:
		return int(value)
	case uint32:
		return int(value)
	case uint64:
		return int(value)
	case int64:
		return int(value)
	default:
		parsed, _ := strconv.Atoi(fmt.Sprintf("%v", value))
		return parsed
	}
}

func formatDuration(seconds int64) string {
	if seconds <= 0 {
		return "unknown"
	}
	duration := time.Duration(seconds) * time.Second
	days := int(duration.Hours()) / 24
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(duration.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", int(duration.Minutes()))
}

func stateForReasons(reasons []string) string {
	if len(reasons) == 0 {
		return "ok"
	}
	return "warning"
}
