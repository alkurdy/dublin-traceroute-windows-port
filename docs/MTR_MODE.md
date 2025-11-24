# MTR Mode - Continuous Probing and Return Path Analysis

## What is MTR Mode?

MTR (My TraceRoute) mode sends **multiple probes per hop** instead of just one, enabling statistical analysis of network behavior. Dublin Traceroute's MTR mode helps identify:

- **Packet loss** per hop (what percentage of probes are dropped)
- **Latency statistics** (min/avg/max response times)
- **Jitter** (RTT variability - indicates congestion or queuing)
- **Return path issues** (asymmetric routing, ICMP filtering)
- **Intermittent problems** (issues that don't appear on single probe)

## Enabling MTR Mode

Use the `-count` flag to specify number of probes per hop (1-10):

```powershell
# Basic MTR mode - 3 probes per hop
dublin-traceroute -target google.com -count 3 -max-ttl 15

# Intensive analysis - 5 probes per hop with TCP
dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 12
```

**⚠️ Performance Impact:** MTR mode multiplies the number of probes:
- Default: 30 hops × 4 flows × 1 probe = 120 total packets
- MTR mode: 15 hops × 2 flows × 3 probes = 90 total packets (recommended)
- Heavy MTR: 30 hops × 4 flows × 5 probes = 600 total packets (slow!)

**💡 Recommendation:** Use `-count 3` with lower `-max-ttl 12-15` for best balance.

## Understanding the Output

### MTR Statistics Table

```
=== MTR-Style Statistics ===
Target: google.com (142.250.80.78)
Duration: 45.2s

TTL Host                                      Loss%    Snt      Min      Avg      Max   StdDev
--------------------------------------------------------------------------------------------------
1   gw-local.example.com (10.4.0.1)            0.0%     12    1.2ms    1.5ms    2.1ms    0.3ms
2   border-router.isp.com (65.175.128.1)       8.3%     12    8.5ms   12.3ms   18.7ms    3.2ms
3   core-router.backbone.net (209.196.1.1)    16.7%     12   15.2ms   24.8ms   45.3ms   12.4ms
4   google-peer.backbone.net (108.170.1.1)    33.3%     12   18.9ms   28.5ms   52.1ms   18.9ms
5   142.250.80.78                              0.0%     12   19.2ms   19.8ms   21.3ms    0.8ms
```

### Column Descriptions

| Column | Meaning | Good Values | Warning Values |
|--------|---------|-------------|----------------|
| **TTL** | Hop number (distance from you) | 1-30 | - |
| **Host** | Router IP and hostname | - | ??? means no response |
| **Loss%** | Percentage of probes that timed out | 0-5% | >20% |
| **Snt** | Total probes sent (flows × count) | Matches your settings | - |
| **Min** | Fastest response time | Low is good | - |
| **Avg** | Average response time | Steadily increasing | Big jumps |
| **Max** | Slowest response time | Close to Min | >>Avg suggests spikes |
| **StdDev** | Jitter (RTT consistency) | <Avg/2 | >Avg/2 is high jitter |

## Interpreting Results

### 1. Forward Path Issues

**Symptoms:**
- Loss increases as TTL increases
- High loss at specific hop(s), continues to target
- Max RTT much higher than Avg at problem hop

**Example:**
```
TTL 1:  Loss 0.0%,  Avg 1.5ms,  StdDev 0.3ms   ✅ Good
TTL 2:  Loss 0.0%,  Avg 12.3ms, StdDev 3.2ms   ✅ Good
TTL 3:  Loss 25.0%, Avg 24.8ms, StdDev 12.4ms  ⚠️ Problem starts here
TTL 4:  Loss 50.0%, Avg 38.5ms, StdDev 28.9ms  ❌ Congestion/loss
TTL 5:  Loss 75.0%, Avg 52.1ms, StdDev 45.3ms  ❌ Severe issues
```

**What it means:**
- Congestion or packet drops on forward path
- Router queue overflow at TTL 3
- May indicate bandwidth saturation or DoS

**Action:**
- Contact ISP if problem is in their network
- Check for bandwidth issues on your network
- Consider alternate routes or load balancing

---

### 2. Return Path Issues (ICMP Filtering)

**Symptoms:**
- High loss (60-100%) at ALL hops
- Even hop 1 (your gateway) shows loss
- BUT: Final destination responds (if you can ping it separately)

**Example:**
```
TTL 1:  Loss 66.7%, Avg 343.8ms, StdDev 228.1ms  ❌ High loss at gateway!
TTL 2:  Loss 66.7%, Avg 93.1ms,  StdDev 86.2ms   ❌ High loss everywhere
TTL 3:  Loss 66.7%, Avg 255.7ms, StdDev 211.9ms  ❌ Consistent high loss
TTL 4:  Loss 66.7%, Avg 730.9ms, StdDev 709.1ms  ❌ Same pattern
```

**What it means:**
- ICMP Time Exceeded responses are being rate-limited or filtered
- Forward path is likely fine
- Problem is with ICMP responses coming BACK to you
- This is NORMAL behavior for many ISPs and enterprise networks

**Why this happens:**
- Security policy: "Don't respond to traceroute"
- ICMP rate limiting to prevent DoS reconnaissance
- Firewall rules blocking ICMP inbound
- Asymmetric routing (return path uses different routers that filter ICMP)

**Action:**
- Use TCP mode instead: `-tcp -dport 80` or `-tcp -dport 443`
- TCP is less likely to be filtered
- If TCP also shows high loss, it's a real forward path issue
- Document as expected behavior if application works fine

---

### 3. High Jitter (Network Congestion)

**Symptoms:**
- StdDev > 50% of Avg RTT
- Max RTT >> Min RTT (e.g., Max 500ms, Min 20ms)
- Loss% may be moderate (10-30%)

**Example:**
```
TTL 4:  Loss 25.0%, Min 5.5ms, Avg 730.9ms, Max 1689.1ms, StdDev 709.1ms
                    ^^^^^       ^^^^^^^^^    ^^^^^^^^^^^   ^^^^^^^^^^^
                    Fast        Highly       Very          Huge
                    sometimes   variable     slow          variance
```

**What it means:**
- Router is queuing packets (congested link)
- Variable routing (load balancing causing different path latencies)
- QoS policies causing priority differences
- Intermittent congestion (peaks at certain times)

**Action:**
- Run MTR mode during different times of day
- Compare against baseline (save with `-output-json baseline.json`)
- Consider QoS/traffic shaping on your network
- May indicate need for bandwidth upgrade

---

### 4. ICMP Rate Limiting (False Positives)

**Symptoms:**
- Moderate loss (20-50%) at middle hops
- BUT final destination has 0% loss
- Avg/Min/Max RTT seem reasonable for responding probes

**Example:**
```
TTL 1:  Loss 0.0%,  Avg 1.5ms   ✅ Perfect
TTL 2:  Loss 0.0%,  Avg 12.3ms  ✅ Perfect
TTL 3:  Loss 33.3%, Avg 18.2ms  ⚠️ Some loss (but Avg is good)
TTL 4:  Loss 25.0%, Avg 24.1ms  ⚠️ Some loss (but Avg is good)
TTL 5:  Loss 0.0%,  Avg 19.8ms  ✅ Target responds perfectly!
```

**What it means:**
- Middle routers are intentionally rate-limiting ICMP responses
- This is a **false positive** - not a real problem
- Forward path and return path are both working fine
- Application traffic is NOT affected

**Action:**
- Ignore moderate loss at intermediate hops if target responds well
- This is expected behavior on the public internet
- Focus on end-to-end connectivity, not intermediate hop behavior

---

## Return Path Analysis Strategy

Dublin Traceroute MTR mode cannot directly trace the return path, but it can **infer return path problems** through statistical analysis.

### Three-Step Diagnostic Process

#### Step 1: Run MTR Mode with UDP (Default)
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15
```

**Analyze results:**
- ✅ Low loss everywhere → Network is healthy
- ❌ High loss (>50%) everywhere → Likely ICMP filtering
- ⚠️ High loss at specific hops → Potential congestion

---

#### Step 2: Run MTR Mode with TCP
```powershell
dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 15
```

**Compare UDP vs TCP:**
- **Same loss pattern** → Real network issue (affects both protocols)
- **TCP has lower loss** → ICMP filtering (return path issue, not forward)
- **TCP has higher loss** → TCP filtering (firewall blocking SYN packets)

---

#### Step 3: Test Application Directly
```powershell
# For web servers
Test-NetConnection example.com -Port 443

# Multiple tests
1..10 | ForEach-Object {
    Measure-Command { Test-NetConnection example.com -Port 443 }
}
```

**Final determination:**
- **Traceroute shows high loss, but application works** → Return path ICMP filtering
- **Traceroute and application both fail** → Real connectivity issue
- **High jitter in traceroute, app is slow** → Network congestion

---

## Best Practices

### 1. Choose Appropriate Probe Count

| Probe Count | Use Case | Time Impact |
|-------------|----------|-------------|
| `-count 1` | Default, single probe (no MTR) | Baseline |
| `-count 3` | Quick statistics, detect major issues | 3x slower |
| `-count 5` | Good statistical confidence | 5x slower |
| `-count 10` | Deep analysis, research | 10x slower |

**Recommendation:** Start with `-count 3`, increase only if needed.

---

### 2. Adjust Max TTL for MTR Mode

MTR mode takes longer, so reduce max TTL:

```powershell
# Normal mode - probe 30 hops
dublin-traceroute -target example.com -max-ttl 30

# MTR mode - probe only 15 hops (still gets good coverage)
dublin-traceroute -target example.com -count 5 -max-ttl 15
```

---

### 3. Use TCP for Corporate Networks

Corporate networks often filter ICMP heavily:

```powershell
# Try UDP first
dublin-traceroute -target internal-server.company.com -count 3 -max-ttl 10

# If high loss, switch to TCP
dublin-traceroute -target internal-server.company.com -tcp -dport 443 -count 3 -max-ttl 10
```

---

### 4. Combine with Application Testing

Always validate traceroute results with actual application behavior:

```powershell
# Run MTR mode
dublin-traceroute -target api.example.com -tcp -dport 443 -count 5 -max-ttl 12

# Then test actual HTTPS
Measure-Command { Invoke-WebRequest https://api.example.com -UseBasicParsing }
```

---

### 5. Save Baselines for Comparison

```powershell
# Create baseline during good performance
dublin-traceroute -target example.com -count 5 -max-ttl 15 -output-json baseline.json

# During issues, compare
dublin-traceroute -target example.com -count 5 -max-ttl 15 -output-json issue.json

# Then diff the JSON files to see what changed
```

---

## Troubleshooting Common Scenarios

### Scenario 1: "Everything Shows 100% Loss"

**Diagnosis:**
```powershell
dublin-traceroute -target example.com -count 3 -max-ttl 15
# Result: All hops show 100% loss
```

**Causes:**
1. **Npcap not installed or not running**
   - Solution: Reinstall Npcap with "WinPcap API-compatible mode"
2. **Not running as Administrator**
   - Solution: Right-click executable → "Run as administrator"
3. **Firewall blocking outbound UDP/ICMP**
   - Solution: Allow dublin-traceroute through firewall
4. **Network adapter selection issue**
   - Solution: Use `-list-devices` to see adapters, specify with `-device`

---

### Scenario 2: "High Loss But Application Works Fine"

**Diagnosis:**
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15
# Result: 60% loss at all hops

Test-NetConnection example.com -Port 443
# Result: Success, RTT 20ms
```

**Conclusion:** ICMP rate limiting (return path filtering) - **not a problem**.

**Action:** Document as expected, use TCP mode for clearer picture:
```powershell
dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 15
```

---

### Scenario 3: "Loss Starts at Specific Hop and Continues"

**Diagnosis:**
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15
# TTL 1-3: 0% loss
# TTL 4-15: 50% loss
```

**Conclusion:** Problem at or after TTL 4 (congestion or routing issue).

**Action:**
1. Identify the router at TTL 4
2. Check if it's your ISP's equipment
3. Contact ISP if problem persists
4. Consider alternate routes or providers

---

### Scenario 4: "Massive Jitter at One Hop"

**Diagnosis:**
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15
# TTL 5: Min 10ms, Avg 500ms, Max 2000ms, StdDev 800ms
```

**Conclusion:** Severe congestion or queuing at that router.

**Action:**
1. Run MTR at different times to see if time-dependent
2. Check if this hop is load-balanced (try more flows: `-npaths 8`)
3. May indicate link near capacity or DoS attack
4. If it's ISP equipment, report to ISP with MTR data

---

## Advanced Techniques

### Continuous Monitoring

Run MTR mode periodically and log results:

```powershell
# PowerShell script for continuous monitoring
$logFile = "C:\Logs\mtr-results.log"
while ($true) {
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Output "[$timestamp] Running MTR..." | Out-File -Append $logFile
    
    .\dublin-traceroute.exe -target critical-server.com -count 3 -max-ttl 12 `
        -output-json "C:\Logs\mtr-$timestamp.json"
    
    Start-Sleep -Seconds 300  # Run every 5 minutes
}
```

### Automated Alerting

Parse JSON output and alert on thresholds:

```powershell
$result = Get-Content baseline.json | ConvertFrom-Json

foreach ($hop in $result.hops.PSObject.Properties) {
    $stats = Calculate-HopStats $hop.Value
    
    if ($stats.LossPercent -gt 50 -and $stats.TTL -lt 10) {
        Send-Alert "High packet loss at TTL $($stats.TTL): $($stats.LossPercent)%"
    }
}
```

### Geographic Path Analysis

Combine MTR results with GeoIP lookup:

```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15 -output-json results.json

# Parse JSON and lookup each IP
$result = Get-Content results.json | ConvertFrom-Json
foreach ($hop in $result.hops) {
    # Use GeoIP service to map path geographically
    $location = Invoke-RestMethod "https://ipapi.co/$($hop.IP)/json"
    Write-Output "TTL $($hop.TTL): $($hop.IP) in $($location.city), $($location.country)"
}
```

---

## Summary

**MTR Mode Key Takeaways:**

1. ✅ **Use `-count 3` or `-count 5`** for statistical confidence
2. ✅ **Reduce `-max-ttl`** to 12-15 when using MTR mode
3. ✅ **Use TCP mode** (`-tcp -dport 443`) for corporate networks
4. ⚠️ **High loss everywhere** usually means ICMP filtering (not a real issue)
5. ⚠️ **Loss at specific hop** usually indicates real congestion
6. ⚠️ **High jitter** (large StdDev) indicates queuing or variable routing
7. 🔍 **Always validate** with application testing (`Test-NetConnection`)
8. 📊 **Save baselines** for comparison during issues

**When to Use MTR Mode:**
- Diagnosing intermittent connectivity issues
- Identifying packet loss and where it occurs
- Measuring network stability (jitter analysis)
- Distinguishing ICMP filtering from real problems
- Creating performance baselines
- Detecting congestion and routing changes

**When NOT to Use MTR Mode:**
- Quick connectivity tests (use default `-count 1`)
- Bandwidth is very limited
- Time-sensitive testing
- Initial exploration (start simple, add complexity only if needed)
