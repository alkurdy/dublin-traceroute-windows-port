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

// UDPProbe represents a UDP-based traceroute probe for Windows
type UDPProbe struct {
	Target     net.IP
	SrcIP      net.IP
	SrcPort    uint16
	DstPort    uint16
	NumPaths   uint16
	MinTTL     uint8
	MaxTTL     uint8
	ProbeCount int  // Number of probes per hop for MTR-style statistics
	Delay      time.Duration
	Timeout    time.Duration
	socket     int
	capture    *capture.WindowsCapture
}

// NewUDPProbe creates a new UDP probe instance
func NewUDPProbe(target string, srcPort, dstPort, numPaths uint16, minTTL, maxTTL uint8, probeCount int) (*UDPProbe, error) {
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
			return nil, fmt.Errorf("no IP addresses found for target %s", target)
		}
		// Use first IPv4 address
		for _, ip := range ips {
			if ip.To4() != nil {
				targetIP = ip
				break
			}
		}
		if targetIP == nil {
			return nil, fmt.Errorf("no IPv4 address found for target %s", target)
		}
	} else {
		targetIP = targetIP.To4()
	}

	if targetIP == nil {
		return nil, fmt.Errorf("invalid IPv4 target: %s", target)
	}

	// Get local source IP
	srcIP, err := platform.GetLocalIPv4Address()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	// Create raw UDP socket
	sock, err := platform.CreateUDPSocket()
	if err != nil {
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}

	// Create packet capture for ICMP responses
	cap, err := capture.NewCapture("", 3*time.Second)
	if err != nil {
		platform.CloseSocket(sock)
		return nil, fmt.Errorf("failed to create packet capture: %w", err)
	}

	// BPF filter disabled - Npcap on Windows seems to have issues with "icmp" filter
	// We'll filter ICMP in software which is slightly less efficient but works reliably

	return &UDPProbe{
		Target:     targetIP,
		SrcIP:      net.ParseIP(srcIP),
		SrcPort:    srcPort,
		DstPort:    dstPort,
		NumPaths:   numPaths,
		MinTTL:     minTTL,
		MaxTTL:     maxTTL,
		ProbeCount: probeCount,
		Delay:      time.Millisecond * 10,
		Timeout:    time.Second * 3,
		socket:     sock,
		capture:    cap,
	}, nil
}

// craftUDPPacket creates a raw UDP/IP packet with specified TTL and flow ID
func (p *UDPProbe) craftUDPPacket(ttl uint8, flowID uint16) ([]byte, error) {
	// Create IP layer
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TOS:      0,
		Length:   0, // Will be set automatically
		Id:       flowID,
		Flags:    layers.IPv4DontFragment,
		FragOffset: 0,
		TTL:      ttl,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    p.SrcIP,
		DstIP:    p.Target,
	}

	// Create UDP layer with flowID encoded in source port
	srcPort := p.SrcPort + flowID
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(p.DstPort),
	}

	// Set checksum computation
	udp.SetNetworkLayerForChecksum(ip)

	// Serialize packet
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	// Add 8-byte payload (Dublin Traceroute signature)
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}

	err := gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to serialize packet: %w", err)
	}

	return buf.Bytes(), nil
}

// sendProbe sends a single probe packet
func (p *UDPProbe) sendProbe(ttl uint8, flowID uint16) error {
	packet, err := p.craftUDPPacket(ttl, flowID)
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
		return fmt.Errorf("failed to send probe (TTL=%d, FlowID=%d): %w", ttl, flowID, err)
	}

	return nil
}

// Traceroute executes the Dublin Traceroute algorithm
func (p *UDPProbe) Traceroute() (*results.TracerouteResult, error) {
	result := &results.TracerouteResult{
		Target:    p.Target.String(),
		SrcIP:     p.SrcIP.String(),
		StartTime: time.Now(),
		Hops:      make(map[uint8]*results.HopResult),
	}

	fmt.Printf("Dublin Traceroute to %s (%s)\n", p.Target, p.Target)
	fmt.Printf("Using UDP ports %d-%d, TTL %d-%d\n", p.SrcPort, p.SrcPort+p.NumPaths-1, p.MinTTL, p.MaxTTL)
	
	if p.ProbeCount > 1 {
		fmt.Printf("MTR mode: %d probes per hop for statistical analysis\n", p.ProbeCount)
	}
	
	fmt.Println()

	// For each TTL level
	for ttl := p.MinTTL; ttl <= p.MaxTTL; ttl++ {
		hopResult := &results.HopResult{
			TTL:   ttl,
			Flows: make(map[uint16]*results.FlowResult),
		}

		// Perform multiple probe rounds if ProbeCount > 1 (MTR mode)
		for round := 0; round < p.ProbeCount; round++ {
			// Send probes for each flow
			for flowID := uint16(0); flowID < p.NumPaths; flowID++ {
				// Calculate unique flow key for this round
				uniqueFlowID := flowID + uint16(round)*p.NumPaths
				
				flowResult := &results.FlowResult{
					FlowID:   uniqueFlowID,
					SrcPort:  p.SrcPort + flowID,
					DstPort:  p.DstPort,
					SentTime: time.Now(),
				}

				// Send probe
				err := p.sendProbe(ttl, flowID)
				if err != nil {
					if round == 0 {
						fmt.Printf("TTL=%2d Flow=%2d: Failed to send probe: %v\n", ttl, flowID, err)
					}
					flowResult.Error = err.Error()
					hopResult.Flows[uniqueFlowID] = flowResult
					continue
				}
				
				// Wait for response
				packet, srcIP, err := p.capture.CaptureICMPResponse(p.SrcIP, p.Target, 0)
				if err != nil {
					// Timeout or no response
					flowResult.Error = "timeout"
					hopResult.Flows[uniqueFlowID] = flowResult
					if round == 0 {
						fmt.Printf("TTL=%2d Flow=%2d: *\n", ttl, flowID)
					}
					continue
				}
				
				flowResult.RecvTime = time.Now()
				flowResult.RTT = flowResult.RecvTime.Sub(flowResult.SentTime)
				flowResult.ResponseIP = srcIP.String()

				// Parse ICMP response
				icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
				if icmpLayer != nil {
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					flowResult.ICMPType = uint8(icmp.TypeCode.Type())
					flowResult.ICMPCode = uint8(icmp.TypeCode.Code())
				}

				// Try to get hostname (only on first round to avoid delays)
				if round == 0 {
					names, err := net.LookupAddr(srcIP.String())
					if err == nil && len(names) > 0 {
						flowResult.Hostname = names[0]
					}
				}

				hopResult.Flows[uniqueFlowID] = flowResult

				// Print result (only first round in MTR mode for cleaner output)
				if round == 0 {
					fmt.Printf("TTL=%2d Flow=%2d: %s", ttl, flowID, srcIP)
					if flowResult.Hostname != "" {
						fmt.Printf(" (%s)", flowResult.Hostname)
					}
					fmt.Printf(" %v\n", flowResult.RTT)
				}

				// Small delay between probes
				time.Sleep(p.Delay)
			}
		}

		result.Hops[ttl] = hopResult

		// Check if we reached the destination
		reachedTarget := false
		for _, flow := range hopResult.Flows {
			if flow.ResponseIP == p.Target.String() {
				reachedTarget = true
				break
			}
		}

		if reachedTarget {
			fmt.Printf("\nReached target %s at TTL %d\n", p.Target, ttl)
			break
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// Close cleans up resources
func (p *UDPProbe) Close() {
	if p.capture != nil {
		p.capture.Close()
	}
	if p.socket != 0 {
		platform.CloseSocket(p.socket)
	}
}

// SetDelay sets the delay between probe packets
func (p *UDPProbe) SetDelay(delay time.Duration) {
	p.Delay = delay
}

// SetTimeout sets the timeout for waiting for responses
func (p *UDPProbe) SetTimeout(timeout time.Duration) {
	p.Timeout = timeout
	// Recreate capture with new timeout
	if p.capture != nil {
		iface := p.capture.GetInterface()
		p.capture.Close()
		cap, err := capture.NewCapture(iface, timeout)
		if err == nil {
			cap.SetBPFFilter("icmp")
			p.capture = cap
		}
	}
}

// GetStats returns probe statistics
func (p *UDPProbe) GetStats() (received, dropped, ifDropped uint, err error) {
	if p.capture == nil {
		return 0, 0, 0, fmt.Errorf("capture not initialized")
	}
	return p.capture.GetStats()
}
