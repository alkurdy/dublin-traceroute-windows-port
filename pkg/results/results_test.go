/* SPDX-License-Identifier: BSD-2-Clause */

package results

import (
	"encoding/json"
	"testing"
	"time"
)

func createSampleResult() *TracerouteResult {
	now := time.Now()
	tr := &TracerouteResult{
		Target:    "1.1.1.1",
		SrcIP:     "192.168.1.100",
		StartTime: now,
		EndTime:   now.Add(2 * time.Second),
		Duration:  2 * time.Second,
		Hops:      make(map[uint8]*HopResult),
	}

	// Hop 1: Gateway (single router, two flows)
	tr.Hops[1] = &HopResult{
		TTL: 1,
		Flows: map[uint16]*FlowResult{
			0: {
				FlowID:     0,
				SrcPort:    33434,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(10 * time.Millisecond),
				RTT:        10 * time.Millisecond,
				ResponseIP: "192.168.1.1",
			},
			1: {
				FlowID:     1,
				SrcPort:    33435,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(12 * time.Millisecond),
				RTT:        12 * time.Millisecond,
				ResponseIP: "192.168.1.1",
			},
		},
	}

	// Hop 2: ISP Load Balancer (Flow 0 hits Router A, Flow 1 hits Router B)
	tr.Hops[2] = &HopResult{
		TTL: 2,
		Flows: map[uint16]*FlowResult{
			0: {
				FlowID:     0,
				SrcPort:    33434,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(25 * time.Millisecond),
				RTT:        25 * time.Millisecond,
				ResponseIP: "10.0.0.1",
			},
			1: {
				FlowID:     1,
				SrcPort:    33435,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(28 * time.Millisecond),
				RTT:        28 * time.Millisecond,
				ResponseIP: "10.0.0.2",
			},
		},
	}

	// Hop 3: Destination Target
	tr.Hops[3] = &HopResult{
		TTL: 3,
		Flows: map[uint16]*FlowResult{
			0: {
				FlowID:     0,
				SrcPort:    33434,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(40 * time.Millisecond),
				RTT:        40 * time.Millisecond,
				ResponseIP: "1.1.1.1",
			},
			1: {
				FlowID:     1,
				SrcPort:    33435,
				DstPort:    33434,
				SentTime:   now,
				RecvTime:   now.Add(42 * time.Millisecond),
				RTT:        42 * time.Millisecond,
				ResponseIP: "1.1.1.1",
			},
		},
	}

	return tr
}

func TestTracerouteResult_ToJSON(t *testing.T) {
	tr := createSampleResult()
	jsonStr, err := tr.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() returned error: %v", err)
	}

	if jsonStr == "" {
		t.Fatal("ToJSON() returned empty string")
	}

	// Verify unmarshalling preserves target and hops
	var decoded TracerouteResult
	if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
		t.Fatalf("Failed to unmarshal generated JSON: %v", err)
	}

	if decoded.Target != tr.Target {
		t.Errorf("Expected target %s, got %s", tr.Target, decoded.Target)
	}
	if len(decoded.Hops) != len(tr.Hops) {
		t.Errorf("Expected %d hops, got %d", len(tr.Hops), len(decoded.Hops))
	}
}

func TestTracerouteResult_GetPaths(t *testing.T) {
	tr := createSampleResult()
	paths := tr.GetPaths()

	if len(paths) != 2 {
		t.Fatalf("Expected 2 paths for 2 flows, got %d", len(paths))
	}

	for _, p := range paths {
		if len(p.Hops) != 3 {
			t.Errorf("Expected path %d to have 3 hops, got %d", p.PathID, len(p.Hops))
		}
		// Confirm path terminates at target
		lastHop := p.Hops[len(p.Hops)-1]
		if lastHop.IP != "1.1.1.1" {
			t.Errorf("Path %d did not terminate at target: got %s", p.PathID, lastHop.IP)
		}
	}
}

// TestTracerouteResult_GetUniquePaths_Deduplication verifies that multiple flows
// traversing identical router hops are deduplicated into a single unique path.
func TestTracerouteResult_GetUniquePaths_Deduplication(t *testing.T) {
	tr := &TracerouteResult{
		Target:   "1.1.1.1",
		SrcIP:    "192.168.12.210",
		NumPaths: 2,
		Hops:     make(map[uint8]*HopResult),
	}

	// Hop 1: Gateway - identical for Flow 0 and Flow 1
	tr.Hops[1] = &HopResult{
		TTL: 1,
		Flows: map[uint16]*FlowResult{
			0: {FlowID: 0, ResponseIP: "192.168.12.1", RTT: 3 * time.Millisecond},
			1: {FlowID: 1, ResponseIP: "192.168.12.1", RTT: 4 * time.Millisecond},
		},
	}

	// Hop 2: ISP Router - identical for Flow 0 and Flow 1
	tr.Hops[2] = &HopResult{
		TTL: 2,
		Flows: map[uint16]*FlowResult{
			0: {FlowID: 0, ResponseIP: "100.36.56.1", RTT: 9 * time.Millisecond},
			1: {FlowID: 1, ResponseIP: "100.36.56.1", RTT: 8 * time.Millisecond},
		},
	}

	// All paths should be 2 flows
	allPaths := tr.GetAllPaths()
	if len(allPaths) != 2 {
		t.Fatalf("Expected 2 raw flow paths, got %d", len(allPaths))
	}

	// Unique paths must deduplicate to 1 path
	uniquePaths := tr.GetUniquePaths()
	if len(uniquePaths) != 1 {
		t.Fatalf("Expected 1 deduplicated unique path, got %d", len(uniquePaths))
	}

	if len(uniquePaths[0].FlowIDs) != 2 {
		t.Errorf("Expected path to associate with both flows [0, 1], got %v", uniquePaths[0].FlowIDs)
	}

	if tr.HasMultiplePaths() {
		t.Error("Expected HasMultiplePaths to be false for identical flows")
	}

	// Check RTT averaging: (9ms + 8ms) / 2 = 8.5ms
	hop2RTT := uniquePaths[0].Hops[1].RTT
	if hop2RTT != 8500*time.Microsecond {
		t.Errorf("Expected averaged RTT of 8.5ms, got %v", hop2RTT)
	}
}

// TestTracerouteResult_AllTimeouts_NoFalseMultipath verifies that when all probes timeout,
// HasMultiplePaths returns false and PrintSummary does not claim multiple paths.
func TestTracerouteResult_AllTimeouts_NoFalseMultipath(t *testing.T) {
	tr := &TracerouteResult{
		Target:   "8.8.8.8",
		SrcIP:    "192.168.1.100",
		NumPaths: 4,
		Hops:     make(map[uint8]*HopResult),
	}

	// All 4 flows time out at Hop 1
	tr.Hops[1] = &HopResult{
		TTL: 1,
		Flows: map[uint16]*FlowResult{
			0: {FlowID: 0, Error: "timeout"},
			1: {FlowID: 1, Error: "timeout"},
			2: {FlowID: 2, Error: "timeout"},
			3: {FlowID: 3, Error: "timeout"},
		},
	}

	if tr.HasMultiplePaths() {
		t.Error("Expected HasMultiplePaths to be false when all probes timed out")
	}

	// PrintSummary should run without error or claiming load balancing
	tr.PrintSummary(false)
}

func TestTracerouteResult_GetHopCount(t *testing.T) {
	tr := createSampleResult()
	count := tr.GetHopCount()
	if count != 3 {
		t.Errorf("Expected hop count 3, got %d", count)
	}
}

func TestTracerouteResult_GetUniqueHosts(t *testing.T) {
	tr := createSampleResult()
	hosts := tr.GetUniqueHosts()

	// 192.168.1.1, 10.0.0.1, 10.0.0.2, 1.1.1.1 -> 4 unique hosts
	if len(hosts) != 4 {
		t.Errorf("Expected 4 unique hosts, got %d: %v", len(hosts), hosts)
	}
}

func TestTracerouteResult_CalculateHopStatistics(t *testing.T) {
	tr := createSampleResult()
	stats := tr.CalculateHopStatistics()

	if len(stats) != 3 {
		t.Fatalf("Expected 3 hop stats, got %d", len(stats))
	}

	hop1, ok := stats[1]
	if !ok {
		t.Fatal("Missing stats for hop 1")
	}

	if hop1.Sent != 2 || hop1.Received != 2 {
		t.Errorf("Expected 2 sent and 2 received at hop 1, got %d/%d", hop1.Sent, hop1.Received)
	}
	if hop1.LossPercent != 0.0 {
		t.Errorf("Expected 0.0%% loss at hop 1, got %.2f%%", hop1.LossPercent)
	}
	if hop1.MinRTT != 10*time.Millisecond {
		t.Errorf("Expected MinRTT 10ms, got %v", hop1.MinRTT)
	}
	if hop1.MaxRTT != 12*time.Millisecond {
		t.Errorf("Expected MaxRTT 12ms, got %v", hop1.MaxRTT)
	}
	if hop1.AvgRTT != 11*time.Millisecond {
		t.Errorf("Expected AvgRTT 11ms, got %v", hop1.AvgRTT)
	}
}

// TestTracerouteResult_PrintSummary_NoPanic tests that PrintSummary (default)
// and PrintMTRStyle execute properly without panic on valid trace results.
func TestTracerouteResult_PrintSummary_NoPanic(t *testing.T) {
	tr := createSampleResult()
	tr.PrintSummary()
	tr.PrintMTRStyle()
}

// TestTracerouteResult_PrintSummary_ShowAnalysisOption tests that PrintSummary
// respects the showAnalysis flag, confirming both true and false execute cleanly.
func TestTracerouteResult_PrintSummary_ShowAnalysisOption(t *testing.T) {
	tr := createSampleResult()
	// Test with analysis explicitly disabled (-analyze=false)
	tr.PrintSummary(false)
	// Test with analysis explicitly enabled (-analyze=true)
	tr.PrintSummary(true)
}

