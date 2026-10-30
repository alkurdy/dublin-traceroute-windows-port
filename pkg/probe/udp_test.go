/* SPDX-License-Identifier: BSD-2-Clause */

package probe

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// TestUDPProbe_CraftUDPPacket tests packet serialization, ensuring correct IPv4 TTL,
// destination IP, source port calculation (srcPort + flowID), destination port,
// and Dublin Traceroute 8-byte payload signature (0xDEADBEEFCAFEBABE).
func TestUDPProbe_CraftUDPPacket(t *testing.T) {
	targetIP := net.ParseIP("93.184.216.34") // example.com
	srcIP := net.ParseIP("192.168.1.100")
	baseSrcPort := uint16(33434)
	dstPort := uint16(33434)

	probe := &UDPProbe{
		Target:   targetIP,
		SrcIP:    srcIP,
		SrcPort:  baseSrcPort,
		DstPort:  dstPort,
		NumPaths: 4,
		MinTTL:   1,
		MaxTTL:   30,
	}

	testTTL := uint8(7)
	testFlowID := uint16(3)

	packetBytes, err := probe.craftUDPPacket(testTTL, testFlowID)
	if err != nil {
		t.Fatalf("Failed to craft UDP packet: %v", err)
	}

	if len(packetBytes) == 0 {
		t.Fatal("Crafted packet is empty")
	}

	// Parse crafted packet with gopacket
	packet := gopacket.NewPacket(packetBytes, layers.LayerTypeIPv4, gopacket.Default)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		t.Fatal("Crafted packet missing IPv4 layer")
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		t.Fatal("Failed to cast to layers.IPv4")
	}

	// Verify IP Header attributes
	if ip.TTL != testTTL {
		t.Errorf("Expected TTL %d, got %d", testTTL, ip.TTL)
	}
	if !ip.SrcIP.Equal(srcIP) {
		t.Errorf("Expected SrcIP %v, got %v", srcIP, ip.SrcIP)
	}
	if !ip.DstIP.Equal(targetIP) {
		t.Errorf("Expected DstIP %v, got %v", targetIP, ip.DstIP)
	}
	if ip.Protocol != layers.IPProtocolUDP {
		t.Errorf("Expected UDP protocol, got %v", ip.Protocol)
	}

	// Verify UDP Layer attributes
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer == nil {
		t.Fatal("Crafted packet missing UDP layer")
	}
	udp, ok := udpLayer.(*layers.UDP)
	if !ok {
		t.Fatal("Failed to cast to layers.UDP")
	}

	expectedSrcPort := layers.UDPPort(baseSrcPort + testFlowID)
	if udp.SrcPort != expectedSrcPort {
		t.Errorf("Expected source port %d, got %d", expectedSrcPort, udp.SrcPort)
	}
	if udp.DstPort != layers.UDPPort(dstPort) {
		t.Errorf("Expected destination port %d, got %d", dstPort, udp.DstPort)
	}

	// Verify Dublin Traceroute 8-byte signature payload
	expectedPayload := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0xCA, 0xFE, 0xBA, 0xBE}
	if !bytes.Equal(udp.Payload, expectedPayload) {
		t.Errorf("Payload mismatch. Expected %x, got %x", expectedPayload, udp.Payload)
	}
}

// TestUDPProbe_SetTimeout tests updating timeout on UDPProbe.
func TestUDPProbe_SetTimeout(t *testing.T) {
	probe := &UDPProbe{
		Timeout: 3 * time.Second,
	}

	newTimeout := 500 * time.Millisecond
	probe.SetTimeout(newTimeout)

	if probe.Timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, probe.Timeout)
	}
}

// TestUDPProbe_SetVerbose tests toggling verbose debug logging on UDPProbe.
func TestUDPProbe_SetVerbose(t *testing.T) {
	probe := &UDPProbe{
		Verbose: false,
	}

	probe.SetVerbose(true)
	if !probe.Verbose {
		t.Error("Expected Verbose to be true after SetVerbose(true)")
	}

	probe.SetVerbose(false)
	if probe.Verbose {
		t.Error("Expected Verbose to be false after SetVerbose(false)")
	}
}

// TestUDPProbe_SetDelay tests updating inter-probe delay duration.
func TestUDPProbe_SetDelay(t *testing.T) {
	probe := &UDPProbe{
		Delay: 10 * time.Millisecond,
	}

	newDelay := 50 * time.Millisecond
	probe.SetDelay(newDelay)

	if probe.Delay != newDelay {
		t.Errorf("Expected delay %v, got %v", newDelay, probe.Delay)
	}
}
