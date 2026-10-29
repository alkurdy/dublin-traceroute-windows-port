# Testing Guide: Dublin Traceroute for Windows

Comprehensive guide for testing the Windows port of Dublin Traceroute, covering offline unit tests, coverage verification, and privileged live network integration tests.

---

## 1. Test Architecture Overview

Testing is structured into two decoupled tiers:

```
+-------------------------------------------------------------------------+
| TIER 1: Offline Go Unit Tests (CI-Ready, No Admin, No Npcap Required)   |
| * Package: pkg/results                                                  |
| * Tests: Serialization, path extraction, load-balancing detection,     |
|   latency calculations, hop statistics, and real-trace replay.          |
+-------------------------------------------------------------------------+
                                    |
                                    v
+-------------------------------------------------------------------------+
| TIER 2: End-to-End Live Integration Tests (Requires Npcap & Admin)       |
| * Script: test-dublin-traceroute.ps1                                    |
| * Tests: Raw socket creation, Npcap packet capture, UDP/TCP probes,     |
|   MTR continuous probing, CLI flag validation, JSON export.             |
+-------------------------------------------------------------------------+
```

---

## 2. Tier 1: Unit Testing (`go test`)

Tier 1 tests run entirely in user-space without network dependencies, raw sockets, or administrative privileges. Suitable for local validation and automated GitHub Actions runners.

### Running Unit Tests

Run all unit tests with verbose output:
```powershell
go test -v ./...
```

Run tests specifically for the results analysis engine:
```powershell
go test -v ./pkg/results
```

### Checking Test Code Coverage
Check statement coverage for the analysis and statistics engine:
```powershell
go test -cover ./pkg/results
```
*Current benchmark: **>80% statement coverage** across `analysis.go` and `results.go`.*

Generate an HTML visual coverage report:
```powershell
go test -coverprofile=coverage.out ./pkg/results
go tool cover -html=coverage.out
```

### What the Unit Tests Verify

| Test Name | Test Target | Verification Details |
|-----------|-------------|----------------------|
| `TestTracerouteResult_ToJSON` | `results.go` | Serializes result to JSON, verifies round-trip unmarshaling, checks field preservation. |
| `TestTracerouteResult_GetPaths` | `results.go` | Extracts multi-flow paths, validates hop sequences, verifies target termination. |
| `TestTracerouteResult_GetHopCount` | `results.go` | Validates max TTL computation across all probe flows. |
| `TestTracerouteResult_GetUniqueHosts` | `results.go` | Deduplicates response IP addresses across multiple probes and flows. |
| `TestTracerouteResult_CalculateHopStatistics` | `results.go` | Computes MTR-style per-hop statistics: min/max/avg RTT and packet loss rates. |
| `TestAnalyzeNetwork_LoadBalancing` | `analysis.go` | Validates multipath detection when multiple distinct router IPs respond at the same TTL. |
| `TestAnalyzeNetwork_PacketLoss` | `analysis.go` | Confirms packet loss calculation across successful vs timeout flows. |
| `TestAnalyzeNetwork_HighLatencyJump` | `analysis.go` | Validates automated detection of high-latency router hops (>100ms or 3x jump). |
| `TestAnalyzeNetwork_RealRecordedTrace` | `analysis.go` | Replays `test_trace.json` (real 1,300-line production trace) to ensure parser resilience. |
| `TestTracerouteResult_PrintSummary_NoPanic` | `results.go` | Validates console table rendering and MTR output formatters execute without panicking. |

---

## 3. Tier 2: End-to-End Live Integration Tests

Tier 2 exercises the compiled binary against live targets, verifying Windows Npcap packet driver integration and raw socket mechanics.

### Prerequisites
1. **Npcap:** Installed with "WinPcap API-compatible mode" enabled.
2. **Elevated Shell:** PowerShell running as Administrator (required for raw packet capture).

### Running the Test Script

Compile the binary first:
```powershell
go build -v -o cmd\dublin-traceroute\dublin-traceroute.exe .\cmd\dublin-traceroute
```

Execute the test harness:
```powershell
.\test-dublin-traceroute.ps1
```

### The 15 E2E Test Scenarios

The PowerShell harness executes 15 distinct functional checks:

1. **Basic UDP Traceroute:** Sends UDP probes to `google.com` (ports 33434+).
2. **TCP Traceroute:** Sends TCP SYN probes on port 443.
3. **Custom Ports:** Uses source port 40000 and destination port 53.
4. **TTL Range:** Restricts probing to hops 5 through 6.
5. **Multipath Detection:** Probes with multiple concurrent flows (`-npaths 2`).
6. **MTR Mode:** Executes multiple probe cycles (`-count 2`).
7. **Custom Timeout:** Tests probe timeout configuration (`-timeout 500`).
8. **JSON Export:** Validates file output (`-output-json test_trace.json`).
9. **Show Version:** Validates non-admin version output.
10. **Routing Help:** Tests help screen display.
11. **Network Tips:** Validates diagnostic advice rendering.
12. **Verbose Output:** Inspects debug logging during execution.
13. **Device Listing:** Tests Npcap interface discovery (`-list-devices`).
14. **Error Handling (Target):** Asserts exit behavior when target argument is omitted.
15. **Error Handling (Port):** Asserts error handling on invalid port numbers (`-dport 99999`).

---

## 4. Troubleshooting Common Failures

| Error | Root Cause | Fix |
|-------|------------|-----|
| `failed to open device \Device\NPF_...: Access is denied` | Non-admin terminal execution | Re-launch PowerShell as Administrator. |
| `no Npcap device found` | Npcap service stopped or not installed | Reinstall Npcap from npcap.com; ensure the driver service is started (`sc query npcap`). |
| `RealRecordedTrace: test_trace.json not found` | Test executed from non-package working directory | Run tests using package syntax: `go test ./pkg/results` from repo root. |
