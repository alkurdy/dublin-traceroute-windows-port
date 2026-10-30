// +build windows

/* SPDX-License-Identifier: BSD-2-Clause */

//
// This file is based on code from the original dublin-traceroute project:
//   https://github.com/insomniacslk/dublin-traceroute
// Copyright (c) insomniacslk (https://github.com/insomniacslk)

package capture

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/alkurdy/dublin-traceroute-windows-port/internal/platform"
)

// WindowsCapture handles packet capture on Windows using Npcap
type WindowsCapture struct {
	handle       *pcap.Handle
	packetSource *gopacket.PacketSource
	packetChan   <-chan gopacket.Packet
	iface        string
	timeout      time.Duration
}

// Device represents a network device available for capture
type Device struct {
	Name        string
	Description string
	Addresses   []string
}

// ListDevices returns all available network devices
func ListDevices() ([]Device, error) {
	// Check Npcap is installed
	installed, err := platform.CheckNpcapInstalled()
	if err != nil {
		return nil, fmt.Errorf("failed to check Npcap installation: %w", err)
	}
	if !installed {
		return nil, fmt.Errorf("%s", platform.GetNpcapInstallMessage())
	}

	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate network devices: %w\n"+
			"Ensure Npcap is installed and you're running as administrator", err)
	}

	result := make([]Device, 0, len(devices))
	for _, dev := range devices {
		addrs := make([]string, 0, len(dev.Addresses))
		for _, addr := range dev.Addresses {
			if addr.IP != nil && addr.IP.To4() != nil {
				addrs = append(addrs, addr.IP.String())
			}
		}

		result = append(result, Device{
			Name:        dev.Name,
			Description: dev.Description,
			Addresses:   addrs,
		})
	}

	return result, nil
}

// FindDeviceByIP finds a network device that has the specified IP address
func FindDeviceByIP(targetIP string) (string, error) {
	devices, err := ListDevices()
	if err != nil {
		return "", err
	}

	for _, dev := range devices {
		for _, addr := range dev.Addresses {
			if addr == targetIP {
				return dev.Name, nil
			}
		}
	}

	return "", fmt.Errorf("no device found with IP address %s", targetIP)
}

// FindDefaultDevice finds the device associated with the default route
func FindDefaultDevice() (string, error) {
	localIP, err := platform.GetLocalIPv4Address()
	if err != nil {
		return "", fmt.Errorf("failed to get local IP: %w", err)
	}

	devName, err := FindDeviceByIP(localIP)
	if err != nil {
		return "", fmt.Errorf("failed to find device for IP %s: %w", localIP, err)
	}

	return devName, nil
}

// NewCapture creates a new packet capture instance
func NewCapture(device string, timeout time.Duration) (*WindowsCapture, error) {
	// Verify admin privileges
	if err := platform.RequireAdmin(); err != nil {
		return nil, err
	}

	// Check Npcap installation
	installed, err := platform.CheckNpcapInstalled()
	if err != nil {
		return nil, fmt.Errorf("failed to check Npcap: %w", err)
	}
	if !installed {
		return nil, fmt.Errorf("%s", platform.GetNpcapInstallMessage())
	}

	// If no device specified, find default
	if device == "" {
		device, err = FindDefaultDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to find default device: %w", err)
		}
	}

	// Open device for capture with 10ms driver buffer timeout for low-latency delivery
	handle, err := pcap.OpenLive(device, 65535, true, 10*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %w\n"+
			"Ensure Npcap is installed and running as administrator", device, err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetChan := packetSource.Packets()

	return &WindowsCapture{
		handle:       handle,
		packetSource: packetSource,
		packetChan:   packetChan,
		iface:        device,
		timeout:      timeout,
	}, nil
}

// SetBPFFilter applies a BPF filter to the capture
func (wc *WindowsCapture) SetBPFFilter(filter string) error {
	err := wc.handle.SetBPFFilter(filter)
	if err != nil {
		return fmt.Errorf("failed to set BPF filter '%s': %w", filter, err)
	}
	return nil
}

// CaptureICMPResponse captures ICMP responses matching the specified criteria
// Returns the response packet and the source IP
func (wc *WindowsCapture) CaptureICMPResponse(srcIP net.IP, dstIP net.IP, expectedType layers.ICMPv4TypeCode) (gopacket.Packet, net.IP, error) {
	deadline := time.Now().Add(wc.timeout)

	for {
		select {
		case packet, ok := <-wc.packetChan:
			if !ok || packet == nil {
				return nil, nil, fmt.Errorf("packet capture closed unexpectedly")
			}

			matched, routerIP := MatchICMPResponse(packet, srcIP, dstIP, expectedType)
			if matched {
				return packet, routerIP, nil
			}

		case <-time.After(time.Until(deadline)):
			return nil, nil, fmt.Errorf("timeout waiting for ICMP response")
		}

		// Check if we've exceeded the deadline
		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("timeout waiting for ICMP response")
		}
	}
}

// CaptureMultipleResponses captures multiple ICMP responses within the timeout period
func (wc *WindowsCapture) CaptureMultipleResponses(srcIP net.IP, dstIP net.IP, count int) ([]gopacket.Packet, []net.IP, error) {
	packets := make([]gopacket.Packet, 0, count)
	sources := make([]net.IP, 0, count)

	deadline := time.Now().Add(wc.timeout)

	for len(packets) < count {
		select {
		case packet, ok := <-wc.packetChan:
			if !ok || packet == nil {
				break
			}

			matched, routerIP := MatchICMPResponse(packet, srcIP, dstIP, 0)
			if matched {
				packets = append(packets, packet)
				sources = append(sources, routerIP)
			}

		case <-time.After(time.Until(deadline)):
			// Timeout - return what we have
			if len(packets) == 0 {
				return nil, nil, fmt.Errorf("timeout: no responses captured")
			}
			return packets, sources, nil
		}

		if time.Now().After(deadline) {
			break
		}
	}

	if len(packets) == 0 {
		return nil, nil, fmt.Errorf("no matching ICMP responses captured")
	}

	return packets, sources, nil
}

// Close closes the packet capture handle
func (wc *WindowsCapture) Close() {
	if wc.handle != nil {
		wc.handle.Close()
	}
}

// GetInterface returns the interface name being captured
func (wc *WindowsCapture) GetInterface() string {
	return wc.iface
}

// GetStats returns capture statistics
func (wc *WindowsCapture) GetStats() (received, dropped, ifDropped uint, err error) {
	stats, err := wc.handle.Stats()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get capture stats: %w", err)
	}
	return uint(stats.PacketsReceived), uint(stats.PacketsDropped), uint(stats.PacketsIfDropped), nil
}

// PrintDeviceList prints all available devices in a user-friendly format
func PrintDeviceList() error {
	devices, err := ListDevices()
	if err != nil {
		return err
	}

	fmt.Println("\nAvailable Network Devices:")
	fmt.Println(strings.Repeat("=", 80))

	for i, dev := range devices {
		fmt.Printf("\n[%d] %s\n", i+1, dev.Name)
		if dev.Description != "" {
			fmt.Printf("    Description: %s\n", dev.Description)
		}
		if len(dev.Addresses) > 0 {
			fmt.Printf("    IP Addresses: %s\n", strings.Join(dev.Addresses, ", "))
		}
	}

	fmt.Println()
	return nil
}

// MatchICMPResponse evaluates if an incoming ICMP packet is a valid response to our traceroute probe.
// For TimeExceeded (Type 11) and DestinationUnreachable (Type 3), it validates the embedded IP header
// to ensure the original packet was sent from srcIP to dstIP. For EchoReply (Type 0), it validates
// that the outer sender matches dstIP. This prevents capturing unrelated background ICMP traffic.
func MatchICMPResponse(packet gopacket.Packet, srcIP, dstIP net.IP, expectedType layers.ICMPv4TypeCode) (bool, net.IP) {
	icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
	if icmpLayer == nil {
		return false, nil
	}

	icmp, _ := icmpLayer.(*layers.ICMPv4)
	if expectedType != 0 && icmp.TypeCode != expectedType {
		return false, nil
	}

	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return false, nil
	}
	ip, _ := ipLayer.(*layers.IPv4)

	switch icmp.TypeCode.Type() {
	case layers.ICMPv4TypeTimeExceeded, layers.ICMPv4TypeDestinationUnreachable:
		// Time Exceeded and Destination Unreachable packets embed the original IP header + payload
		if len(icmp.Payload) < 20 {
			return false, nil
		}
		embeddedIP := &layers.IPv4{}
		err := embeddedIP.DecodeFromBytes(icmp.Payload, gopacket.NilDecodeFeedback)
		if err != nil {
			return false, nil
		}
		// Strictly verify embedded packet matches our probe destination and source
		if embeddedIP.SrcIP.Equal(srcIP) && embeddedIP.DstIP.Equal(dstIP) {
			return true, ip.SrcIP
		}

	case layers.ICMPv4TypeEchoReply:
		// Echo Reply comes directly from destination target to local source IP
		if ip.SrcIP.Equal(dstIP) && ip.DstIP.Equal(srcIP) {
			return true, ip.SrcIP
		}
	}

	return false, nil
}

// MatchTCPOrICMPResponse matches intermediate ICMP TimeExceeded/Unreachable OR destination TCP response (SYN-ACK / RST)
func MatchTCPOrICMPResponse(packet gopacket.Packet, srcIP, dstIP net.IP, srcPort, dstPort uint16) (bool, net.IP) {
	// First check for ICMP response from intermediate hop
	if matched, routerIP := MatchICMPResponse(packet, srcIP, dstIP, 0); matched {
		return true, routerIP
	}

	// Check for direct TCP response from destination target (SYN-ACK or RST)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return false, nil
	}
	ip, _ := ipLayer.(*layers.IPv4)

	// Must be from destination target to local source IP
	if !ip.SrcIP.Equal(dstIP) || !ip.DstIP.Equal(srcIP) {
		return false, nil
	}

	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return false, nil
	}
	tcp, _ := tcpLayer.(*layers.TCP)

	// Verify TCP ports match our probe flow
	if uint16(tcp.SrcPort) == dstPort && uint16(tcp.DstPort) == srcPort {
		return true, ip.SrcIP
	}

	return false, nil
}

// CaptureTCPOrICMPResponse captures either an intermediate ICMP or target TCP response
func (wc *WindowsCapture) CaptureTCPOrICMPResponse(srcIP, dstIP net.IP, srcPort, dstPort uint16) (gopacket.Packet, net.IP, error) {
	deadline := time.Now().Add(wc.timeout)

	for {
		select {
		case packet, ok := <-wc.packetChan:
			if !ok || packet == nil {
				return nil, nil, fmt.Errorf("packet capture closed unexpectedly")
			}

			matched, responseIP := MatchTCPOrICMPResponse(packet, srcIP, dstIP, srcPort, dstPort)
			if matched {
				return packet, responseIP, nil
			}

		case <-time.After(time.Until(deadline)):
			return nil, nil, fmt.Errorf("timeout waiting for probe response")
		}

		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("timeout waiting for probe response")
		}
	}
}

// Drain clears any queued packets from the channel before starting a new probe
func (wc *WindowsCapture) Drain() {
	for {
		select {
		case <-wc.packetChan:
		default:
			return
		}
	}
}
