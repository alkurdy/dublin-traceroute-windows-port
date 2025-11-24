# Dublin Traceroute - User Guide

## What This Tool Shows You

### ✅ Forward Path Analysis
Dublin Traceroute shows the **path your packets take FROM your computer TO the target**:
- Each "hop" is a router that forwards your data
- You see latency (delay) at each hop
- You can detect if traffic takes multiple paths (load balancing)

### ❌ What It Doesn't Show
- **Return path**: The route FROM target BACK to you (usually different!)
- End-to-end application performance
- Bandwidth or throughput

Use `-help-routing` to understand why return paths are different.

---

## Quick Start

### Basic Usage
```powershell
# Simple trace to any website
dublin-traceroute -target google.com

# Trace to specific IP
dublin-traceroute -target 8.8.8.8

# Limit hops for faster results
dublin-traceroute -target 192.168.1.1 -max-ttl 10
```

### Detect Load Balancing
```powershell
# Use more parallel flows to find multiple paths
dublin-traceroute -target google.com -npaths 8
```

### Save for Later Analysis
```powershell
# Save baseline
dublin-traceroute -target example.com -output-json baseline.json

# Compare later during issues
dublin-traceroute -target example.com -output-json problem.json
# Then manually compare the JSON files
```

---

## Understanding the Output

### Path Display
```
Path 0 (3 hops):
   1: your-gateway (192.168.1.1)                         2ms
   5: isp-router (65.175.128.72)                        15ms
  10: backbone-router (168.143.191.66)                  45ms
```

**What each line means:**
- Number (1, 5, 10): TTL value - how many hops away
- Hostname/IP: The router at that position
- Time (2ms, 15ms, 45ms): Round-trip time to that hop

**Missing hops (gaps in numbers):**
- Normal! Some routers don't respond to traceroute
- Doesn't affect your actual traffic

### Network Analysis Section

#### 🔀 Load Balancing
```
Multiple paths detected - traffic load-balanced across 4 routes
```
**What it means:** Your traffic can take different routes. This is GOOD because:
- Increases reliability (if one path fails, others work)
- Better performance (distributes load)
- Common on modern Internet infrastructure

#### ⚠️ High Latency Hops
```
Hop 8: ae-7.router.net (168.143.191.66)
└─ Latency: 250ms (jump: +200ms)
└─ Likely intercontinental or satellite link
```
**What it means:** Significant delay added at this hop. Common causes:
- Long-distance (e.g., cross-country or international)
- Satellite links
- Congested router
- Normal for geographically distant targets

#### 📉 Packet Loss
```
70% of probes timed out
```
**Don't panic!** High timeout rates are usually NORMAL because:
- Many routers deliberately ignore traceroute packets (ICMP rate limiting)
- Protects routers from being overwhelmed
- **Your actual traffic is NOT affected** - only traceroute probes

**When to worry:**
- If you're also experiencing slow connections to the target
- If packet loss suddenly increases compared to baseline

#### 🌍 Geographic Path
```
Your traffic appears to traverse: Boston → New York → London
```
**What it means:** Educated guess based on router hostnames about geographic path.
- Helps understand why latency is high
- Shows if traffic takes inefficient route

---

## Common Questions

### Why do I see timeouts?
**Short answer:** Normal. Routers often don't respond to traceroute.

**Long answer:** Routers prioritize forwarding your real traffic (web, email, etc.) over responding to diagnostic probes. When busy, they'll drop traceroute responses first. This doesn't mean packets are being lost - just that the router chose not to tell you it's there.

### Why are there gaps in hop numbers?
Some routers at those TTL values didn't respond. Your packets still went through them.

Example:
```
1: gateway
2: (no response)
3: (no response)  
4: backbone-router
```
Your packets traveled: gateway → router 2 → router 3 → backbone-router.
Routers 2 and 3 just didn't reply to the traceroute probe.

### What's a "good" result?
**Low latency (< 100ms):**
- Target is nearby or well-connected
- Good network path

**Multiple paths:**
- Load balancing is working
- Increased reliability

**Some timeouts (20-70%):**
- Completely normal
- Doesn't indicate problems

**High latency (> 500ms):**
- Target is far away
- Check where latency jumps occur
- Normal for international connections

### Is the return path the same?
**NO!** Internet routing is asymmetric:
- Forward: Your PC → ISP A → Backbone X → Target
- Return: Target → Backbone Y → ISP B → Your PC

Both directions work, but use different routers. This is normal and beneficial.

Use `-help-routing` for detailed explanation.

### How do I troubleshoot slow connections?
1. **Establish baseline:**
   ```powershell
   dublin-traceroute -target example.com -output-json baseline.json
   ```

2. **Run during problem:**
   ```powershell
   dublin-traceroute -target example.com -output-json problem.json
   ```

3. **Compare results:**
   - Did route change? (Different router IPs)
   - Did latency increase? (Higher RTT values)
   - New packet loss patterns?

4. **Look for:**
   - Sudden latency jumps at specific hop
   - Route going through unexpected geography
   - All traffic hitting single overloaded path

### Can I use this for internal networks?
Yes! Perfect for:
```powershell
# Trace to internal server
dublin-traceroute -target 10.0.5.100 -max-ttl 10

# Trace to gateway
dublin-traceroute -target 192.168.1.1 -max-ttl 5
```

---

## Advanced Usage

### Detect Routing Changes
Save regular traces and compare:
```powershell
# Daily baseline
dublin-traceroute -target critical-server.com -output-json "trace-$(Get-Date -Format yyyyMMdd).json"
```

Compare JSON files to see:
- Route flapping (unstable routing)
- Failover events
- ISP routing policy changes

### Test Specific Ports
```powershell
# Some networks filter UDP differently per port
dublin-traceroute -target example.com -dport 53    # DNS port
dublin-traceroute -target example.com -dport 443   # HTTPS port
```

### Focus on Specific Hop Range
```powershell
# Skip first 5 hops (local network)
dublin-traceroute -target example.com -min-ttl 6 -max-ttl 15
```

---

## Interpreting for Different Skill Levels

### For Non-Technical Users
**What you need to know:**
- Lower numbers = better (for latency)
- Multiple paths = good (redundancy)
- Timeouts = usually not a problem
- Save the JSON file if you need to show your IT department

### For Network Admins
**Key insights:**
- Asymmetric routing detection
- Load balancer behavior under different 5-tuple hashes
- Per-hop latency for pinpointing congestion
- ECMP path diversity validation
- BGP route stability over time

### For Developers
**Use cases:**
- Diagnose API latency issues
- Validate CDN routing
- Understand anycast behavior
- Document network topology for architecture diagrams
- A/B test different target IPs for optimal routing

---

## Troubleshooting

### "Administrator privileges required"
**Solution:** Right-click PowerShell → "Run as administrator"

### "Npcap is required but not installed"
**Solution:**
1. Download: https://npcap.com/#download
2. Install with "WinPcap API-compatible Mode" enabled
3. Restart computer

### 100% timeouts
**Possible causes:**
1. Firewall blocking outbound UDP
2. Firewall blocking inbound ICMP
3. Target doesn't exist or is down
4. Try different destination port: `-dport 53`

### Results don't match normal tracert
**This is expected!** Dublin Traceroute:
- Uses UDP (tracert uses ICMP)
- Tests multiple paths simultaneously
- May trigger different filtering rules

---

## Learn More

```powershell
# Understand return path routing
dublin-traceroute -help-routing

# Tips for comparing routes over time
dublin-traceroute -tips

# See all options
dublin-traceroute -h
```

---

## Quick Reference Card

| Command | Purpose |
|---------|---------|
| `-target <host>` | Specify destination (required) |
| `-npaths 8` | Test 8 parallel flows (detect load balancing) |
| `-max-ttl 15` | Limit to 15 hops (faster) |
| `-output-json file.json` | Save results for comparison |
| `-help-routing` | Understand forward vs return paths |
| `-tips` | Learn about route comparison |
| `-list-devices` | Show network adapters |

**Remember:** This shows the forward path only. Return path is usually different!
