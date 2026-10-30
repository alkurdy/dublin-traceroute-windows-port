/* SPDX-License-Identifier: BSD-2-Clause */

//
// This file is based on code from the original dublin-traceroute project:
//   https://github.com/insomniacslk/dublin-traceroute
// Copyright (c) insomniacslk (https://github.com/insomniacslk)

package results

import (
	"fmt"
	"strings"
	"time"
)

// AnalyzeNetwork performs comprehensive analysis of the traceroute results
func (tr *TracerouteResult) AnalyzeNetwork() *NetworkAnalysis {
	analysis := &NetworkAnalysis{
		LoadBalancingHops: make([]uint8, 0),
		HighLatencyHops:   make([]LatencyIssue, 0),
		Recommendations:   make([]string, 0),
	}

	// Calculate packet loss and RTT statistics
	totalProbes := 0
	successfulProbes := 0
	var rtts []time.Duration
	uniqueRouters := make(map[string]bool)

	for _, hopResult := range tr.Hops {
		for _, flowResult := range hopResult.Flows {
			totalProbes++
			if flowResult.Error == "" && flowResult.RTT > 0 {
				successfulProbes++
				rtts = append(rtts, flowResult.RTT)
			}
			if flowResult.ResponseIP != "" {
				uniqueRouters[flowResult.ResponseIP] = true
			}
		}
	}

	analysis.UniqueRouters = len(uniqueRouters)

	if totalProbes > 0 {
		analysis.PacketLossRate = float64(totalProbes-successfulProbes) / float64(totalProbes) * 100
	}

	// RTT statistics
	if len(rtts) > 0 {
		analysis.MinRTT = rtts[0]
		analysis.MaxRTT = rtts[0]
		var total time.Duration
		for _, rtt := range rtts {
			total += rtt
			if rtt < analysis.MinRTT {
				analysis.MinRTT = rtt
			}
			if rtt > analysis.MaxRTT {
				analysis.MaxRTT = rtt
			}
		}
		analysis.AverageRTT = total / time.Duration(len(rtts))
	}

	// Detect load balancing (multiple IPs at same TTL)
	for ttl, hopResult := range tr.Hops {
		ips := make(map[string]bool)
		for _, flowResult := range hopResult.Flows {
			if flowResult.ResponseIP != "" && flowResult.Error == "" {
				ips[flowResult.ResponseIP] = true
			}
		}
		if len(ips) > 1 {
			analysis.HasLoadBalancing = true
			analysis.LoadBalancingHops = append(analysis.LoadBalancingHops, ttl)
		}
	}

	// Detect high latency hops
	prevRTT := time.Duration(0)
	for ttl := uint8(1); ttl != 0; ttl++ {
		hopResult, ok := tr.Hops[ttl]
		if !ok {
			break
		}

		avgHopRTT := tr.GetAverageRTT(ttl)
		if avgHopRTT == 0 {
			continue
		}

		// Check for significant latency jump (> 100ms increase or > 3x previous)
		if prevRTT > 0 {
			jump := avgHopRTT - prevRTT
			if jump > 100*time.Millisecond || (prevRTT > 0 && avgHopRTT > prevRTT*3) {
				// Find representative hop for this TTL
				for _, flowResult := range hopResult.Flows {
					if flowResult.Error == "" && flowResult.ResponseIP != "" {
						cause := "Long-distance link or congestion"
						if jump > 500*time.Millisecond {
							cause = "Likely intercontinental or satellite link"
						} else if avgHopRTT > 200*time.Millisecond {
							cause = "High latency link - possible congestion or routing inefficiency"
						}

						analysis.HighLatencyHops = append(analysis.HighLatencyHops, LatencyIssue{
							TTL:           ttl,
							IP:            flowResult.ResponseIP,
							Hostname:      flowResult.Hostname,
							Latency:       avgHopRTT,
							LatencyJump:   jump,
							PossibleCause: cause,
						})
						break
					}
				}
			}
		}

		if avgHopRTT > 0 {
			prevRTT = avgHopRTT
		}
	}

	// Detect asymmetric routing (common with load balancing)
	paths := tr.GetPaths()
	if len(paths) > 1 {
		// Check if paths have different intermediate hops
		different := false
		for i := 0; i < len(paths)-1; i++ {
			for j := i + 1; j < len(paths); j++ {
				if !pathsEqual(paths[i], paths[j]) {
					different = true
					break
				}
			}
		}
		analysis.AsymmetricRouting = different
	}

	// Generate recommendations
	if analysis.PacketLossRate > 20 {
		analysis.Recommendations = append(analysis.Recommendations,
			"High packet loss detected (>20%) - some routers may not respond to UDP probes, or there's network congestion")
	}

	if analysis.HasLoadBalancing {
		analysis.Recommendations = append(analysis.Recommendations,
			"Load balancing detected - your traffic takes multiple paths, which can improve reliability and performance")
	}

	if len(analysis.HighLatencyHops) > 0 {
		analysis.Recommendations = append(analysis.Recommendations,
			fmt.Sprintf("Found %d high-latency hop(s) - review LatencyIssue details below", len(analysis.HighLatencyHops)))
	}

	if analysis.AverageRTT > 200*time.Millisecond {
		analysis.Recommendations = append(analysis.Recommendations,
			"High average latency detected - target may be geographically distant or network path is suboptimal")
	}

	return analysis
}

// PrintNetworkAnalysis prints detailed network analysis for end users
func (tr *TracerouteResult) PrintNetworkAnalysis(analysis *NetworkAnalysis) {
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("📊 NETWORK ANALYSIS")
	fmt.Println(strings.Repeat("─", 80))

	// Explain what this tool shows
	fmt.Println("\n💡 What This Shows:")
	fmt.Println("   This traceroute reveals the FORWARD PATH from your computer to the target.")
	fmt.Println("   The return path (target → you) may be different due to asymmetric routing.")
	fmt.Println("   Each 'hop' is a router that forwards your packets toward the destination.")
	fmt.Println()

	// Load balancing analysis
	if analysis.HasLoadBalancing {
		fmt.Println("🔀 Load Balancing Detected:")
		fmt.Printf("   Your traffic is distributed across multiple network paths.\n")
		fmt.Printf("   This is NORMAL and GOOD - it improves reliability and performance.\n")
		fmt.Printf("   Load balancing occurs at hop(s): %v\n", analysis.LoadBalancingHops)
		fmt.Println()
	} else {
		fmt.Println("🛣️  Single Path Routing:")
		fmt.Println("   Your traffic follows a single path - no load balancing detected.")
		fmt.Println()
	}

	// Latency analysis
	if len(analysis.HighLatencyHops) > 0 {
		fmt.Println("⚠️  High Latency Hops:")
		for _, issue := range analysis.HighLatencyHops {
			hostname := issue.IP
			if issue.Hostname != "" {
				hostname = fmt.Sprintf("%s (%s)", issue.Hostname, issue.IP)
			}
			fmt.Printf("   Hop %d: %s\n", issue.TTL, hostname)
			fmt.Printf("   └─ Latency: %v (jump: +%v)\n", issue.Latency, issue.LatencyJump)
			fmt.Printf("   └─ %s\n", issue.PossibleCause)
			fmt.Println()
		}
	}

	// Packet loss interpretation
	if analysis.PacketLossRate > 50 {
		fmt.Println("📉 High Packet Loss:")
		fmt.Printf("   %.1f%% of probes timed out. This is often NORMAL because:\n", analysis.PacketLossRate)
		fmt.Println("   • Many routers deprioritize or ignore UDP traceroute packets")
		fmt.Println("   • Rate limiting protects routers from being overwhelmed")
		fmt.Println("   • This usually doesn't affect your actual data traffic")
		fmt.Println()
	} else if analysis.PacketLossRate > 20 {
		fmt.Printf("📊 Moderate Packet Loss: %.1f%% - Some routers not responding\n\n", analysis.PacketLossRate)
	}

	// Geographic insights
	tr.PrintGeographicInsights()

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		fmt.Println("💡 Insights:")
		for _, rec := range analysis.Recommendations {
			fmt.Printf("   • %s\n", rec)
		}
		fmt.Println()
	}

	// Asymmetric routing explanation
	if analysis.AsymmetricRouting {
		fmt.Println("🔄 Asymmetric Routing Detected:")
		fmt.Println("   Different flows take different paths through the network.")
		fmt.Println("   This is common on the Internet and indicates load balancing.")
		fmt.Println()
	}

	// Parameter-specific guidance and diagnostics
	tr.PrintParameterInsights()

	fmt.Println(strings.Repeat("─", 80))
}

// PrintParameterInsights displays contextual guidance and diagnostic interpretation
// tailored to the specific parameters used during the trace.
func (tr *TracerouteResult) PrintParameterInsights() {
	fmt.Println("⚙️  Parameter-Specific Guidance & Diagnostics:")

	// 1. Protocol and Port Guidance
	proto := tr.Protocol
	if proto == "" {
		proto = "UDP"
	}

	if proto == "TCP" {
		dstPort := tr.DstPort
		if dstPort == 0 {
			dstPort = 80
		}
		fmt.Printf("   • Protocol (TCP SYN to port %d):\n", dstPort)
		fmt.Printf("     - Stateful Traversal: TCP SYN on web ports (80/443) traverses corporate & stateful firewalls that block UDP.\n")
		fmt.Printf("     - Target Completion: Open target ports return TCP SYN-ACK; closed ports return TCP RST.\n")
		fmt.Printf("     - Local NAT Note: If Hop 1 or 2 times out (*), your local NAT router may silently drop outbound\n")
		fmt.Printf("       low-TTL TCP packets without returning ICMP Time Exceeded (Type 11).\n")
	} else {
		dstPort := tr.DstPort
		if dstPort == 0 {
			dstPort = 33434
		}
		fmt.Printf("   • Protocol (UDP to port %d):\n", dstPort)
		fmt.Printf("     - Standard Probing: Intermediate hops decrement TTL to 0 and emit ICMP Time Exceeded (Type 11).\n")
		fmt.Printf("     - Target Completion: Destination host responds with ICMP Port Unreachable (Type 3, Code 3).\n")
		if dstPort == 53 {
			fmt.Printf("     - DNS Port 53: Target DNS servers often silently drop non-DNS UDP payloads without returning\n")
			fmt.Printf("       Port Unreachable. Timeouts (*) at the final target hop on port 53 are expected.\n")
		} else {
			fmt.Printf("     - High UDP Ports (%d+): Timeouts (*) at intermediate hops are common because transit routers\n", dstPort)
			fmt.Printf("       rate-limit ICMP generation on their control planes, or intermediate firewalls drop high UDP.\n")
		}
	}

	// 2. Multipath / Load Balancing Guidance
	npaths := tr.NumPaths
	if npaths <= 1 {
		fmt.Println("   • Multipath Discovery (-npaths 1):")
		fmt.Println("     - Probed with 1 flow. To uncover Equal-Cost Multi-Path (ECMP) routing and divergent paths,")
		fmt.Println("       run with '-npaths 4' or '-npaths 8'.")
	} else {
		srcPort := tr.SrcPort
		if srcPort == 0 {
			srcPort = 33434
		}
		fmt.Printf("   • Multipath ECMP Discovery (-npaths %d, source ports %d-%d):\n", npaths, srcPort, srcPort+npaths-1)
		fmt.Println("     - Flow Hashing: Dublin Traceroute modifies source ports to traverse distinct ECMP hash buckets.")
		if tr.HasMultiplePaths() {
			fmt.Printf("     - Divergent Paths: Detected %d distinct routes. Compare per-path latencies to identify asymmetric routing.\n", len(tr.GetPaths()))
		} else {
			fmt.Println("     - Single Path Active: All flows traversed the same routers; upstream network uses single-path or per-dest hashing.")
		}
	}

	// 3. Multi-Probe / MTR Guidance
	if tr.ProbeCount > 1 {
		fmt.Printf("   • Multi-Probe / MTR Mode (-count %d):\n", tr.ProbeCount)
		fmt.Println("     - Rate-Limiting vs Real Loss: Loss at an isolated hop followed by 0% loss downstream indicates")
		fmt.Println("       router ICMP rate-limiting (benign). Loss that persists through to the destination indicates genuine congestion.")
		fmt.Println("     - Jitter (StdDev): High standard deviation reveals queue oscillation, bufferbloat, or route flapping.")
	} else {
		fmt.Println("   • Single Probe Mode (-count 1):")
		fmt.Println("     - Run with '-count 5' or '-count 10' for MTR-style statistics (loss %, min/avg/max latency, jitter).")
	}

	// 4. Timeout Guidance
	if tr.Timeout > 0 {
		if tr.Timeout < 500*time.Millisecond {
			fmt.Printf("   • Aggressive Timeout (-timeout %v):\n", tr.Timeout)
			fmt.Printf("     - Fast scanning. Note that transoceanic hops or congested WAN links (>%v RTT) may falsely timeout (*).\n", tr.Timeout)
		} else if tr.Timeout >= 2*time.Second {
			fmt.Printf("   • Conservative Timeout (-timeout %v):\n", tr.Timeout)
			fmt.Println("     - Thorough for high-latency or satellite links, but increases total wait time on unresponsive hops.")
		}
	}

	// 5. Target Reachability and Max TTL Guidance
	if tr.MaxTTL > 0 && !tr.IsTargetReached() {
		fmt.Printf("   • Destination Not Reached (-max-ttl %d):\n", tr.MaxTTL)
		fmt.Printf("     - Target %s was not reached within %d hops. Internet paths typically span 12-25 hops.\n", tr.Target, tr.MaxTTL)
		fmt.Println("       Try increasing search depth with '-max-ttl 30'.")
	}

	fmt.Println()
}

// PrintGeographicInsights provides geographic context based on hostnames
func (tr *TracerouteResult) PrintGeographicInsights() {
	locations := make(map[string]bool)

	for _, hopResult := range tr.Hops {
		for _, flowResult := range hopResult.Flows {
			if flowResult.Hostname == "" {
				continue
			}

			hostname := strings.ToLower(flowResult.Hostname)

			// Detect common location codes in hostnames
			locationHints := map[string]string{
				"nyc":  "New York",
				"bstn": "Boston",
				"lax":  "Los Angeles",
				"sfo":  "San Francisco",
				"chi":  "Chicago",
				"dfw":  "Dallas",
				"sea":  "Seattle",
				"mia":  "Miami",
				"atl":  "Atlanta",
				"lon":  "London",
				"fra":  "Frankfurt",
				"ams":  "Amsterdam",
				"tok":  "Tokyo",
				"sin":  "Singapore",
				"syd":  "Sydney",
			}

			for code, location := range locationHints {
				if strings.Contains(hostname, code) {
					locations[location] = true
				}
			}
		}
	}

	if len(locations) > 0 {
		fmt.Println("🌍 Geographic Path:")
		fmt.Print("   Your traffic appears to traverse: ")
		locs := make([]string, 0, len(locations))
		for loc := range locations {
			locs = append(locs, loc)
		}
		fmt.Println(strings.Join(locs, " → "))
		fmt.Println()
	}
}

// PrintComparisonHelp prints explanation of how to compare paths over time
func PrintComparisonHelp() {
	fmt.Println("\n💡 TIP: Comparing Routes Over Time")
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("To detect route changes or diagnose intermittent issues:")
	fmt.Println()
	fmt.Println("  1. Save results to JSON:")
	fmt.Println("     dublin-traceroute -target example.com -output-json baseline.json")
	fmt.Println()
	fmt.Println("  2. Run again during issues:")
	fmt.Println("     dublin-traceroute -target example.com -output-json problem.json")
	fmt.Println()
	fmt.Println("  3. Compare the JSON files to see:")
	fmt.Println("     • Route changes (different IPs at same hop)")
	fmt.Println("     • Latency increases")
	fmt.Println("     • New load balancing behavior")
	fmt.Println()
	fmt.Println("Route changes are normal, but sudden changes during problems can indicate:")
	fmt.Println("  • Network failover (a link went down)")
	fmt.Println("  • ISP routing policy changes")
	fmt.Println("  • BGP route flapping")
	fmt.Println(strings.Repeat("─", 80))
}

// ExplainReturnPath provides education about return path analysis
func ExplainReturnPath() string {
	return `
UNDERSTANDING RETURN PATHS
═══════════════════════════════════════════════════════════════════════════════

What Dublin Traceroute Shows:
  ✓ Forward path: YOUR COMPUTER → TARGET
  ✗ Return path:  TARGET → YOUR COMPUTER

Why Can't We See Return Path?
  Traceroute works by sending packets with increasing TTL (Time To Live) values.
  When a packet's TTL expires at a router, that router sends back an ICMP message.
  This tells us about routers on the FORWARD path only.

Is Return Path Different?
  YES - The Internet uses "asymmetric routing" where:
  • Forward and return paths can be completely different
  • Each direction is optimized independently
  • Return path depends on routing policies at the TARGET's network

How to Analyze Return Path:
  You would need to run traceroute FROM the target server TO your computer.
  This requires:
  1. Access to the target server (not possible for public IPs)
  2. Or ask the target's operator to run traceroute toward you
  3. Or use looking glass servers near the target

Common Scenarios:
  • Your packet to Google: goes through ISP → backbone → Google (shown here)
  • Google's reply: might use different backbone, different ISP entry point
  
  This is NORMAL and actually beneficial:
  • Each direction uses the fastest available path at that moment
  • Provides better reliability (if one path fails, other may still work)
  • Optimizes for different traffic engineering policies

Real-World Example:
  Forward:  You → ISP A → NTT → Cogent → Google
  Return:   Google → Level3 → ISP B → You
  
  Both work fine even though they're different!
═══════════════════════════════════════════════════════════════════════════════
`
}
