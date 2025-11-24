# MTR Mode Implementation - Release Notes

## Version: 1.1.0 (MTR Mode Update)
**Date:** November 22, 2025
**Feature:** MTR-style continuous probing with per-hop statistics

---

## What's New

### MTR Mode (My TraceRoute)

Dublin Traceroute now supports **continuous probing** mode, sending multiple probes per hop to calculate:

- **Packet loss percentage** per hop
- **Latency statistics**: Min/Avg/Max RTT
- **Jitter analysis**: Standard deviation of RTT
- **Network stability** indicators
- **Return path issue detection** (ICMP filtering vs real problems)

This enables **return path analysis** by distinguishing between:
1. Forward path congestion (loss at specific hops)
2. Return path ICMP filtering (high loss everywhere but application works)
3. Network instability (high jitter/standard deviation)

---

## Usage

### Enable MTR Mode

Use the `-count` flag (1-10) to specify probes per hop:

```powershell
# Basic MTR mode - 3 probes per hop
dublin-traceroute -target google.com -count 3 -max-ttl 15

# Intensive analysis - 5 probes per hop
dublin-traceroute -target example.com -count 5 -max-ttl 12

# With TCP for better firewall traversal
dublin-traceroute -target api.example.com -tcp -dport 443 -count 3 -max-ttl 12
```

### Default Behavior

- `-count 1` (default): Traditional single-probe traceroute
- `-count 2-10`: MTR mode with statistical analysis

---

## Output Format

### Traditional Mode (count=1)
```
TTL=1 Flow=0: 10.4.0.129 (gateway.local) 1.5ms
TTL=1 Flow=1: 10.4.0.129 (gateway.local) 1.8ms
```

### MTR Mode (count ≥ 2)
```
=== MTR-Style Statistics ===
Target: google.com (142.250.80.78)
Duration: 45.2s

TTL Host                                      Loss%    Snt      Min      Avg      Max   StdDev
--------------------------------------------------------------------------------------------------
1   gateway.local (10.4.0.1)                   0.0%     12    1.2ms    1.5ms    2.1ms    0.3ms
2   border-router.isp.com (65.175.128.1)       8.3%     12    8.5ms   12.3ms   18.7ms    3.2ms
3   core-router.backbone.net (209.196.1.1)    16.7%     12   15.2ms   24.8ms   45.3ms   12.4ms

=== Analysis ===
⚠ High jitter at TTL 3 (209.196.1.1): Avg=24.8ms, StdDev=12.4ms
  • Suggests congestion, queuing, or load balancing
```

---

## Key Features

### 1. Automatic Problem Detection

MTR mode automatically identifies:

- ✅ **High packet loss** (>50% at any hop)
- ⚠️ **Complete loss** (100% everywhere - suggests ICMP filtering)
- 📊 **Increasing loss** toward target (forward path congestion)
- ⚡ **High jitter** (StdDev > Avg/2 - indicates queuing/congestion)

### 2. Return Path Analysis

While you cannot directly trace the return path, MTR mode helps **infer return path issues**:

| Symptom | Cause | Action |
|---------|-------|--------|
| High loss everywhere (60-100%) | ICMP rate limiting/filtering | Switch to TCP mode |
| Loss at specific hop(s) | Real congestion/routing issue | Contact ISP |
| High jitter (large StdDev) | Queuing/variable routing | Check for congestion |
| 0% loss but application slow | Forward path OK, app-level issue | Test application directly |

### 3. Statistical Confidence

More probes = better confidence in results:

- **1 probe**: Can miss intermittent issues
- **3 probes**: Good balance (recommended)
- **5 probes**: High confidence
- **10 probes**: Research-grade (slow)

---

## Implementation Details

### Code Changes

#### New Files
- `docs/MTR_MODE.md` - Complete MTR mode documentation (500+ lines)

#### Modified Files
1. **pkg/results/results.go**
   - Added `HopStatistics` struct with Loss%, Min/Avg/Max/StdDev RTT
   - Added `CalculateHopStatistics()` method for per-hop analysis
   - Added `PrintMTRStyle()` method for formatted output
   - Implemented standard deviation calculation using math package

2. **pkg/probe/udp.go**
   - Added `ProbeCount` field to `UDPProbe` struct
   - Modified `NewUDPProbe()` to accept `probeCount` parameter
   - Updated `Traceroute()` to loop through multiple probe rounds
   - Unique flow IDs per round: `uniqueFlowID = flowID + round*NumPaths`
   - Hostname lookup only on first round (performance)

3. **pkg/probe/tcp.go**
   - Added `ProbeCount` field to `TCPProbe` struct
   - Modified `NewTCPProbe()` to accept `probeCount` parameter
   - Updated `Traceroute()` to loop through multiple probe rounds
   - Same unique flow ID logic as UDP probe

4. **cmd/dublin-traceroute/main.go**
   - Added `-count` flag (default: 1, range: 1-10)
   - Added validation for probe count
   - Updated usage examples to show MTR mode
   - Conditional output: `PrintMTRStyle()` if count>1, else `PrintSummary()`
   - Updated both `NewUDPProbe()` and `NewTCPProbe()` calls

5. **README.md**
   - Added MTR mode examples to Quick Start
   - Updated feature list

6. **EXAMPLES.md**
   - Added entire "MTR-Style Statistics Mode" section (100+ lines)
   - Sample output interpretation
   - What each column means
   - When to use MTR mode

### Algorithm

```
For each TTL (1 to max-ttl):
  For each round (1 to probe-count):
    For each flow (1 to npaths):
      Send probe with unique flow ID = flow + (round × npaths)
      Wait for ICMP response
      Record: RTT, loss, response IP
      
After all probes complete:
  For each TTL:
    Calculate:
      - Loss% = (sent - received) / sent × 100
      - Min RTT = minimum of all RTTs
      - Avg RTT = mean of all RTTs
      - Max RTT = maximum of all RTTs
      - StdDev = standard deviation of RTTs
```

### Performance Characteristics

| Setting | Packets | Duration | Use Case |
|---------|---------|----------|----------|
| Default (count=1, 30 hops, 4 flows) | 120 | 2-5 min | Standard trace |
| Quick MTR (count=3, 12 hops, 2 flows) | 72 | 1-2 min | Recommended MTR |
| Deep MTR (count=5, 15 hops, 4 flows) | 300 | 5-10 min | Detailed analysis |
| Heavy MTR (count=10, 30 hops, 4 flows) | 1200 | 20-60 min | Research |

**Recommendation:** `-count 3 -max-ttl 12 -npaths 2` for optimal speed/statistics balance

---

## Interpreting Results

### Scenario 1: Healthy Network

```
TTL 1:  Loss 0.0%,  Avg 1.5ms,   StdDev 0.3ms   ✅
TTL 2:  Loss 0.0%,  Avg 12.3ms,  StdDev 2.1ms   ✅
TTL 3:  Loss 0.0%,  Avg 18.7ms,  StdDev 3.8ms   ✅
```
**Diagnosis:** Perfect network, no issues.

---

### Scenario 2: ICMP Filtering (Not a Problem)

```
TTL 1:  Loss 66.7%, Avg 343.8ms, StdDev 228.1ms ⚠️
TTL 2:  Loss 66.7%, Avg 93.1ms,  StdDev 86.2ms  ⚠️
TTL 3:  Loss 66.7%, Avg 255.7ms, StdDev 211.9ms ⚠️
```
**Diagnosis:** Return path ICMP rate limiting - application likely works fine.
**Action:** Use TCP mode or test application directly.

---

### Scenario 3: Real Network Problem

```
TTL 1:  Loss 0.0%,  Avg 1.5ms   ✅
TTL 2:  Loss 0.0%,  Avg 12.3ms  ✅
TTL 3:  Loss 50.0%, Avg 48.2ms  ❌ Problem starts here
TTL 4:  Loss 75.0%, Avg 92.1ms  ❌ Getting worse
```
**Diagnosis:** Real congestion or routing issue at TTL 3.
**Action:** Contact ISP, check network capacity.

---

### Scenario 4: High Jitter (Congestion)

```
TTL 4:  Loss 25.0%, Min 5.5ms, Avg 730.9ms, Max 1689.1ms, StdDev 709.1ms
```
**Diagnosis:** Severe queuing delay, link near capacity.
**Action:** Check for bandwidth saturation, consider QoS.

---

## Testing Performed

### Test Environments
1. **Corporate VDI Environment** (Atlantic Broadband)
   - High ICMP filtering detected (66-100% loss)
   - TCP mode performed better (lower loss)
   - MTR mode correctly identified ICMP rate limiting

2. **Public Internet Targets**
   - Cloudflare (1.1.1.1): Complete ICMP filtering
   - Google (8.8.8.8): Complete ICMP filtering
   - Expected behavior - public targets often rate-limit ICMP

### Test Cases
✅ Single probe mode (count=1): Works as before
✅ MTR mode with 3 probes: Statistical output correct
✅ MTR mode with 5+ probes: Increased statistical confidence
✅ UDP mode with MTR: Works correctly
✅ TCP mode with MTR: Works correctly
✅ Loss percentage calculation: Accurate
✅ RTT statistics (Min/Avg/Max): Correct
✅ Standard deviation calculation: Accurate
✅ Hostname resolution: Only on first round (performance)
✅ Unique flow IDs per round: No collisions
✅ Help output: Shows new -count flag
✅ Validation: Rejects count < 1 or > 10

---

## Known Limitations

1. **Cannot directly trace return path** - This is a fundamental limitation of traceroute. MTR mode helps **infer** return path issues through statistical analysis.

2. **ICMP filtering is common** - Many networks rate-limit or block ICMP Time Exceeded messages. High packet loss doesn't always indicate a problem. Use TCP mode for comparison.

3. **Performance impact** - MTR mode multiplies probe count. Recommended to reduce `-max-ttl` when using higher `-count` values.

4. **Hostname lookups add delay** - Only performed on first round to balance information vs speed.

5. **No bidirectional testing** - Cannot initiate traces from target back to source. Would require server mode (future enhancement).

---

## Future Enhancements

Potential additions for future versions:

1. **Continuous monitoring mode** (`-continuous`)
   - Real-time updating display like classic MTR
   - Run indefinitely until Ctrl+C

2. **Latency heatmap** (`-heatmap`)
   - Visual representation of RTT distribution per hop

3. **Comparison mode** (`-compare baseline.json`)
   - Automatic diff against saved baseline
   - Highlight changes (latency increases, new hops, loss changes)

4. **Server mode** (`-server`)
   - Listen for connection from remote dublin-traceroute
   - Enable true bidirectional testing

5. **Historical trending** (`-history`)
   - Track metrics over time
   - Database integration

6. **GeoIP integration** (`-geoip`)
   - Show geographic location of each hop
   - Detect unusual routing (e.g., traffic going overseas unexpectedly)

---

## Migration Guide

### Existing Scripts

No breaking changes - existing commands work identically:

```powershell
# Old command (still works)
dublin-traceroute -target google.com

# New command (adds MTR mode)
dublin-traceroute -target google.com -count 3
```

### New Recommended Workflow

1. **Quick test** (no MTR):
   ```powershell
   dublin-traceroute -target example.com -max-ttl 12 -npaths 2
   ```

2. **If issues detected, run MTR mode**:
   ```powershell
   dublin-traceroute -target example.com -count 5 -max-ttl 12 -npaths 2
   ```

3. **If high loss detected, try TCP**:
   ```powershell
   dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 12
   ```

4. **Always validate with application test**:
   ```powershell
   Test-NetConnection example.com -Port 443
   ```

---

## Documentation

### New Documentation Files
- **docs/MTR_MODE.md** - Complete MTR mode guide (4000+ words)
  - What is MTR mode
  - How to enable and use it
  - Interpreting output
  - Return path analysis strategy
  - Troubleshooting scenarios
  - Best practices
  - Advanced techniques

### Updated Documentation Files
- **README.md** - Added MTR mode to Quick Start
- **EXAMPLES.md** - Added MTR mode section with 5+ examples
- **TCP_VS_UDP.md** - Referenced MTR mode for protocol comparison

---

## Credits

Implementation based on:
- **MTR (My TraceRoute)** by Matt Kimball - Classic continuous traceroute tool
- **Dublin Traceroute** by Leonid Vasilyev - Original multipath detection algorithm
- **PathPing** (Microsoft Windows) - Inspiration for statistical analysis

---

## Support

For issues, questions, or feature requests:
- Create an issue on GitHub
- Include: Windows version, Npcap version, command used, full output
- For MTR mode issues: Include both `-count 1` and `-count 5` outputs for comparison

---

## Summary

MTR mode transforms Dublin Traceroute from a **single-probe path discovery tool** into a **comprehensive network diagnostics platform** capable of:

- ✅ Detecting intermittent connectivity issues
- ✅ Identifying packet loss and where it occurs
- ✅ Measuring network stability (jitter)
- ✅ Distinguishing ICMP filtering from real problems
- ✅ Inferring return path issues through statistical analysis
- ✅ Creating performance baselines for comparison

**Recommended Usage:**
```powershell
dublin-traceroute -target example.com -count 3 -max-ttl 12 -tcp -dport 443
```

This provides optimal balance of:
- Speed (12 hops, not 30)
- Statistical confidence (3 probes per hop)
- Firewall traversal (TCP mode)
- Comprehensive analysis (MTR-style output)
