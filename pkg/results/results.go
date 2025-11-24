/* SPDX-License-Identifier: BSD-2-Clause */

//
// This file is based on code from the original dublin-traceroute project:
//   https://github.com/insomniacslk/dublin-traceroute
// Copyright (c) insomniacslk (https://github.com/insomniacslk)

package results

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// TracerouteResult represents the complete result of a Dublin Traceroute
type TracerouteResult struct {
	Target    string               `json:"target"`
	SrcIP     string               `json:"src_ip"`
	StartTime time.Time            `json:"start_time"`
	EndTime   time.Time            `json:"end_time"`
	Duration  time.Duration        `json:"duration"`
	Hops      map[uint8]*HopResult `json:"hops"`
}

// HopResult represents all flows at a specific TTL level
type HopResult struct {
	TTL   uint8                  `json:"ttl"`
	Flows map[uint16]*FlowResult `json:"flows"`
}

// FlowResult represents a single probe flow result
type FlowResult struct {
	FlowID     uint16        `json:"flow_id"`
	SrcPort    uint16        `json:"src_port"`
	DstPort    uint16        `json:"dst_port"`
	SentTime   time.Time     `json:"sent_time"`
	RecvTime   time.Time     `json:"recv_time"`
	RTT        time.Duration `json:"rtt"`
	ResponseIP string        `json:"response_ip"`
	Hostname   string        `json:"hostname,omitempty"`
	ICMPType   uint8         `json:"icmp_type,omitempty"`
	ICMPCode   uint8         `json:"icmp_code,omitempty"`
	Error      string        `json:"error,omitempty"`
}

// Path represents a unique path through the network
type Path struct {
	PathID int       `json:"path_id"`
	Hops   []PathHop `json:"hops"`
}

// PathHop represents a single hop in a path
type PathHop struct {
	TTL      uint8         `json:"ttl"`
	IP       string        `json:"ip"`
	Hostname string        `json:"hostname,omitempty"`
	RTT      time.Duration `json:"rtt"`
}

// NetworkAnalysis provides insights about the network path
type NetworkAnalysis struct {
	HasLoadBalancing  bool           `json:"has_load_balancing"`
	LoadBalancingHops []uint8        `json:"load_balancing_hops,omitempty"`
	PacketLossRate    float64        `json:"packet_loss_rate"`
	AverageRTT        time.Duration  `json:"average_rtt"`
	MinRTT            time.Duration  `json:"min_rtt"`
	MaxRTT            time.Duration  `json:"max_rtt"`
	HighLatencyHops   []LatencyIssue `json:"high_latency_hops,omitempty"`
	AsymmetricRouting bool           `json:"asymmetric_routing_detected"`
	UniqueRouters     int            `json:"unique_routers"`
	Recommendations   []string       `json:"recommendations,omitempty"`
}

// LatencyIssue identifies hops with unusual latency
type LatencyIssue struct {
	TTL           uint8         `json:"ttl"`
	IP            string        `json:"ip"`
	Hostname      string        `json:"hostname,omitempty"`
	Latency       time.Duration `json:"latency"`
	LatencyJump   time.Duration `json:"latency_jump"`
	PossibleCause string        `json:"possible_cause"`
}

// HopStatistics tracks statistics for continuous probing (MTR-style)
type HopStatistics struct {
	TTL         uint8           `json:"ttl"`
	IP          string          `json:"ip"`
	Hostname    string          `json:"hostname,omitempty"`
	Sent        int             `json:"sent"`
	Received    int             `json:"received"`
	LossPercent float64         `json:"loss_percent"`
	RTTs        []time.Duration `json:"-"` // Raw RTT values for calculation
	MinRTT      time.Duration   `json:"min_rtt"`
	AvgRTT      time.Duration   `json:"avg_rtt"`
	MaxRTT      time.Duration   `json:"max_rtt"`
	StdDevRTT   time.Duration   `json:"stddev_rtt"`
	BestTime    time.Time       `json:"best_time"`
	WorstTime   time.Time       `json:"worst_time"`
}

// ToJSON converts the result to JSON format
func (tr *TracerouteResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal to JSON: %w", err)
	}
	return string(data), nil
}

// GetPaths extracts unique paths from the traceroute results
func (tr *TracerouteResult) GetPaths() []Path {
	paths := make([]Path, 0)

	// Get all flow IDs from the first hop
	var flowIDs []uint16
	if firstHop, ok := tr.Hops[1]; ok {
		for flowID := range firstHop.Flows {
			flowIDs = append(flowIDs, flowID)
		}
	}

	// Build a path for each flow
	for pathID, flowID := range flowIDs {
		path := Path{
			PathID: pathID,
			Hops:   make([]PathHop, 0),
		}

		// Collect all hops for this flow
		for ttl := uint8(1); ttl <= 255; ttl++ {
			hopResult, ok := tr.Hops[ttl]
			if !ok {
				break
			}

			flowResult, ok := hopResult.Flows[flowID]
			if !ok || flowResult.Error != "" {
				continue
			}

			pathHop := PathHop{
				TTL:      ttl,
				IP:       flowResult.ResponseIP,
				Hostname: flowResult.Hostname,
				RTT:      flowResult.RTT,
			}

			path.Hops = append(path.Hops, pathHop)

			// Stop if we reached the target
			if flowResult.ResponseIP == tr.Target {
				break
			}
		}

		paths = append(paths, path)
	}

	return paths
}

// PrintSummary prints a human-readable summary of the results
func (tr *TracerouteResult) PrintSummary() {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("Dublin Traceroute Results\n")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Target:   %s\n", tr.Target)
	fmt.Printf("Source:   %s\n", tr.SrcIP)
	fmt.Printf("Duration: %v\n", tr.Duration)
	fmt.Println()

	// Get analysis
	analysis := tr.AnalyzeNetwork()

	paths := tr.GetPaths()
	fmt.Printf("Discovered %d unique path(s):\n", len(paths))
	if len(paths) > 1 {
		fmt.Printf("  ℹ️  Multiple paths detected - your traffic is load-balanced across %d routes\n", len(paths))
	}
	fmt.Println()

	for _, path := range paths {
		fmt.Printf("Path %d (%d hops):\n", path.PathID, len(path.Hops))
		for _, hop := range path.Hops {
			hostname := hop.IP
			if hop.Hostname != "" {
				hostname = fmt.Sprintf("%s (%s)", hop.Hostname, hop.IP)
			}
			fmt.Printf("  %2d: %-50s %8v\n", hop.TTL, hostname, hop.RTT)
		}
		fmt.Println()
	}

	// Print statistics
	totalProbes := 0
	successfulProbes := 0
	timeouts := 0

	for _, hopResult := range tr.Hops {
		for _, flowResult := range hopResult.Flows {
			totalProbes++
			if flowResult.Error == "" {
				successfulProbes++
			} else if flowResult.Error == "timeout" {
				timeouts++
			}
		}
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("Statistics:\n")
	fmt.Printf("  Total Probes:      %d\n", totalProbes)
	fmt.Printf("  Successful:        %d (%.1f%%)\n", successfulProbes, float64(successfulProbes)/float64(totalProbes)*100)
	fmt.Printf("  Timeouts:          %d (%.1f%%)\n", timeouts, float64(timeouts)/float64(totalProbes)*100)
	fmt.Printf("  Packet Loss:       %.1f%%\n", analysis.PacketLossRate)
	if successfulProbes > 0 {
		fmt.Printf("  Average Latency:   %v\n", analysis.AverageRTT)
		fmt.Printf("  Min/Max Latency:   %v / %v\n", analysis.MinRTT, analysis.MaxRTT)
	}
	fmt.Printf("  Unique Routers:    %d\n", analysis.UniqueRouters)

	// Print network analysis
	tr.PrintNetworkAnalysis(analysis)

	fmt.Println(strings.Repeat("=", 80))
}

// GetHopCount returns the number of hops to reach the target
func (tr *TracerouteResult) GetHopCount() int {
	maxTTL := uint8(0)
	for ttl := range tr.Hops {
		if ttl > maxTTL {
			maxTTL = ttl
		}
	}
	return int(maxTTL)
}

// GetUniqueHosts returns a list of unique IP addresses seen in the trace
func (tr *TracerouteResult) GetUniqueHosts() []string {
	seen := make(map[string]bool)
	hosts := make([]string, 0)

	for _, hopResult := range tr.Hops {
		for _, flowResult := range hopResult.Flows {
			if flowResult.ResponseIP != "" && !seen[flowResult.ResponseIP] {
				seen[flowResult.ResponseIP] = true
				hosts = append(hosts, flowResult.ResponseIP)
			}
		}
	}

	return hosts
}

// HasMultiplePaths checks if multiple paths were detected
func (tr *TracerouteResult) HasMultiplePaths() bool {
	paths := tr.GetPaths()

	// Check if any two paths differ
	if len(paths) <= 1 {
		return false
	}

	for i := 0; i < len(paths)-1; i++ {
		for j := i + 1; j < len(paths); j++ {
			if !pathsEqual(paths[i], paths[j]) {
				return true
			}
		}
	}

	return false
}

// pathsEqual compares two paths for equality
func pathsEqual(p1, p2 Path) bool {
	if len(p1.Hops) != len(p2.Hops) {
		return false
	}

	for i := range p1.Hops {
		if p1.Hops[i].IP != p2.Hops[i].IP {
			return false
		}
	}

	return true
}

// GetAverageRTT calculates the average RTT for a specific TTL
func (tr *TracerouteResult) GetAverageRTT(ttl uint8) time.Duration {
	hopResult, ok := tr.Hops[ttl]
	if !ok {
		return 0
	}

	total := time.Duration(0)
	count := 0

	for _, flowResult := range hopResult.Flows {
		if flowResult.Error == "" && flowResult.RTT > 0 {
			total += flowResult.RTT
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return total / time.Duration(count)
}

// SaveToFile saves the results to a JSON file
func (tr *TracerouteResult) SaveToFile(filename string) error {
	jsonData, err := tr.ToJSON()
	if err != nil {
		return err
	}

	// Note: Would need to import io/ioutil or os for actual file writing
	// For now, return the JSON string
	_ = jsonData
	return fmt.Errorf("file saving not yet implemented")
}

// CalculateHopStatistics computes per-hop statistics from multiple probe rounds
func (tr *TracerouteResult) CalculateHopStatistics() map[uint8]*HopStatistics {
	stats := make(map[uint8]*HopStatistics)

	for ttl, hopResult := range tr.Hops {
		stat := &HopStatistics{
			TTL:  ttl,
			RTTs: make([]time.Duration, 0),
		}

		// Aggregate data from all flows at this hop
		ipMap := make(map[string]int) // Track which IP responded most
		for _, flowResult := range hopResult.Flows {
			stat.Sent++

			if flowResult.Error == "" && flowResult.ResponseIP != "" {
				stat.Received++

				// Track most common responding IP
				ipMap[flowResult.ResponseIP]++

				// Collect RTT data
				if flowResult.RTT > 0 {
					stat.RTTs = append(stat.RTTs, flowResult.RTT)

					// Track best/worst times
					if stat.BestTime.IsZero() || flowResult.RTT < stat.MinRTT || stat.MinRTT == 0 {
						stat.MinRTT = flowResult.RTT
						stat.BestTime = flowResult.RecvTime
					}
					if flowResult.RTT > stat.MaxRTT {
						stat.MaxRTT = flowResult.RTT
						stat.WorstTime = flowResult.RecvTime
					}
				}
			}
		}

		// Use most common IP as the hop IP
		maxCount := 0
		for ip, count := range ipMap {
			if count > maxCount {
				stat.IP = ip
				maxCount = count
			}
		}

		// Get hostname if available
		if stat.IP != "" {
			for _, flowResult := range hopResult.Flows {
				if flowResult.ResponseIP == stat.IP && flowResult.Hostname != "" {
					stat.Hostname = flowResult.Hostname
					break
				}
			}
		}

		// Calculate loss percentage
		if stat.Sent > 0 {
			stat.LossPercent = float64(stat.Sent-stat.Received) / float64(stat.Sent) * 100.0
		}

		// Calculate average RTT
		if len(stat.RTTs) > 0 {
			total := time.Duration(0)
			for _, rtt := range stat.RTTs {
				total += rtt
			}
			stat.AvgRTT = total / time.Duration(len(stat.RTTs))

			// Calculate standard deviation
			if len(stat.RTTs) > 1 {
				variance := float64(0)
				avgFloat := float64(stat.AvgRTT.Microseconds())
				for _, rtt := range stat.RTTs {
					diff := float64(rtt.Microseconds()) - avgFloat
					variance += diff * diff
				}
				variance /= float64(len(stat.RTTs))
				stat.StdDevRTT = time.Duration(math.Sqrt(variance)) * time.Microsecond
			}
		}

		stats[ttl] = stat
	}

	return stats
}

// PrintMTRStyle displays results in MTR-style table format with statistics
func (tr *TracerouteResult) PrintMTRStyle() {
	stats := tr.CalculateHopStatistics()

	// Get sorted TTL list
	ttls := make([]uint8, 0, len(stats))
	for ttl := range stats {
		ttls = append(ttls, ttl)
	}

	// Simple bubble sort
	for i := 0; i < len(ttls); i++ {
		for j := i + 1; j < len(ttls); j++ {
			if ttls[i] > ttls[j] {
				ttls[i], ttls[j] = ttls[j], ttls[i]
			}
		}
	}

	fmt.Println("\n=== MTR-Style Statistics ===")
	fmt.Printf("Target: %s (%s)\n", tr.Target, tr.SrcIP)
	fmt.Printf("Duration: %v\n\n", tr.Duration.Round(time.Millisecond))

	// Header
	fmt.Printf("%-3s %-40s %6s %6s %8s %8s %8s %8s\n",
		"TTL", "Host", "Loss%", "Snt", "Min", "Avg", "Max", "StdDev")
	fmt.Println(strings.Repeat("-", 110))

	// Rows
	for _, ttl := range ttls {
		stat := stats[ttl]

		// Format hostname/IP
		host := stat.IP
		if host == "" {
			host = "???"
		}
		if stat.Hostname != "" {
			host = fmt.Sprintf("%s (%s)", stat.Hostname, stat.IP)
		}
		if len(host) > 40 {
			host = host[:37] + "..."
		}

		// Format RTT values
		minStr := "---"
		avgStr := "---"
		maxStr := "---"
		stdStr := "---"

		if stat.MinRTT > 0 {
			minStr = fmt.Sprintf("%.1fms", float64(stat.MinRTT.Microseconds())/1000.0)
		}
		if stat.AvgRTT > 0 {
			avgStr = fmt.Sprintf("%.1fms", float64(stat.AvgRTT.Microseconds())/1000.0)
		}
		if stat.MaxRTT > 0 {
			maxStr = fmt.Sprintf("%.1fms", float64(stat.MaxRTT.Microseconds())/1000.0)
		}
		if stat.StdDevRTT > 0 {
			stdStr = fmt.Sprintf("%.1fms", float64(stat.StdDevRTT.Microseconds())/1000.0)
		}

		fmt.Printf("%-3d %-40s %5.1f%% %6d %8s %8s %8s %8s\n",
			ttl, host, stat.LossPercent, stat.Sent,
			minStr, avgStr, maxStr, stdStr)
	}

	fmt.Println()

	// Identify problem areas
	problemHops := make([]string, 0)
	for _, ttl := range ttls {
		stat := stats[ttl]
		if stat.LossPercent >= 50.0 && stat.Sent > 0 {
			problemHops = append(problemHops, fmt.Sprintf("TTL %d: %.1f%% loss", ttl, stat.LossPercent))
		}
	}

	if len(problemHops) > 0 {
		fmt.Println("\n⚠ High Packet Loss Detected:")
		for _, issue := range problemHops {
			fmt.Printf("  • %s\n", issue)
		}
	}

	// Check for asymmetric routing (100% loss could indicate return path issues)
	allLoss := true
	for _, stat := range stats {
		if stat.LossPercent < 100.0 {
			allLoss = false
			break
		}
	}

	if allLoss && len(stats) > 0 {
		fmt.Println("\n⚠ Complete Packet Loss - Possible Causes:")
		fmt.Println("  • Firewall blocking ICMP responses")
		fmt.Println("  • Return path routing failure")
		fmt.Println("  • Target host unreachable")
		fmt.Println("  • Network filtering on return path")
	}

	// Check for increasing loss toward target (suggests forward path issues)
	if len(ttls) >= 3 {
		lastThreeAvgLoss := 0.0
		count := 0
		for i := len(ttls) - 3; i < len(ttls); i++ {
			if ttls[i] < 255 {
				lastThreeAvgLoss += stats[ttls[i]].LossPercent
				count++
			}
		}
		if count > 0 {
			lastThreeAvgLoss /= float64(count)
			if lastThreeAvgLoss > 20.0 {
				fmt.Println("\n📊 Loss increases toward target - likely forward path congestion/filtering")
			}
		}
	}

	// Check for high jitter (large StdDev)
	for _, ttl := range ttls {
		stat := stats[ttl]
		if stat.AvgRTT > 0 && stat.StdDevRTT > stat.AvgRTT/2 {
			fmt.Printf("\n⚠ High jitter at TTL %d (%s): Avg=%.1fms, StdDev=%.1fms\n",
				ttl, stat.IP,
				float64(stat.AvgRTT.Microseconds())/1000.0,
				float64(stat.StdDevRTT.Microseconds())/1000.0)
			fmt.Println("  • Suggests congestion, queuing, or load balancing")
		}
	}

	fmt.Println()
}
