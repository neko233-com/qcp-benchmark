package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderComparisonMarkdownOmitsUDP(t *testing.T) {
	md := renderComparisonMarkdown([]ComparisonSection{{
		Title:    "场景：lan",
		Subtitle: "test",
		Results: []Result{
			{Protocol: "udp", Latency: LatencyStats{P50: 10 * time.Millisecond}},
			{Protocol: "tcp", Latency: LatencyStats{P50: 20 * time.Millisecond}, Throughput: 1.2, Bandwidth: 80, Connections: 10},
			{Protocol: "kcp", Latency: LatencyStats{P50: 30 * time.Millisecond}, Throughput: 1.1, Bandwidth: 76, Connections: 10},
			{Protocol: "qcp", Latency: LatencyStats{P50: 5 * time.Millisecond}, Throughput: 1.8, Bandwidth: 42, Connections: 10},
		},
	}})

	if strings.Contains(md, "| UDP |") {
		t.Fatalf("markdown should omit UDP rows: %s", md)
	}
	if !strings.Contains(md, "| QCP |") || !strings.Contains(md, "| TCP |") || !strings.Contains(md, "| KCP |") {
		t.Fatalf("markdown missing expected protocol rows: %s", md)
	}
}
