// +build windows

/* SPDX-License-Identifier: BSD-2-Clause */

package capture

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"

	"github.com/atlanticbb/dublin-traceroute-windows/internal/platform"
)

// WindowsCapture handles packet capture on Windows using Npcap
type WindowsCapture struct {
	handle  *pcap.Handle
	iface   string
	timeout time.Duration
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

	// Open device for capture
	// Snapshot length 65535 (max), promiscuous mode enabled
	handle, err := pcap.OpenLive(device, 65535, true, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %w\n"+
			"Ensure Npcap is installed and running as administrator", device, err)
	}

	return &WindowsCapture{
		handle:  handle,
		iface:   device,
		timeout: timeout,
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
	packetSource := gopacket.NewPacketSource(wc.handle, wc.handle.LinkType())
	packetChan := packetSource.Packets()

	deadline := time.Now().Add(wc.timeout)

	for {
		select {
		case packet := <-packetChan:
			if packet == nil {
				return nil, nil, fmt.Errorf("packet capture closed unexpectedly")
			}

			// Parse ICMP layer
			icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
			if icmpLayer == nil {
				continue
			}

			icmp, _ := icmpLayer.(*layers.ICMPv4)

			// Check if this is the expected ICMP type
			if expectedType != 0 && icmp.TypeCode != expectedType {
				continue
			}

			// Parse IP layer to check source/destination
			ipLayer := packet.Layer(layers.LayerTypeIPv4)
			if ipLayer == nil {
				continue
			}

			ip, _ := ipLayer.(*layers.IPv4)

			// For Time Exceeded messages, check the embedded IP packet
			if icmp.TypeCode.Type() == layers.ICMPv4TypeTimeExceeded {
				// The payload contains the original IP header + 8 bytes of data
				// We want to verify this was our packet
				if len(icmp.Payload) < 20 {
					continue // Not enough data for IP header
				}

				// Parse embedded IP header
				embeddedIP := &layers.IPv4{}
				err := embeddedIP.DecodeFromBytes(icmp.Payload, gopacket.NilDecodeFeedback)
				if err != nil {
					continue
				}

				// Check if embedded packet matches our probe
				if embeddedIP.SrcIP.Equal(srcIP) && embeddedIP.DstIP.Equal(dstIP) {
					return packet, ip.SrcIP, nil
				}
			} else if icmp.TypeCode.Type() == layers.ICMPv4TypeDestinationUnreachable ||
				icmp.TypeCode.Type() == layers.ICMPv4TypeEchoReply {
				// For other ICMP types, just check the outer IP addresses
				if ip.DstIP.Equal(srcIP) {
					return packet, ip.SrcIP, nil
				}
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

	packetSource := gopacket.NewPacketSource(wc.handle, wc.handle.LinkType())
	packetChan := packetSource.Packets()

	deadline := time.Now().Add(wc.timeout)

	for len(packets) < count {
		select {
		case packet := <-packetChan:
			if packet == nil {
				break
			}

			// Parse ICMP layer
			icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
			if icmpLayer == nil {
				continue
			}

			icmp, _ := icmpLayer.(*layers.ICMPv4)

			// Parse IP layer
			ipLayer := packet.Layer(layers.LayerTypeIPv4)
			if ipLayer == nil {
				continue
			}

			ip, _ := ipLayer.(*layers.IPv4)

			// Check for Time Exceeded with embedded packet matching our probe
			if icmp.TypeCode.Type() == layers.ICMPv4TypeTimeExceeded {
				if len(icmp.Payload) >= 20 {
					embeddedIP := &layers.IPv4{}
					err := embeddedIP.DecodeFromBytes(icmp.Payload, gopacket.NilDecodeFeedback)
					if err == nil && embeddedIP.SrcIP.Equal(srcIP) && embeddedIP.DstIP.Equal(dstIP) {
						packets = append(packets, packet)
						sources = append(sources, ip.SrcIP)
					}
				}
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
