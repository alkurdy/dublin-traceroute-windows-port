/* SPDX-License-Identifier: BSD-2-Clause */

package capture

import (
	"testing"
)

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
