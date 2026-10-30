/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"flag"
	"testing"
)

// resetFlags resets CommandLine and package flag variables for testing
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	target = flag.CommandLine.String("target", "", "Target host")
	useTCP = flag.CommandLine.Bool("tcp", false, "Use TCP")
	srcPort = flag.CommandLine.Uint("sport", 33434, "Starting source port")
	dstPort = flag.CommandLine.Uint("dport", 33434, "Destination port")
	minTTL = flag.CommandLine.Uint("min-ttl", 1, "Minimum TTL")
	maxTTL = flag.CommandLine.Uint("max-ttl", 30, "Maximum TTL")
	numPaths = flag.CommandLine.Uint("npaths", 4, "Number of paths")
	probeCount = flag.CommandLine.Uint("count", 1, "Probe count")
	timeout = flag.CommandLine.Uint("timeout", 0, "Timeout")
	outputJSON = flag.CommandLine.String("output-json", "", "Output JSON")
	showVersion = flag.CommandLine.Bool("version", false, "Show version")
	showAnalysis = flag.CommandLine.Bool("analyze", true, "Show analysis")
	showHelp = flag.CommandLine.Bool("help-routing", false, "Help routing")
	showTips = flag.CommandLine.Bool("tips", false, "Show tips")
	verbose = flag.CommandLine.Bool("verbose", false, "Verbose")
	listDevices = flag.CommandLine.Bool("list-devices", false, "List devices")
	device = flag.CommandLine.String("device", "", "Device")
}

// TestValidateParameters_MissingTarget tests that parameter validation fails
// when neither a -target flag nor a positional host argument is supplied.
func TestValidateParameters_MissingTarget(t *testing.T) {
	resetFlags()

	err := validateParameters()
	if err == nil {
		t.Fatal("Expected error for missing target, got nil")
	}
	if err.Error() != "target host is required" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestParseArgs_LeadingPositionalTarget tests positional host placed before flags
// (e.g. dublin-traceroute 8.8.8.8 -max-ttl 5 -min-ttl 2).
func TestParseArgs_LeadingPositionalTarget(t *testing.T) {
	resetFlags()

	parseArgs([]string{"8.8.8.8", "-min-ttl", "2", "-max-ttl", "5"})

	err := validateParameters()
	if err != nil {
		t.Fatalf("Expected leading positional target to validate cleanly, got: %v", err)
	}
	if *target != "8.8.8.8" {
		t.Errorf("Expected target to be '8.8.8.8', got: '%s'", *target)
	}
	if *minTTL != 2 {
		t.Errorf("Expected minTTL to be 2, got: %d", *minTTL)
	}
	if *maxTTL != 5 {
		t.Errorf("Expected maxTTL to be 5, got: %d", *maxTTL)
	}
}

// TestParseArgs_TrailingPositionalTarget tests positional host placed after flags
// (e.g. dublin-traceroute -min-ttl 2 -max-ttl 5 8.8.8.8).
func TestParseArgs_TrailingPositionalTarget(t *testing.T) {
	resetFlags()

	parseArgs([]string{"-min-ttl", "2", "-max-ttl", "5", "8.8.8.8"})

	err := validateParameters()
	if err != nil {
		t.Fatalf("Expected trailing positional target to validate cleanly, got: %v", err)
	}
	if *target != "8.8.8.8" {
		t.Errorf("Expected target to be '8.8.8.8', got: '%s'", *target)
	}
	if *minTTL != 2 {
		t.Errorf("Expected minTTL to be 2, got: %d", *minTTL)
	}
	if *maxTTL != 5 {
		t.Errorf("Expected maxTTL to be 5, got: %d", *maxTTL)
	}
}

// TestValidateParameters_PortValidation tests port range boundary checks for source and destination ports.
func TestValidateParameters_PortValidation(t *testing.T) {
	resetFlags()
	*target = "1.1.1.1"

	// Test invalid destination port: 0
	*dstPort = 0
	if err := validateParameters(); err == nil {
		t.Error("Expected error for destination port 0, got nil")
	}

	// Test invalid destination port: 99999 (> 65535)
	*dstPort = 99999
	if err := validateParameters(); err == nil {
		t.Error("Expected error for destination port 99999, got nil")
	}

	// Reset destination port and test invalid source port
	*dstPort = 33434
	*srcPort = 0
	if err := validateParameters(); err == nil {
		t.Error("Expected error for source port 0, got nil")
	}
}

// TestValidateParameters_TTLValidation tests TTL boundary checks and TTL inversion (min > max).
func TestValidateParameters_TTLValidation(t *testing.T) {
	resetFlags()
	*target = "1.1.1.1"

	// Test min-ttl 0
	*minTTL = 0
	*maxTTL = 30
	if err := validateParameters(); err == nil {
		t.Error("Expected error for min-ttl 0, got nil")
	}

	// Test min-ttl > max-ttl
	*minTTL = 20
	*maxTTL = 10
	if err := validateParameters(); err == nil {
		t.Error("Expected error for min-ttl > max-ttl, got nil")
	}
}

// TestValidateParameters_PortRangeOverflow tests detection of flow source port range overflow past 65535.
func TestValidateParameters_PortRangeOverflow(t *testing.T) {
	resetFlags()
	*target = "1.1.1.1"
	*srcPort = 65535
	*numPaths = 4 // 65535 + 4 - 1 = 65538 > 65535

	if err := validateParameters(); err == nil {
		t.Error("Expected error for port range overflow, got nil")
	}
}

// TestValidateParameters_ProbeCountValidation tests MTR probe count bounds (1-10 allowed).
func TestValidateParameters_ProbeCountValidation(t *testing.T) {
	resetFlags()
	*target = "1.1.1.1"

	// Count 0 rejected
	*probeCount = 0
	if err := validateParameters(); err == nil {
		t.Error("Expected error for probe count 0, got nil")
	}

	// Count 11 rejected (max is 10)
	*probeCount = 11
	if err := validateParameters(); err == nil {
		t.Error("Expected error for probe count 11, got nil")
	}
}

// TestValidateParameters_NonTargetFlags tests that utility flags (-list-devices, -version)
// can be used without requiring a target.
func TestValidateParameters_NonTargetFlags(t *testing.T) {
	resetFlags()
	*target = ""
	*listDevices = true

	if err := validateParameters(); err != nil {
		t.Errorf("Expected -list-devices to validate without target, got: %v", err)
	}
}
