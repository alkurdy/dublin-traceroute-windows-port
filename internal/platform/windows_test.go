/* SPDX-License-Identifier: BSD-2-Clause */

package platform

import (
	"strings"
	"testing"
)

// TestCheckNpcapInstalled tests checking for Npcap / WinPcap installation on Windows.
func TestCheckNpcapInstalled(t *testing.T) {
	installed, err := CheckNpcapInstalled()
	if err != nil {
		t.Fatalf("CheckNpcapInstalled error: %v", err)
	}
	t.Logf("Npcap installed: %v", installed)
}

// TestGetNpcapInstallMessage tests that the Npcap installation guidance message
// contains download links and key instructions for Windows users.
func TestGetNpcapInstallMessage(t *testing.T) {
	msg := GetNpcapInstallMessage()
	if !strings.Contains(msg, "npcap.com") {
		t.Error("Expected install message to contain npcap.com URL")
	}
	if !strings.Contains(msg, "WinPcap API-compatible") {
		t.Error("Expected install message to mention WinPcap compatibility mode")
	}
}

// TestIsAdmin tests checking administrative privilege token on Windows.
func TestIsAdmin(t *testing.T) {
	isAdmin, err := IsAdmin()
	if err != nil {
		t.Fatalf("IsAdmin failed with error: %v", err)
	}
	t.Logf("Running as administrator: %v", isAdmin)
}

// TestGetLocalIPv4Address tests resolving the operational local IPv4 unicast address.
func TestGetLocalIPv4Address(t *testing.T) {
	ip, err := GetLocalIPv4Address()
	if err != nil {
		t.Logf("GetLocalIPv4Address error (may occur if disconnected): %v", err)
		return
	}
	if ip == "" {
		t.Error("Expected non-empty IPv4 address string")
	}
	t.Logf("Discovered local IPv4: %s", ip)
}

// TestGetPublicNATIP tests fast external NAT IP discovery via DNS reflection.
func TestGetPublicNATIP(t *testing.T) {
	natIP := GetPublicNATIP()
	t.Logf("Discovered Public NAT IP: %s", natIP)
	// NAT IP may be empty if offline, but must not panic or cause fatal error
}
