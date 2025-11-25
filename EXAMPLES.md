# Dublin Traceroute - Usage Examples

## ⚡ Performance Tips

**Dublin Traceroute probes multiple hops × multiple flows = many packets!**

- Default settings: 30 hops × 4 flows = **120 probes** (takes 2-5 minutes)
- Quick settings: 10 hops × 2 flows = **20 probes** (takes ~30 seconds)
- MTR mode: Add `-count 3` multiplies probes by 3 (use with lower max-ttl!)

**💡 RECOMMENDED: Start with quick settings, increase only if needed:**
```powershell
dublin-traceroute -target google.com -max-ttl 12 -npaths 2
```

**🆕 For MTR mode (continuous probing), use fewer hops:**
```powershell
dublin-traceroute -target google.com -count 3 -max-ttl 12 -npaths 2
```

---

## MTR-Style Statistics Mode (NEW!)

MTR mode sends multiple probes per hop to calculate packet loss, latency statistics, and jitter. Perfect for identifying network issues and return path problems.

### 1. Basic MTR Mode - 3 Probes Per Hop
```powershell
dublin-traceroute -target google.com -count 3 -max-ttl 15
```
**What it does:** Sends 3 probes to each of 15 hops, calculates loss%, min/avg/max/stddev RTT

**Use case:** Quick statistical overview of network path quality

**Output includes:**
- Packet loss percentage per hop
- Min/Avg/Max RTT per hop
- Standard deviation (jitter)
- Automatic detection of high loss areas
- Identification of congestion/queuing issues

---

### 2. MTR Mode with TCP for Firewall Traversal
```powershell
dublin-traceroute -target example.com -tcp -dport 443 -count 5 -max-ttl 12
```
**What it does:** Uses TCP SYN packets (better firewall traversal) with 5 probes per hop

**Use case:** Corporate networks where ICMP/UDP may be filtered

**When to use TCP:**
- Corporate/enterprise networks
- Testing HTTPS path (port 443)
- Testing HTTP path (port 80)
- When UDP traceroute shows high packet loss

---

### 3. Intensive MTR Analysis - 10 Probes Per Hop
```powershell
dublin-traceroute -target 1.1.1.1 -count 10 -max-ttl 8 -npaths 1 -timeout 500
```
**What it does:** Sends 10 probes per hop with fast timeout (500ms) to first 8 hops

**Use case:** Deep analysis of specific problem area with statistical confidence

**⚠️ Warning:** This takes time! 8 hops × 10 probes = 80 packets

---

### 4. Return Path Analysis
```powershell
dublin-traceroute -target example.com -count 5 -max-ttl 15 -tcp -dport 80
```
**What it does:** Multiple probes help identify if high loss is forward path vs return path issue

**Interpretation:**
- **High loss everywhere (80-100%)**: Likely return path issue (ICMP responses being dropped)
- **High loss only at specific hop**: Congestion or rate limiting at that router
- **Increasing loss toward target**: Forward path congestion/filtering
- **High jitter (large StdDev)**: Queuing delays, congestion, or variable routing

**Use case:** Diagnosing asymmetric routing, one-way packet loss, ICMP rate limiting

---

### 5. MTR Mode to Internal Server
```powershell
dublin-traceroute -target 10.172.23.218 -count 3 -max-ttl 5 -npaths 2 -timeout 1000
```
**What it does:** Quick internal network analysis with moderate timeout

**Use case:** Testing path to database server, application server, or network device

---

## Understanding MTR Output

### Sample MTR Output:
```
=== MTR-Style Statistics ===
TTL Host                                      Loss%    Snt      Min      Avg      Max   StdDev
1   gw.example.net. (10.4.20.121)    33.3%     12   36.4ms  343.8ms  680.2ms  228.1ms
2   router1.example.net. (192.0.2.1)  16.7%     12    2.0ms   93.1ms  200.2ms   86.2ms
3   router2.example.net. (198.51.100.1)   8.3%     12    3.5ms   18.2ms   25.8ms    7.1ms
```

**What each column means:**
- **TTL**: Hop number (distance from your computer)
- **Host**: Router IP address and hostname
- **Loss%**: Percentage of probes that timed out (0% = perfect)
- **Snt**: Total probes sent to this hop
- **Min/Avg/Max**: Fastest/average/slowest response time
- **StdDev**: Jitter - how much RTT varies (low = consistent, high = unstable)

**What to look for:**
- ✅ **Loss% < 5%**: Excellent
- ⚠️ **Loss% 5-20%**: Acceptable (may be ICMP rate limiting)
- ❌ **Loss% > 50%**: Problem area - investigate further
- ⚠️ **High StdDev**: Network congestion, queuing, or load balancing
- 📊 **Increasing loss toward target**: Forward path issues
- 🔄 **High loss everywhere**: Likely return path/ICMP filtering

---

## Basic Examples

### 1. Simple Traceroute to a Website
```powershell
dublin-traceroute -target google.com
```
**What it does:** Traces the path from your computer to Google using default settings (4 parallel flows, max 30 hops).

**Use case:** Quick check to see the route to any website.

---

### 2. Traceroute to Specific IP Address
```powershell
dublin-traceroute -target 8.8.8.8
```
**What it does:** Traces path to Google's DNS server.

**Use case:** Test connectivity to known stable target.

---

### 3. Quick Trace (Limited Hops)
```powershell
dublin-traceroute -target 192.168.1.1 -max-ttl 5
```
**What it does:** Only traces first 5 hops (useful for internal networks).

**Use case:** Testing path to gateway or local server where you know it's nearby.

---

### 3b. TCP Trace (Better Firewall Traversal)
```powershell
dublin-traceroute -target google.com -tcp -dport 443
```
**What it does:** Uses TCP SYN packets to port 443 instead of UDP.

**Use case:** 
- More reliable when UDP is blocked by firewalls
- Tests actual path your HTTPS traffic takes
- Often gets responses from more routers
- Better success rate in corporate networks

**Why use TCP:**
- Many routers prioritize/respond to TCP over UDP
- Firewalls often allow TCP to ports 80/443
- More representative of real application traffic

---

## Load Balancing Detection

### 4. Detect Multiple Paths with More Flows
```powershell
dublin-traceroute -target google.com -npaths 8
```
**What it does:** Tests 8 parallel flows to increase chances of discovering load-balanced paths.

**Use case:** See if your ISP or target uses ECMP (Equal Cost Multi-Path) routing.

**Expected output:**
```
Discovered 3 unique path(s):
  ℹ️  Multiple paths detected - your traffic is load-balanced across 3 routes
```

---

### 5. Maximum Path Detection
```powershell
dublin-traceroute -target cloudflare.com -npaths 16 -max-ttl 20
```
**What it does:** Tests 16 flows with 20 hop limit for comprehensive multipath analysis.

**Use case:** Deep analysis of complex load-balanced infrastructure like CDNs.

---

## Saving Results for Later

### 6. Create Baseline for Future Comparison
```powershell
dublin-traceroute -target company-server.com -output-json baseline.json
```
**What it does:** Saves complete trace data to JSON file.

**Use case:** Document "normal" routing for later troubleshooting.

---

### 7. Capture During Problem
```powershell
dublin-traceroute -target company-server.com -output-json problem-$(Get-Date -Format "yyyyMMdd-HHmm").json
```
**What it does:** Saves trace with timestamp in filename.

**Use case:** Document network state during performance issues.

**Follow-up:** Compare JSON files to find route changes or latency increases.

---

### 8. Daily Monitoring
```powershell
# Run via Task Scheduler
dublin-traceroute -target critical-service.com -output-json "C:\Logs\trace-$(Get-Date -Format yyyyMMdd).json"
```
**What it does:** Creates daily trace logs.

**Use case:** Track routing stability over time.

---

## Troubleshooting Scenarios

### 9. Diagnose Slow Connection
```powershell
# Step 1: Baseline when working
dublin-traceroute -target slow-server.com -output-json working.json

# Step 2: During slowness
dublin-traceroute -target slow-server.com -output-json slow.json

# Step 3: Compare the files manually or look for latency spikes
```
**What to look for:**
- Different router IPs (route changed)
- Higher RTT values at specific hop
- New high-latency hop in path

---

### 10. Test Internal Network
```powershell
dublin-traceroute -target 10.5.100.50 -max-ttl 10
```
**What it does:** Traces path to internal server.

**Use case:** 
- Verify VLAN routing
- Check if traffic goes through expected switches/routers
- Diagnose internal routing issues

---

### 11. Test Different Ports
```powershell
# TCP trace to common ports (better success rate)
dublin-traceroute -target example.com -tcp -dport 443  # HTTPS
dublin-traceroute -target example.com -tcp -dport 80   # HTTP
dublin-traceroute -target example.com -tcp -dport 22   # SSH

# UDP trace (original method)
dublin-traceroute -target example.com -dport 53   # DNS
```
**What it does:** Tests both TCP and UDP with different destination ports.

**Use case:** 
- TCP traces often work better through firewalls
- Some networks treat TCP/UDP traffic differently
- Port-based QoS or routing policies
- Compare UDP vs TCP path differences

**💡 Pro Tip:** If UDP traceroute has high packet loss, try `-tcp -dport 443` for better results.

---

### 12. Skip Local Hops
```powershell
dublin-traceroute -target remote-server.com -min-ttl 5 -max-ttl 20
```
**What it does:** Starts trace at hop 5, skips local network hops 1-4.

**Use case:** Focus on ISP/backbone routing, ignore local infrastructure you already know.

---

## Learning & Understanding

### 13. Understand Return Path Routing
```powershell
dublin-traceroute -help-routing
```
**What it does:** Shows educational explanation about:
- Why you only see forward path
- Asymmetric routing concept
- How to analyze return paths

---

### 14. Learn Route Comparison Techniques
```powershell
dublin-traceroute -tips
```
**What it does:** Shows tips for comparing traces over time to detect route changes.

---

### 15. List Available Network Adapters
```powershell
dublin-traceroute -list-devices
```
**What it does:** Shows all network interfaces available for packet capture.

**Use case:** 
- Verify Npcap is working
- Select specific adapter if you have multiple
- Troubleshoot capture issues

---

## Real-World Scenarios

### Scenario 1: ISP Performance Investigation
**Problem:** Connection to work VPN is slow today.

```powershell
# Test to VPN server
dublin-traceroute -target vpn.company.com -npaths 8 -output-json vpn-slow.json
```

**Look for:**
- High latency at ISP's routers (first few hops)
- Route going through unexpected geography
- Packet loss at specific hop

**Compare with:** Previous baseline to see what changed.

---

### Scenario 2: CDN Routing Validation
**Problem:** Want to verify CDN is routing you to nearby edge server.

```powershell
dublin-traceroute -target cdn.example.com -npaths 6
```

**Look for:**
- Geographic path in analysis (should show nearby cities)
- Low hop count (CDN should be close)
- Multiple paths (CDN uses heavy load balancing)

**Expected good result:**
```
Geographic Path: Your City → Nearby CDN Pop
Unique Routers: 5-8
Average Latency: < 50ms
```

---

### Scenario 3: Intermittent Packet Loss Investigation
**Problem:** Video calls drop randomly.

```powershell
# Run multiple times
for($i=1; $i -le 5; $i++) {
    dublin-traceroute -target teams.microsoft.com -output-json "trace-$i.json"
    Start-Sleep -Seconds 60
}
```

**Look for:**
- Route flapping (different paths each time)
- Consistent high latency at same hop
- Packet loss patterns

---

### Scenario 4: After ISP Maintenance
**Problem:** Verify routing is back to normal after ISP work.

```powershell
# Run before maintenance (if possible)
dublin-traceroute -target 8.8.8.8 -output-json pre-maintenance.json

# Run after maintenance
dublin-traceroute -target 8.8.8.8 -output-json post-maintenance.json
```

**Compare:**
- Are you using same routers?
- Is latency similar?
- Any new load balancing behavior?

---

### Scenario 5: Cloud Provider Path Analysis
**Problem:** Testing connectivity to AWS us-east-1.

```powershell
dublin-traceroute -target ec2.us-east-1.amazonaws.com -npaths 8 -max-ttl 25
```

**Look for:**
- Direct peering or goes through backbone
- Geographic path (should head toward Virginia)
- Multiple paths (AWS uses extensive load balancing)

---

## Advanced Usage

### 16. Comprehensive Analysis with All Features
```powershell
dublin-traceroute -target example.com -npaths 12 -max-ttl 25 -output-json comprehensive-$(Get-Date -Format "yyyyMMdd-HHmmss").json
```
**What it does:** 
- 12 flows for maximum multipath detection
- 25 hops to reach distant targets
- Timestamped JSON for records

**Use case:** Complete documentation for escalating issues to ISP.

---

### 17. Focus on Specific Hop Range
```powershell
dublin-traceroute -target example.com -min-ttl 8 -max-ttl 15
```
**What it does:** Only probes hops 8-15.

**Use case:** You've identified problem area and want to focus analysis there.

---

### 18. Test with Specific Device
```powershell
# First list devices
dublin-traceroute -list-devices

# Then use specific one
dublin-traceroute -target example.com -device "\Device\NPF_{YOUR-GUID-HERE}"
```
**What it does:** Forces use of specific network adapter.

**Use case:** Multi-homed systems where you want to test specific interface.

---

## Monitoring & Automation

### 19. Scheduled Monitoring Script
```powershell
# Save as Monitor-Route.ps1
$target = "critical-service.com"
$logDir = "C:\NetworkLogs"

# Create log directory if needed
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

# Run trace
$timestamp = Get-Date -Format "yyyyMMdd-HHmm"
$outputFile = Join-Path $logDir "trace-$timestamp.json"

& "dublin-traceroute.exe" -target $target -npaths 8 -output-json $outputFile

# Check for high latency in last trace (parse JSON)
$result = Get-Content $outputFile | ConvertFrom-Json
$avgLatency = $result.average_rtt

if($avgLatency -gt 200ms) {
    # Alert: High latency detected
    Write-Host "WARNING: High latency detected: $avgLatency"
    # Send email, Teams notification, etc.
}
```

**Use case:** Automated monitoring with alerting.

---

### 20. Compare Two Traces
```powershell
# Helper function to compare
function Compare-Routes {
    param(
        [string]$BaselineFile,
        [string]$CurrentFile
    )
    
    $baseline = Get-Content $BaselineFile | ConvertFrom-Json
    $current = Get-Content $CurrentFile | ConvertFrom-Json
    
    Write-Host "=== Route Comparison ===" -ForegroundColor Cyan
    Write-Host "Baseline: $BaselineFile"
    Write-Host "Current:  $CurrentFile"
    Write-Host ""
    
    # Compare average latency
    Write-Host "Average Latency:"
    Write-Host "  Baseline: $($baseline.average_rtt)"
    Write-Host "  Current:  $($current.average_rtt)"
    
    # Compare unique routers
    Write-Host "`nUnique Routers:"
    Write-Host "  Baseline: $($baseline.unique_routers)"
    Write-Host "  Current:  $($current.unique_routers)"
    
    # More detailed comparison can be added
}

# Usage
Compare-Routes -BaselineFile "baseline.json" -CurrentFile "current.json"
```

---

## Quick Reference

| Scenario | Command |
|----------|---------|
| Basic UDP trace | `dublin-traceroute -target google.com` |
| TCP trace (HTTPS) | `dublin-traceroute -target google.com -tcp -dport 443` |
| TCP trace (HTTP) | `dublin-traceroute -target google.com -tcp -dport 80` |
| Local network | `dublin-traceroute -target 192.168.1.1 -max-ttl 5` |
| Find load balancing | `dublin-traceroute -target example.com -npaths 8` |
| Save baseline | `dublin-traceroute -target server.com -output-json baseline.json` |
| Quick test (fast) | `dublin-traceroute -target 8.8.8.8 -max-ttl 10 -npaths 2` |
| Comprehensive | `dublin-traceroute -target example.com -npaths 12 -max-ttl 25` |
| Learn about routing | `dublin-traceroute -help-routing` |
| See adapters | `dublin-traceroute -list-devices` |

---

## Pro Tips

1. **Save baselines when things work well** - You can't troubleshoot without knowing what "normal" looks like

2. **Use TCP for better results** - If UDP traceroute has high packet loss or many timeouts, try:
   ```powershell
   dublin-traceroute -target example.com -tcp -dport 443
   ```
   TCP often gets responses from more routers, especially through firewalls.

3. **Use descriptive filenames** - Include target, date, and context:
   ```powershell
   dublin-traceroute -target vpn.company.com -output-json "vpn-morning-slow-20251122.json"
   ```

4. **Run multiple times** - Network is dynamic, single trace may not be representative

5. **Higher npaths = better multipath detection** - But slower execution

6. **Normal timeout rates: 30-70%** - Don't panic at packet loss in traceroute

7. **Focus on latency jumps** - Big increases (>100ms) indicate problem areas

8. **Compare geographic paths** - Unexpected routing (e.g., coast-to-coast when target is local) indicates issues

9. **Document for escalations** - JSON files are great evidence when calling ISP support

10. **Choose protocol wisely:**
    - **UDP (default)**: Standard traceroute, works in most networks
    - **TCP**: Better for corporate networks, firewalls, testing web traffic paths

---

## Common Patterns

**Morning performance issues:**
```powershell
# Run during problem hours
dublin-traceroute -target problem-site.com -npaths 8 -output-json morning-slow.json

# Run during good hours  
dublin-traceroute -target problem-site.com -npaths 8 -output-json afternoon-fast.json

# Compare: Look for route changes or latency patterns
```

**VPN troubleshooting:**
```powershell
# Test before VPN connects
dublin-traceroute -target 8.8.8.8 -output-json pre-vpn.json

# Test after VPN connects
dublin-traceroute -target 8.8.8.8 -output-json during-vpn.json

# Compare: VPN routes everything differently
```

**Load balancer validation:**
```powershell
# Maximum path detection
dublin-traceroute -target your-loadbalancer.com -npaths 16 -max-ttl 15

# Should show: "Multiple paths detected - traffic load-balanced across X routes"
```
