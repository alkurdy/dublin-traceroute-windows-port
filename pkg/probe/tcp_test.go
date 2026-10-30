/* SPDX-License-Identifier: BSD-2-Clause */

package probe

import (
	"net"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// TestTCPProbe_CraftTCPPacket tests packet serialization, ensuring correct IPv4 TTL,
// destination IP, source port calculation (srcPort + flowID), destination port,
// and TCP SYN flag configuration for firewall traversal.
func TestTCPProbe_CraftTCPPacket(t *testing.T) {
	targetIP := net.ParseIP("172.253.122.100") // google.com
	srcIP := net.ParseIP("192.168.1.100")
	baseSrcPort := uint16(33434)
	dstPort := uint16(443) // HTTPS

	probe := &TCPProbe{
		Target:   targetIP,
		SrcIP:    srcIP,
		SrcPort:  baseSrcPort,
		DstPort:  dstPort,
		NumPaths: 4,
		MinTTL:   1,
		MaxTTL:   30,
	}

	testTTL := uint8(12)
	testFlowID := uint16(2)

	packetBytes, err := probe.craftTCPPacket(testTTL, testFlowID)
	if err != nil {
		t.Fatalf("Failed to craft TCP packet: %v", err)
	}

	if len(packetBytes) == 0 {
		t.Fatal("Crafted TCP packet is empty")
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
	if ip.Protocol != layers.IPProtocolTCP {
		t.Errorf("Expected TCP protocol, got %v", ip.Protocol)
	}

	// Verify TCP Layer attributes
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		t.Fatal("Crafted packet missing TCP layer")
	}
	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		t.Fatal("Failed to cast to layers.TCP")
	}

	expectedSrcPort := layers.TCPPort(baseSrcPort + testFlowID)
	if tcp.SrcPort != expectedSrcPort {
		t.Errorf("Expected source port %d, got %d", expectedSrcPort, tcp.SrcPort)
	}
	if tcp.DstPort != layers.TCPPort(dstPort) {
		t.Errorf("Expected destination port %d, got %d", dstPort, tcp.DstPort)
	}
	if !tcp.SYN {
		t.Error("Expected SYN flag to be true")
	}
	if tcp.ACK || tcp.FIN || tcp.RST {
		t.Errorf("Unexpected TCP flags set: ACK=%v, FIN=%v, RST=%v", tcp.ACK, tcp.FIN, tcp.RST)
	}
}

// TestTCPProbe_SetTimeout tests updating timeout on TCPProbe.
func TestTCPProbe_SetTimeout(t *testing.T) {
	probe := &TCPProbe{
		Timeout: 1 * time.Second,
	}

	newTimeout := 250 * time.Millisecond
	probe.SetTimeout(newTimeout)

	if probe.Timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, probe.Timeout)
	}
}

// TestTCPProbe_SetVerbose tests toggling verbose debug logging on TCPProbe.
func TestTCPProbe_SetVerbose(t *testing.T) {
	probe := &TCPProbe{
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

// TestTCPProbe_SetDelay tests updating inter-probe delay duration.
func TestTCPProbe_SetDelay(t *testing.T) {
	probe := &TCPProbe{
		Delay: 10 * time.Millisecond,
	}

	newDelay := 100 * time.Millisecond
	probe.SetDelay(newDelay)

	if probe.Delay != newDelay {
		t.Errorf("Expected delay %v, got %v", newDelay, probe.Delay)
	}
}
