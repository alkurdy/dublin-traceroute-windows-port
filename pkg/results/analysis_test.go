/* SPDX-License-Identifier: BSD-2-Clause */

package results

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAnalyzeNetwork_LoadBalancing(t *testing.T) {
	tr := createSampleResult()
	analysis := tr.AnalyzeNetwork()

	if analysis == nil {
		t.Fatal("AnalyzeNetwork() returned nil")
	}

	if !analysis.HasLoadBalancing {
		t.Error("Expected HasLoadBalancing to be true (TTL 2 has two distinct IPs)")
	}

	if len(analysis.LoadBalancingHops) != 1 || analysis.LoadBalancingHops[0] != 2 {
		t.Errorf("Expected LoadBalancingHops to be [2], got %v", analysis.LoadBalancingHops)
	}

	if analysis.UniqueRouters != 4 {
		t.Errorf("Expected 4 unique routers, got %d", analysis.UniqueRouters)
	}

	if analysis.PacketLossRate != 0.0 {
		t.Errorf("Expected 0.0%% packet loss, got %.2f%%", analysis.PacketLossRate)
	}
}

func TestAnalyzeNetwork_PacketLoss(t *testing.T) {
	tr := &TracerouteResult{
		Target: "8.8.8.8",
		Hops:   make(map[uint8]*HopResult),
	}

	// 1 successful probe, 1 timeout probe
	tr.Hops[1] = &HopResult{
		TTL: 1,
		Flows: map[uint16]*FlowResult{
			0: {
				FlowID:     0,
				RTT:        15 * time.Millisecond,
				ResponseIP: "192.168.1.1",
			},
			1: {
				FlowID: 1,
				Error:  "timeout",
			},
		},
	}

	analysis := tr.AnalyzeNetwork()
	if analysis.PacketLossRate != 50.0 {
		t.Errorf("Expected 50.0%% packet loss, got %.2f%%", analysis.PacketLossRate)
	}
}

func TestAnalyzeNetwork_HighLatencyJump(t *testing.T) {
	tr := &TracerouteResult{
		Target: "8.8.8.8",
		Hops:   make(map[uint8]*HopResult),
	}

	// Hop 1: 10ms
	tr.Hops[1] = &HopResult{
		TTL: 1,
		Flows: map[uint16]*FlowResult{
			0: {RTT: 10 * time.Millisecond, ResponseIP: "192.168.1.1"},
		},
	}

	// Hop 2: 180ms (+170ms jump > 100ms threshold)
	tr.Hops[2] = &HopResult{
		TTL: 2,
		Flows: map[uint16]*FlowResult{
			0: {RTT: 180 * time.Millisecond, ResponseIP: "10.0.0.1"},
		},
	}

	analysis := tr.AnalyzeNetwork()
	if len(analysis.HighLatencyHops) == 0 {
		t.Fatal("Expected high latency jump to be detected at Hop 2")
	}

	issue := analysis.HighLatencyHops[0]
	if issue.TTL != 2 {
		t.Errorf("Expected issue at TTL 2, got %d", issue.TTL)
	}
	if issue.LatencyJump != 170*time.Millisecond {
		t.Errorf("Expected latency jump of 170ms, got %v", issue.LatencyJump)
	}
}

func TestAnalyzeNetwork_RealRecordedTrace(t *testing.T) {
	tracePath := filepath.Join("..", "..", "test_trace.json")
	data, err := os.ReadFile(tracePath)
	if err != nil {
		t.Skipf("Skipping real trace test: %s not found", tracePath)
		return
	}

	var tr TracerouteResult
	if err := json.Unmarshal(data, &tr); err != nil {
		t.Fatalf("Failed to parse real recorded trace JSON: %v", err)
	}

	if tr.Target == "" {
		t.Fatal("Parsed trace has empty target")
	}

	analysis := tr.AnalyzeNetwork()
	if analysis == nil {
		t.Fatal("AnalyzeNetwork() failed on real trace")
	}

	if analysis.UniqueRouters == 0 {
		t.Error("Expected positive number of unique routers in real trace")
	}

	// Verify paths extract cleanly without panic
	paths := tr.GetPaths()
	if len(paths) == 0 {
		t.Error("Expected paths to be extracted from real trace")
	}
}
