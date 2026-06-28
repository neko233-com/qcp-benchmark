package main

import (
	"testing"
	"time"
)

func TestQCPBeatsKCPAllScenarios(t *testing.T) {
	cfg := Config{
		Duration:    2 * time.Second,
		Connections: 10,
		PayloadSize: 256,
	}
	for _, name := range allProfileNames() {
		profile := profileByName(name)
		kcp := benchKCPWithProfile(cfg, profile)
		qcp := benchQCPLib(cfg, profile)
		if qcp.Latency.P50 >= kcp.Latency.P50 {
			t.Errorf("[%s] P50: QCP %v >= KCP %v", name, qcp.Latency.P50, kcp.Latency.P50)
		}
		if qcp.Latency.P99 >= kcp.Latency.P99 {
			t.Errorf("[%s] P99: QCP %v >= KCP %v", name, qcp.Latency.P99, kcp.Latency.P99)
		}
		t.Logf("[%s] KCP P50=%v QCP P50=%v (%.0f%% faster)",
			name, kcp.Latency.P50, qcp.Latency.P50,
			float64(kcp.Latency.P50-qcp.Latency.P50)/float64(kcp.Latency.P50)*100)
	}
}
