package main

import (
	"os"
	"testing"
	"time"
)

// Fast CI gate — pure latency model, no loopback, no sockets.
func TestQCPBeatsKCPSimulated(t *testing.T) {
	cfg := Config{
		Duration:    2 * time.Second,
		Connections: 10,
		PayloadSize: 256,
	}
	for _, name := range allProfileNames() {
		profile := profileByName(name)
		kcp := benchKCPWithProfile(cfg, profile)
		qcp := benchQCPSimulated(cfg, profile)
		if qcp.Latency.P50 >= kcp.Latency.P50 {
			t.Errorf("[%s] P50: QCP %v >= KCP %v", name, qcp.Latency.P50, kcp.Latency.P50)
		}
		if qcp.Latency.P99 >= kcp.Latency.P99 {
			t.Errorf("[%s] P99: QCP %v >= KCP %v", name, qcp.Latency.P99, kcp.Latency.P99)
		}
		t.Logf("[%s] KCP P50=%v QCP P50=%v", name, kcp.Latency.P50, qcp.Latency.P50)
	}
}

// Real UDP over Docker bridge or LAN — skipped unless QCP_NET_VERIFY=1.
func TestQCPBeatsKCPNet(t *testing.T) {
	if os.Getenv("QCP_NET_VERIFY") == "" {
		t.Skip("set QCP_NET_VERIFY=1 and QCP_BENCH_SERVER=HOST:9000 (non-loopback)")
	}
	server := os.Getenv("QCP_BENCH_SERVER")
	if server == "" {
		t.Fatal("QCP_BENCH_SERVER required")
	}
	if isLoopbackHost(hostFromServer(server)) {
		t.Fatal("refusing loopback; use Docker 10.10.0.10 or LAN IP")
	}

	cfg := Config{
		Server:      server,
		Scenario:    os.Getenv("QCP_BENCH_SCENARIO"),
		Duration:    5 * time.Second,
		Connections: 10,
		PayloadSize: 256,
	}
	if cfg.Scenario == "" {
		cfg.Scenario = "lan"
	}

	kcp := benchKCPNet(cfg)
	qcp := benchQCPLibNet(cfg)
	if qcp.PacketsSent == 0 {
		t.Fatal("QCP received no packets on real network")
	}
	if qcp.Latency.P50 >= kcp.Latency.P50 {
		t.Errorf("P50: QCP %v >= KCP %v", qcp.Latency.P50, kcp.Latency.P50)
	}
	if qcp.Latency.P99 >= kcp.Latency.P99 {
		t.Errorf("P99: QCP %v >= KCP %v", qcp.Latency.P99, kcp.Latency.P99)
	}
	t.Logf("scenario=%s KCP P50=%v QCP P50=%v pkts=%d/%d",
		cfg.Scenario, kcp.Latency.P50, qcp.Latency.P50, qcp.PacketsSent, kcp.PacketsSent)
}
