# Testing Guide: Dublin Traceroute for Windows

Comprehensive guide for testing the Windows port of Dublin Traceroute, covering offline unit tests, coverage verification, and privileged live network integration tests.

---

## 1. Test Architecture Overview

Testing is structured into two decoupled tiers:

```
+-------------------------------------------------------------------------+
| TIER 1: Offline Go Unit Tests (CI-Ready, Local & Automated)             |
| * Packages:                                                             |
|   - cmd/dublin-traceroute: CLI flag validation & positional arguments   |
|   - pkg/probe: UDP/TCP packet crafting, signatures, timeout/verbosity   |
|   - pkg/capture: Device enumeration & IP lookup error handling          |
|   - internal/platform: Privilege checking, Npcap install verification   |
|   - pkg/results: Serialization, path extraction, load-balancing         |
|     detection, latency calculations, hop statistics, & trace replay.    |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| TIER 2: End-to-End Live Integration Tests (Requires Npcap & Admin)       |
| * Script: test-dublin-traceroute.ps1                                    |
| * Tests: 18 scenarios verifying raw socket creation, Npcap packet      |
|   capture, UDP/TCP probes, MTR continuous probing, CLI flags, JSON      |
|   export, positional target syntax, and error boundaries.               |
+-------------------------------------------------------------------------+
```

---

## 2. Tier 1: Unit Testing (`go test`)

Tier 1 tests run in user-space and validate core packet craftsmanship, parameter validation, and analysis engine logic without requiring raw packet sniffing permissions or network connectivity.

### Running Unit Tests

Run all unit tests across all packages:
```powershell
go test -v ./...
```

Run tests specifically for individual packages:
```powershell
go test -v ./cmd/dublin-traceroute
go test -v ./pkg/probe
go test -v ./pkg/results
go test -v ./pkg/capture
go test -v ./internal/platform
```

### Checking Test Code Coverage
Check statement coverage for the results analysis engine:
```powershell
go test -cover ./pkg/results
```

Generate an HTML visual coverage report:
```powershell
go test -coverprofile=coverage.out ./pkg/results
go tool cover -html=coverage.out
```

### What the Unit Tests Verify

| Package | Test Name | Verification Details |
|---------|-----------|----------------------|
| `cmd/dublin-traceroute` | `TestValidateParameters_MissingTarget` | Asserts exit error when no target host is supplied. |
| `cmd/dublin-traceroute` | `TestParseArgs_LeadingPositionalTarget` | Validates `dublin-traceroute <host> [flags]` syntax. |
| `cmd/dublin-traceroute` | `TestParseArgs_TrailingPositionalTarget` | Validates `dublin-traceroute [flags] <host>` syntax. |
| `cmd/dublin-traceroute` | `TestValidateParameters_PortValidation` | Enforces port bounds (1-65535). |
| `cmd/dublin-traceroute` | `TestValidateParameters_TTLValidation` | Enforces TTL bounds (1-255) and prevents `min-ttl > max-ttl`. |
| `cmd/dublin-traceroute` | `TestValidateParameters_PortRangeOverflow` | Prevents flow source port math exceeding 65535. |
| `cmd/dublin-traceroute` | `TestValidateParameters_ProbeCountValidation` | Enforces MTR probe count bounds (1-10). |
| `cmd/dublin-traceroute` | `TestValidateParameters_NonTargetFlags` | Confirms utility flags (`-list-devices`, `-version`) run without a target. |
| `pkg/probe` | `TestUDPProbe_CraftUDPPacket` | Validates IPv4 TTL, target IP, source port flow offset, and `0xDEADBEEFCAFEBABE` payload signature. |
| `pkg/probe` | `TestUDPProbe_SetTimeout` | Verifies dynamic timeout configuration on UDP probe. |
| `pkg/probe` | `TestUDPProbe_SetVerbose` | Tests verbose mode toggle on UDP probe. |
| `pkg/probe` | `TestUDPProbe_SetDelay` | Verifies inter-probe delay adjustment. |
| `pkg/probe` | `TestTCPProbe_CraftTCPPacket` | Validates TCP SYN packet generation, TTL, destination port, and source port flow offset. |
| `pkg/probe` | `TestTCPProbe_SetTimeout` | Verifies dynamic timeout configuration on TCP probe. |
| `pkg/probe` | `TestTCPProbe_SetVerbose` | Tests verbose mode toggle on TCP probe. |
| `pkg/probe` | `TestTCPProbe_SetDelay` | Verifies inter-probe delay adjustment. |
| `pkg/capture` | `TestListDevices` | Confirms Npcap device enumeration populates device name and metadata. |
| `pkg/capture` | `TestFindDeviceByIP_NonExistent` | Validates clean error handling when querying non-existent IP address. |
| `internal/platform` | `TestCheckNpcapInstalled` | Verifies detection of Npcap/WinPcap system driver DLLs. |
| `internal/platform` | `TestGetNpcapInstallMessage` | Confirms install instructions contain documentation and compatible mode guides. |
| `internal/platform` | `TestIsAdmin` | Validates Windows token privilege queries. |
| `internal/platform` | `TestGetLocalIPv4Address` | Verifies operational unicast IPv4 adapter detection. |
| `internal/platform` | `TestGetPublicNATIP` | Verifies fast public NAT WAN IP discovery via DNS reflection. |
| `pkg/results` | `TestTracerouteResult_ToJSON` | Serializes result to JSON, verifies round-trip unmarshaling, checks field preservation. |
| `pkg/results` | `TestTracerouteResult_GetPaths` | Extracts multi-flow paths, validates hop sequences, verifies target termination. |
| `pkg/results` | `TestTracerouteResult_GetHopCount` | Validates max TTL computation across all probe flows. |
| `pkg/results` | `TestTracerouteResult_GetUniqueHosts` | Deduplicates response IP addresses across multiple probes and flows. |
| `pkg/results` | `TestTracerouteResult_CalculateHopStatistics` | Computes MTR-style per-hop statistics: min/max/avg RTT and packet loss rates. |
| `pkg/results` | `TestAnalyzeNetwork_LoadBalancing` | Validates multipath detection when multiple distinct router IPs respond at the same TTL. |
| `pkg/results` | `TestAnalyzeNetwork_PacketLoss` | Confirms packet loss calculation across successful vs timeout flows. |
| `pkg/results` | `TestAnalyzeNetwork_HighLatencyJump` | Validates automated detection of high-latency router hops (>100ms or 3x jump). |
| `pkg/results` | `TestAnalyzeNetwork_RealRecordedTrace` | Replays `test_trace.json` (real 1,300-line production trace) to ensure parser resilience. |
| `pkg/results` | `TestTracerouteResult_PrintSummary_NoPanic` | Validates console table rendering and MTR output formatters execute without panicking. |
| `pkg/results` | `TestTracerouteResult_PrintSummary_ShowAnalysisOption` | Confirms `PrintSummary` respects `-analyze=false` suppression. |
| `pkg/results` | `TestPrintParameterInsights_Variants` | Validates parameter-specific diagnostic guidance across UDP, TCP, port 53, multipath, and MTR modes. |
| `pkg/results` | `TestIsTargetReached` | Asserts target reachability detection across intermediate and destination hops. |
| `pkg/results` | `TestTracerouteResult_GetUniquePaths_Deduplication` | Confirms flows with identical router hop sequences deduplicate into 1 unique path with averaged RTTs. |
| `pkg/results` | `TestTracerouteResult_AllTimeouts_NoFalseMultipath` | Asserts 100% timeout traces do not falsely claim multiple load-balanced paths. |

---

## 3. Tier 2: End-to-End Live Integration Tests

Tier 2 exercises the compiled binary against live targets, verifying Windows Npcap packet driver integration and raw socket mechanics.

### Prerequisites
1. **Npcap:** Installed with "WinPcap API-compatible mode" enabled.
2. **Elevated Shell:** PowerShell running as Administrator (required for raw packet capture).

### Running the Test Script

Compile the binary first:
```powershell
go build -v -o dublin-traceroute.exe .\cmd\dublin-traceroute
```

Execute the test harness:
```powershell
.\test-dublin-traceroute.ps1
```

### The 18 E2E Test Scenarios

The PowerShell harness executes 18 distinct functional checks:

1. **Basic UDP Traceroute:** Sends UDP probes to `google.com` (ports 33434+).
2. **Positional Target:** Tests running target without flag (`dublin-traceroute.exe google.com`).
3. **TCP Traceroute:** Sends TCP SYN probes on port 443 with responsive probe timeout.
4. **Custom Ports:** Uses source port 40000 and destination port 53.
5. **TTL Range:** Restricts probing to hops 2 through 3.
6. **Multipath Detection:** Probes with multiple concurrent flows (`-npaths 2`).
7. **MTR Mode:** Executes multiple probe cycles (`-count 2`).
8. **Custom Timeout:** Tests probe timeout configuration (`-timeout 500`).
9. **JSON Export:** Validates file output (`-output-json test_trace.json`).
10. **Show Version:** Validates non-admin version output.
11. **Routing Help:** Tests help screen display.
12. **Network Tips:** Validates diagnostic advice rendering.
13. **Verbose Output:** Inspects `[DEBUG]` logging during execution (`-verbose`).
14. **Analyze Flag Disabled:** Confirms analysis section suppression (`-analyze=false`).
15. **Device Listing:** Tests Npcap interface discovery (`-list-devices`).
16. **Error Handling (Target):** Asserts exit behavior when target argument is omitted.
17. **Error Handling (Port):** Asserts error handling on invalid port numbers (`-dport 99999`).
18. **Error Handling (Device):** Asserts error handling when specifying non-existent network adapter (`-device`).

---

## 4. Troubleshooting Common Failures

| Error | Root Cause | Fix |
|-------|------------|-----|
| `failed to open device \Device\NPF_...: Access is denied` | Non-admin terminal execution | Re-launch PowerShell as Administrator. |
| `no Npcap device found` | Npcap service stopped or not installed | Reinstall Npcap from npcap.com; ensure the driver service is started (`sc query npcap`). |
| `RealRecordedTrace: test_trace.json not found` | Test executed from non-package working directory | Run tests using package syntax: `go test ./pkg/results` from repo root. |
