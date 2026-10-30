/* SPDX-License-Identifier: BSD-2-Clause */

package capture

import (
	"net"
	"testing"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// helper to build a mock ICMP packet
func craftMockICMP(outerSrc, outerDst net.IP, icmpType layers.ICMPv4TypeCode, embeddedSrc, embeddedDst net.IP) gopacket.Packet {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}

	ipOuter := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		Protocol: layers.IPProtocolICMPv4,
		SrcIP:    outerSrc,
		DstIP:    outerDst,
	}

	icmp := &layers.ICMPv4{
		TypeCode: icmpType,
	}

	var payload []byte
	if embeddedSrc != nil && embeddedDst != nil {
		embBuf := gopacket.NewSerializeBuffer()
		embIP := &layers.IPv4{
			Version:  4,
			IHL:      5,
			TTL:      1,
			Protocol: layers.IPProtocolUDP,
			SrcIP:    embeddedSrc,
			DstIP:    embeddedDst,
		}
		_ = gopacket.SerializeLayers(embBuf, opts, embIP, gopacket.Payload([]byte{0, 0, 0, 0, 0, 0, 0, 0}))
		payload = embBuf.Bytes()
	}

	if len(payload) > 0 {
		_ = gopacket.SerializeLayers(buf, opts, ipOuter, icmp, gopacket.Payload(payload))
	} else {
		_ = gopacket.SerializeLayers(buf, opts, ipOuter, icmp)
	}

	return gopacket.NewPacket(buf.Bytes(), layers.LayerTypeIPv4, gopacket.Default)
}

// TestMatchICMPResponse_TimeExceededMatching asserts that a TimeExceeded response
// embedding the probe's source and destination IP is accepted.
func TestMatchICMPResponse_TimeExceededMatching(t *testing.T) {
	localIP := net.ParseIP("192.168.12.210")
	targetIP := net.ParseIP("8.8.8.8")
	routerIP := net.ParseIP("100.36.56.1")

	pkt := craftMockICMP(routerIP, localIP, layers.CreateICMPv4TypeCode(layers.ICMPv4TypeTimeExceeded, 0), localIP, targetIP)
	matched, src := MatchICMPResponse(pkt, localIP, targetIP, 0)
	if !matched {
		t.Fatal("Expected matching TimeExceeded packet to be accepted")
	}
	if !src.Equal(routerIP) {
		t.Errorf("Expected router IP %v, got %v", routerIP, src)
	}
}

// TestMatchICMPResponse_DestinationUnreachableUnrelatedTarget asserts that an
// unrelated background ICMP DestinationUnreachable packet (e.g. from an external server
// like example.com) is REJECTED when its embedded target does not match our probe.
func TestMatchICMPResponse_DestinationUnreachableUnrelatedTarget(t *testing.T) {
	localIP := net.ParseIP("192.168.12.210")
	targetIP := net.ParseIP("8.8.8.8")
	unrelatedHost := net.ParseIP("198.51.100.4") // example.com

	// External host sends DestinationUnreachable about unrelated traffic, NOT to 8.8.8.8
	pkt := craftMockICMP(unrelatedHost, localIP, layers.CreateICMPv4TypeCode(layers.ICMPv4TypeDestinationUnreachable, 3), localIP, unrelatedHost)

	matched, _ := MatchICMPResponse(pkt, localIP, targetIP, 0)
	if matched {
		t.Fatal("CRITICAL: Unrelated DestinationUnreachable packet from outside host was incorrectly matched!")
	}
}

// TestMatchICMPResponse_DestinationUnreachableTargetReached asserts that a
// DestinationUnreachable (Port Unreachable) sent by the actual probe target
// (verifying target reached) is accepted.
func TestMatchICMPResponse_DestinationUnreachableTargetReached(t *testing.T) {
	localIP := net.ParseIP("192.168.12.210")
	targetIP := net.ParseIP("8.8.8.8")

	// Target replies with Port Unreachable
	pkt := craftMockICMP(targetIP, localIP, layers.CreateICMPv4TypeCode(layers.ICMPv4TypeDestinationUnreachable, 3), localIP, targetIP)

	matched, src := MatchICMPResponse(pkt, localIP, targetIP, 0)
	if !matched {
		t.Fatal("Expected matching DestinationUnreachable (Port Unreachable) to be accepted")
	}
	if !src.Equal(targetIP) {
		t.Errorf("Expected target IP %v, got %v", targetIP, src)
	}
}

// TestMatchICMPResponse_EchoReply asserts that EchoReply is only accepted if
// the outer source matches the probe target IP.
func TestMatchICMPResponse_EchoReply(t *testing.T) {
	localIP := net.ParseIP("192.168.12.210")
	targetIP := net.ParseIP("8.8.8.8")
	fakeSender := net.ParseIP("1.2.3.4")

	// Reply from target -> should match
	validPkt := craftMockICMP(targetIP, localIP, layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoReply, 0), nil, nil)
	matched, src := MatchICMPResponse(validPkt, localIP, targetIP, 0)
	if !matched || !src.Equal(targetIP) {
		t.Errorf("Expected valid EchoReply to match target %v", targetIP)
	}

	// Reply from unrelated host -> must reject
	invalidPkt := craftMockICMP(fakeSender, localIP, layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoReply, 0), nil, nil)
	matched, _ = MatchICMPResponse(invalidPkt, localIP, targetIP, 0)
	if matched {
		t.Error("Expected fake EchoReply to be rejected")
	}
}

// TestListDevices tests enumerating network devices when Npcap is present on Windows.
func TestListDevices(t *testing.T) {
	devices, err := ListDevices()
	if err != nil {
		t.Logf("ListDevices returned error (expected if Npcap missing or unprivileged): %v", err)
		return
	}

	if len(devices) == 0 {
		t.Log("No network devices enumerated by Npcap")
		return
	}

	// Verify device struct fields are populated properly
	foundNamed := false
	for _, dev := range devices {
		if dev.Name != "" {
			foundNamed = true
			break
		}
	}

	if !foundNamed {
		t.Error("Expected at least one device with non-empty Name")
	}
}

// TestFindDeviceByIP_NonExistent tests that searching for a non-existent IP address
// correctly returns a descriptive error rather than panicking.
func TestFindDeviceByIP_NonExistent(t *testing.T) {
	_, err := FindDeviceByIP("203.0.113.199") // TEST-NET-3 non-routable dummy IP
	if err == nil {
		t.Error("Expected error when searching for non-existent IP, got nil")
	}
}
