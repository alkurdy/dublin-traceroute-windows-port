// +build windows

/* SPDX-License-Identifier: BSD-2-Clause */

//
// This file is based on code from the original dublin-traceroute project:
//   https://github.com/insomniacslk/dublin-traceroute
// Copyright (c) insomniacslk (https://github.com/insomniacslk)

package probe

import (
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/sys/windows"

	"github.com/atlanticbb/dublin-traceroute-windows/internal/platform"
	"github.com/atlanticbb/dublin-traceroute-windows/pkg/capture"
	"github.com/atlanticbb/dublin-traceroute-windows/pkg/results"
)

// TCPProbe represents a TCP-based traceroute probe for Windows
// Uses TCP SYN packets instead of UDP for better firewall traversal
type TCPProbe struct {
	Target     net.IP
	SrcIP      net.IP
	SrcPort    uint16
	DstPort    uint16  // Target port (80, 443, etc.)
	NumPaths   uint16
	MinTTL     uint8
	MaxTTL     uint8
	ProbeCount int  // Number of probes per hop for MTR-style statistics
	Delay      time.Duration
	Timeout    time.Duration
	socket     int
	capture    *capture.WindowsCapture
}

// NewTCPProbe creates a new TCP probe instance
func NewTCPProbe(target string, srcPort, dstPort, numPaths uint16, minTTL, maxTTL uint8, probeCount int) (*TCPProbe, error) {
	// Verify admin privileges
	if err := platform.RequireAdmin(); err != nil {
		return nil, err
	}

	// Resolve target IP
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		ips, err := net.LookupIP(target)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target %s: %w", target, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses found for %s", target)
		}
		// Prefer IPv4
		for _, ip := range ips {
			if ip.To4() != nil {
				targetIP = ip
				break
			}
		}
		if targetIP == nil {
			targetIP = ips[0]
		}
	}

	// Only support IPv4 for now
	if targetIP.To4() == nil {
		return nil, fmt.Errorf("IPv6 not supported yet")
	}

	// Get local IP
	srcIPStr, err := platform.GetLocalIPv4Address()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}
	srcIP := net.ParseIP(srcIPStr)

	// Create raw socket
	sock, err := platform.CreateUDPSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create raw socket: %w", err)
	}

	probe := &TCPProbe{
		Target:     targetIP,
		SrcIP:      srcIP,
		SrcPort:    srcPort,
		DstPort:    dstPort,
		NumPaths:   numPaths,
		MinTTL:     minTTL,
		MaxTTL:     maxTTL,
		ProbeCount: probeCount,
		Delay:      10 * time.Millisecond,
		Timeout:    1 * time.Second,  // Shorter timeout for TCP (more responsive)
		socket:     sock,
	}

	return probe, nil
}

// SetTimeout sets the probe timeout
func (p *TCPProbe) SetTimeout(timeout time.Duration) {
	p.Timeout = timeout
}

// SetDelay sets the delay between probes
func (p *TCPProbe) SetDelay(delay time.Duration) {
	p.Delay = delay
}

// Close cleans up resources
func (p *TCPProbe) Close() error {
	if p.capture != nil {
		p.capture.Close()
	}
	if p.socket != 0 {
		return platform.CloseSocket(p.socket)
	}
	return nil
}

// craftTCPPacket creates a TCP SYN packet with specified parameters
func (p *TCPProbe) craftTCPPacket(ttl uint8, flowID uint16) ([]byte, error) {
	// Create IP layer
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TOS:      0,
		Id:       uint16(time.Now().Unix() & 0xFFFF),
		Flags:    layers.IPv4DontFragment,
		TTL:      ttl,
		Protocol: layers.IPProtocolTCP,
		SrcIP:    p.SrcIP,
		DstIP:    p.Target,
	}

	// Encode flow ID in source port (Dublin Traceroute style)
	srcPort := p.SrcPort + flowID

	// Create TCP layer (SYN packet)
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(p.DstPort),
		Seq:     uint32(time.Now().Unix()),
		SYN:     true,
		Window:  65535,
	}

	// Set network layer for checksum calculation
	tcp.SetNetworkLayerForChecksum(ip)

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	if err := gopacket.SerializeLayers(buf, opts, ip, tcp); err != nil {
		return nil, fmt.Errorf("failed to serialize TCP packet: %w", err)
	}

	return buf.Bytes(), nil
}

// sendProbe sends a single TCP SYN probe
func (p *TCPProbe) sendProbe(ttl uint8, flowID uint16) error {
	packet, err := p.craftTCPPacket(ttl, flowID)
	if err != nil {
		return err
	}

	// Create destination sockaddr
	dest := &windows.SockaddrInet4{
		Port: int(p.DstPort),
	}
	copy(dest.Addr[:], p.Target.To4())

	// Send packet
	err = platform.SendPacket(p.socket, packet, dest)
	if err != nil {
		return fmt.Errorf("failed to send TCP probe: %w", err)
	}

	return nil
}

// Traceroute performs TCP-based multipath traceroute
func (p *TCPProbe) Traceroute() (*results.TracerouteResult, error) {
	// Initialize packet capture
	cap, err := capture.NewCapture("", 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize capture: %w", err)
	}
	p.capture = cap
	defer cap.Close()

	result := &results.TracerouteResult{
		Target:    p.Target.String(),
		SrcIP:     p.SrcIP.String(),
		StartTime: time.Now(),
		Hops:      make(map[uint8]*results.HopResult),
	}

	fmt.Printf("\nDublin Traceroute (TCP) to %s (%s)\n", p.Target, p.Target)
	fmt.Printf("Using TCP SYN to port %d, source ports %d-%d, TTL %d-%d\n", 
		p.DstPort, p.SrcPort, p.SrcPort+p.NumPaths-1, p.MinTTL, p.MaxTTL)
	fmt.Printf("Timeout per probe: %v\n", p.Timeout)
	
	if p.ProbeCount > 1 {
		fmt.Printf("MTR mode: %d probes per hop for statistical analysis\n", p.ProbeCount)
	}
	
	fmt.Println()

	reachedTarget := false

	// Send probes for each TTL
	for ttl := p.MinTTL; ttl <= p.MaxTTL && !reachedTarget; ttl++ {
		hopResult := &results.HopResult{
			TTL:   ttl,
			Flows: make(map[uint16]*results.FlowResult),
		}

		// Perform multiple probe rounds if ProbeCount > 1 (MTR mode)
		for round := 0; round < p.ProbeCount; round++ {
			// Send probe for each flow
			for flowID := uint16(0); flowID < p.NumPaths; flowID++ {
				// Calculate unique flow key for this round
				uniqueFlowID := flowID + uint16(round)*p.NumPaths
				srcPort := p.SrcPort + flowID
				
				flowResult := &results.FlowResult{
					FlowID:   uniqueFlowID,
					SrcPort:  srcPort,
					DstPort:  p.DstPort,
					SentTime: time.Now(),
				}

				// Send probe
				sendErr := p.sendProbe(ttl, flowID)
				if sendErr != nil {
					if round == 0 {
						fmt.Printf("TTL=%2d Flow=%2d: Send failed: %v\n", ttl, flowID, sendErr)
					}
					flowResult.Error = sendErr.Error()
					hopResult.Flows[uniqueFlowID] = flowResult
					continue
				}

				// Wait for ICMP response
				packet, srcIP, err := cap.CaptureICMPResponse(p.SrcIP, p.Target, 0)
				
				if err == nil && packet != nil && srcIP != nil {
					flowResult.RecvTime = time.Now()
					flowResult.RTT = flowResult.RecvTime.Sub(flowResult.SentTime)
					flowResult.ResponseIP = srcIP.String()
					
					// Only lookup hostname on first round to avoid delays
					if round == 0 {
						flowResult.Hostname = p.lookupHostname(srcIP)
					}
					
					hopResult.Flows[uniqueFlowID] = flowResult

					// Print result (only first round in MTR mode for cleaner output)
					if round == 0 {
						fmt.Printf("TTL=%2d Flow=%2d: %s (%s) %.4fms\n",
							ttl, flowID, flowResult.ResponseIP, flowResult.Hostname, flowResult.RTT.Seconds()*1000)
					}

					// Check if we reached the target
					if srcIP.Equal(p.Target) {
						reachedTarget = true
					}
				} else {
					// Timeout or no response
					flowResult.Error = "timeout"
					hopResult.Flows[uniqueFlowID] = flowResult
					if round == 0 {
						fmt.Printf("TTL=%2d Flow=%2d: *\n", ttl, flowID)
					}
				}

				time.Sleep(p.Delay)
			}
		}

		if len(hopResult.Flows) > 0 {
			result.Hops[ttl] = hopResult
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if reachedTarget {
		fmt.Printf("\nReached target %s\n", p.Target)
	}

	return result, nil
}

// captureTCPResponse attempts to capture TCP SYN-ACK or RST from target
func (p *TCPProbe) captureTCPResponse(srcPort uint16, timeout time.Duration) (net.IP, time.Duration) {
	// This would require capturing TCP packets in addition to ICMP
	// For now, return nil - can be enhanced later
	// Full implementation would listen for TCP packets with:
	// - Source: target IP, port p.DstPort
	// - Dest: source IP, port srcPort
	// - Flags: SYN-ACK or RST
	return nil, 0
}

// lookupHostname performs reverse DNS lookup
func (p *TCPProbe) lookupHostname(ip net.IP) string {
	names, err := net.LookupAddr(ip.String())
	if err != nil || len(names) == 0 {
		return ""
	}
	return names[0]
}
