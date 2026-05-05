package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
)

type fakeSNMPClient struct{}

func (f fakeSNMPClient) Get(_ []string) (*gosnmp.SnmpPacket, error) {
	return &gosnmp.SnmpPacket{
		Variables: []gosnmp.SnmpPDU{
			{Name: ".1.3.6.1.2.1.1.1.0", Value: "pfSense test firewall"},
			{Name: ".1.3.6.1.2.1.1.3.0", Value: uint32(360000)},
			{Name: ".1.3.6.1.2.1.1.5.0", Value: "edge-fw"},
		},
	}, nil
}

func (f fakeSNMPClient) WalkAll(rootOid string) ([]gosnmp.SnmpPDU, error) {
	switch rootOid {
	case ".1.3.6.1.2.1.2.2.1.2":
		return []gosnmp.SnmpPDU{
			{Name: ".1.3.6.1.2.1.2.2.1.2.1", Value: "wan"},
			{Name: ".1.3.6.1.2.1.2.2.1.2.2", Value: "lan"},
		}, nil
	case ".1.3.6.1.2.1.2.2.1.8":
		return []gosnmp.SnmpPDU{
			{Name: ".1.3.6.1.2.1.2.2.1.8.1", Value: 1},
			{Name: ".1.3.6.1.2.1.2.2.1.8.2", Value: 2},
		}, nil
	default:
		return nil, nil
	}
}

func TestProbeWithClientSummarizesPFSenseSNMP(t *testing.T) {
	factory := func(Config) (SNMPClient, func() error, error) {
		return fakeSNMPClient{}, func() error { return nil }, nil
	}

	snapshot := ProbeWithClient(context.Background(), factory, Config{
		Host:                 "pfsense.local",
		Port:                 161,
		Community:            "readonly",
		ExpectedUpInterfaces: []string{"wan", "lan"},
		Timeout:              time.Second,
	})

	if !snapshot.Configured || !snapshot.Reachable {
		t.Fatalf("expected reachable configured snapshot, got %#v", snapshot)
	}
	if snapshot.Severity != "warning" {
		t.Fatalf("expected warning for down expected interface, got %q", snapshot.Severity)
	}

	metrics := snapshot.Metrics.(map[string]any)
	if metrics["systemName"] != "edge-fw" {
		t.Fatalf("expected system name edge-fw, got %#v", metrics["systemName"])
	}
	if metrics["downInterfaceCount"] != 1 {
		t.Fatalf("expected one down interface, got %#v", metrics["downInterfaceCount"])
	}
}

func TestProbeWithClientReportsUnconfiguredPFSense(t *testing.T) {
	snapshot := ProbeWithClient(context.Background(), nil, Config{})

	if snapshot.Configured {
		t.Fatalf("expected unconfigured snapshot")
	}
	if snapshot.Severity != "unknown" {
		t.Fatalf("expected unknown severity, got %q", snapshot.Severity)
	}
}
