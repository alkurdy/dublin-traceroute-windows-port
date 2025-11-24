# MTR Mode - Quick Reference Card

## Enable MTR Mode

```powershell
# Add -count flag (2-10)
dublin-traceroute -target <host> -count 3
```

## Recommended Settings

| Scenario | Command |
|----------|---------|
| **Quick MTR** | `dublin-traceroute -target <host> -count 3 -max-ttl 12 -npaths 2` |
| **TCP MTR** | `dublin-traceroute -target <host> -tcp -dport 443 -count 3 -max-ttl 12` |
| **Deep Analysis** | `dublin-traceroute -target <host> -count 5 -max-ttl 15 -npaths 4` |
| **Internal Network** | `dublin-traceroute -target <host> -count 3 -max-ttl 8 -npaths 2 -timeout 1000` |

## Output Columns

| Column | Meaning | Good | Warning |
|--------|---------|------|---------|
| **Loss%** | Packet loss | 0-5% | >20% |
| **Min** | Fastest RTT | Low | - |
| **Avg** | Average RTT | Steady increase | Big jumps |
| **Max** | Slowest RTT | Close to Min | Much higher |
| **StdDev** | Jitter | <Avg/2 | >Avg/2 |

## Quick Diagnosis

### ✅ Healthy Network
```
TTL 1:  Loss 0.0%,  Avg 1.5ms,   StdDev 0.3ms
TTL 2:  Loss 0.0%,  Avg 12.3ms,  StdDev 2.1ms
TTL 3:  Loss 0.0%,  Avg 18.7ms,  StdDev 3.8ms
```

### ⚠️ ICMP Filtering (False Positive)
```
TTL 1:  Loss 66.7%, Avg 343.8ms, StdDev 228.1ms
TTL 2:  Loss 66.7%, Avg 93.1ms,  StdDev 86.2ms
TTL 3:  Loss 66.7%, Avg 255.7ms, StdDev 211.9ms
```
**High loss everywhere = ICMP rate limiting (not a real problem)**

### ❌ Real Network Problem
```
TTL 1:  Loss 0.0%,  Avg 1.5ms   ← Good
TTL 2:  Loss 0.0%,  Avg 12.3ms  ← Good
TTL 3:  Loss 50.0%, Avg 48.2ms  ← Problem starts here
TTL 4:  Loss 75.0%, Avg 92.1ms  ← Getting worse
```
**Loss at specific hop = real congestion or routing issue**

### ⚡ High Jitter (Congestion)
```
TTL 4:  Loss 25.0%, Min 5.5ms, Avg 730.9ms, Max 1689.1ms, StdDev 709.1ms
        ^^^^^^^^^^^^            ^^^^^^^^^^               ^^^^^^^^^^^^
        Some loss              Highly variable          Huge variance
```
**Large StdDev = queuing delays, congestion, or variable routing**

## Troubleshooting Workflow

### Step 1: Run UDP MTR
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15
```

### Step 2: If High Loss, Try TCP
```powershell
dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 15
```

### Step 3: Compare Results

| UDP Loss | TCP Loss | Diagnosis |
|----------|----------|-----------|
| High | High | Real network issue |
| High | Low | ICMP filtering (not a problem) |
| Low | High | TCP filtering (firewall) |
| Low | Low | Network is healthy |

### Step 4: Test Application
```powershell
Test-NetConnection example.com -Port 443
```

## Performance Tips

| Probes | Hops | Flows | Total Packets | Duration | Use Case |
|--------|------|-------|---------------|----------|----------|
| 1 | 30 | 4 | 120 | 2-5 min | Standard trace |
| 3 | 12 | 2 | 72 | 1-2 min | **Recommended MTR** |
| 5 | 15 | 4 | 300 | 5-10 min | Deep analysis |
| 10 | 30 | 4 | 1200 | 20-60 min | Research |

**Formula:** `Total Packets = Hops × Flows × Probes`

## Key Flags

| Flag | Default | Range | Description |
|------|---------|-------|-------------|
| `-count` | 1 | 1-10 | Probes per hop (MTR mode) |
| `-max-ttl` | 30 | 1-255 | Maximum hops to trace |
| `-npaths` | 4 | 1-256 | Parallel flows (multipath) |
| `-timeout` | UDP: 3000<br>TCP: 1000 | - | Milliseconds per probe |
| `-tcp` | false | - | Use TCP SYN instead of UDP |
| `-dport` | 33434 | 1-65535 | TCP: target port (80, 443)<br>UDP: dest port |

## Common Scenarios

### Internal Server (Corporate Network)
```powershell
dublin-traceroute -target 10.172.23.218 -count 3 -max-ttl 8 -npaths 2 -timeout 1000
```

### Public Website (HTTPS)
```powershell
dublin-traceroute -target google.com -tcp -dport 443 -count 3 -max-ttl 12
```

### API Endpoint (HTTP)
```powershell
dublin-traceroute -target api.example.com -tcp -dport 80 -count 5 -max-ttl 15
```

### VPN Tunnel
```powershell
dublin-traceroute -target vpn-server.company.com -count 3 -max-ttl 10
```

## Analysis Indicators

### 📊 Forward Path Issues
- Loss increases with TTL
- Problem at specific hop continues to target

### 🔄 Return Path Issues (ICMP Filtering)
- High loss everywhere (even hop 1)
- BUT: Application works fine (`Test-NetConnection` succeeds)

### ⚡ Network Congestion
- High jitter (StdDev > Avg/2)
- Max RTT >> Min RTT
- Variable performance

### 🚫 ICMP Rate Limiting
- Moderate loss (20-50%) at middle hops
- Final destination has 0% loss
- Application works perfectly

## Save & Compare

### Create Baseline
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15 -output-json baseline.json
```

### During Issues
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15 -output-json issue.json
```

### Compare
```powershell
# PowerShell comparison
$baseline = Get-Content baseline.json | ConvertFrom-Json
$issue = Get-Content issue.json | ConvertFrom-Json

# Compare loss rates, latencies, paths
```

## When to Use MTR Mode

### ✅ Use MTR Mode When:
- Diagnosing intermittent connectivity
- Measuring network stability
- Identifying packet loss location
- Creating performance baselines
- Detecting congestion
- Distinguishing ICMP filtering from real issues

### ❌ Don't Use MTR Mode When:
- Quick connectivity test (use default `-count 1`)
- Very limited bandwidth
- Time-sensitive testing
- Initial exploration (start simple)

## Remember

1. **High loss everywhere** → Usually ICMP filtering (not a problem)
2. **Loss at specific hop** → Real congestion or routing issue
3. **High jitter** → Network congestion or variable routing
4. **Always validate** with `Test-NetConnection` or application testing

## Help Commands

```powershell
# Show all flags
dublin-traceroute -help

# Version info
dublin-traceroute -version

# List network devices
dublin-traceroute -list-devices

# Return path explanation
dublin-traceroute -help-routing

# Tips for baseline comparison
dublin-traceroute -tips
```

## Documentation

- **Full MTR Guide:** `docs/MTR_MODE.md`
- **Usage Examples:** `EXAMPLES.md`
- **TCP vs UDP:** `docs/TCP_VS_UDP.md`
- **Release Notes:** `RELEASE_NOTES_MTR.md`
