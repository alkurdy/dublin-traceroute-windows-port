# TCP vs UDP Traceroute

## Overview

Dublin Traceroute for Windows supports both **UDP** and **TCP** protocols for network path tracing. Each has distinct advantages depending on your network environment and troubleshooting needs.

---

## Quick Comparison

| Feature | UDP (Default) | TCP (`-tcp` flag) |
|---------|---------------|-------------------|
| **Firewall Traversal** | Often blocked | Usually allowed (ports 80/443) |
| **Router Responses** | 30-70% success rate | 50-80% success rate |
| **Real-World Traffic** | Less representative | Mirrors actual web/app traffic |
| **Rate Limiting** | More aggressive | Less aggressive |
| **Use Case** | Standard traceroute | Corporate networks, web services |

---

## When to Use UDP (Default)

```powershell
dublin-traceroute -target example.com
```

### Advantages
- **Standard Protocol**: Traditional traceroute method, universally understood
- **No Connection Overhead**: Stateless, simpler packet structure
- **High Port Detection**: Uses ports 33434+ which clearly indicate traceroute traffic
- **Historical Comparison**: Matches standard `traceroute` command behavior

### Best For
- ✅ Home/consumer ISP connections
- ✅ Open internet paths
- ✅ Academic/research networks
- ✅ When comparing with traditional traceroute tools

### Disadvantages
- ❌ Often blocked or deprioritized by firewalls
- ❌ Many routers rate-limit or drop UDP probe responses
- ❌ Less representative of actual application traffic
- ❌ Higher timeout rates (30-70% is normal)

---

## When to Use TCP (`-tcp`)

```powershell
dublin-traceroute -target example.com -tcp -dport 443
```

### Advantages
- **Better Firewall Traversal**: TCP to ports 80/443 is rarely blocked
- **Higher Success Rate**: More routers respond to TCP SYN packets
- **Real-World Paths**: Tests the actual path your web traffic takes
- **Less Rate Limiting**: Routers treat TCP more permissively
- **Better for Analysis**: More complete hop visibility

### Best For
- ✅ **Corporate networks** with strict firewall policies
- ✅ **Testing web services** (port 443 for HTTPS, port 80 for HTTP)
- ✅ **Troubleshooting connectivity** to specific services
- ✅ **Production environments** where UDP may be blocked
- ✅ **When UDP traceroute times out excessively** (>70% loss)

### Disadvantages
- ❌ May trigger IDS/IPS alerts (looks like port scanning)
- ❌ Some security tools may flag TCP SYN floods
- ❌ Target must have port open (or you'll get RST at target)

---

## Common Scenarios

### Scenario 1: UDP Traceroute Has High Loss
```powershell
# UDP trace - 70% packet loss, incomplete path
dublin-traceroute -target vpn.company.com -npaths 4
# Result: Many timeouts, missing hops

# Switch to TCP - much better visibility
dublin-traceroute -target vpn.company.com -tcp -dport 443 -npaths 4
# Result: More complete path, lower loss
```

**Why?** Corporate firewalls often block UDP traceroute but allow TCP to standard ports.

---

### Scenario 2: Testing Web Application Routing
```powershell
# Trace actual HTTPS path
dublin-traceroute -target api.example.com -tcp -dport 443

# Trace HTTP path (might differ)
dublin-traceroute -target api.example.com -tcp -dport 80
```

**Why?** Some load balancers or CDNs route HTTP and HTTPS differently. TCP traceroute to the actual port shows the real path.

---

### Scenario 3: Comparing Protocols
```powershell
# Baseline with UDP
dublin-traceroute -target google.com -npaths 6 -output-json udp-trace.json

# Compare with TCP
dublin-traceroute -target google.com -tcp -dport 443 -npaths 6 -output-json tcp-trace.json
```

**Why?** Different protocols may take different paths due to QoS policies or routing preferences.

---

### Scenario 4: Behind Restrictive Firewall
```powershell
# UDP likely blocked
dublin-traceroute -target 8.8.8.8
# Times out at first hop

# TCP to common port - works!
dublin-traceroute -target 8.8.8.8 -tcp -dport 443 -max-ttl 15
# Complete path visible
```

**Why?** Enterprise firewalls often only allow TCP to known services.

---

## Technical Details

### UDP Traceroute Mechanics
1. Sends UDP packets with increasing TTL
2. Uses high ports (33434+) as destination
3. Encodes flow ID in source port for multipath detection
4. Listens for ICMP Time Exceeded and ICMP Port Unreachable

### TCP Traceroute Mechanics
1. Sends TCP SYN packets with increasing TTL
2. Uses specified destination port (80, 443, etc.)
3. Encodes flow ID in source port for multipath detection
4. Listens for:
   - ICMP Time Exceeded (from intermediate routers)
   - TCP SYN-ACK or RST (from target if reached)

### Why TCP Gets Better Responses
- **Firewall Rules**: Most firewalls have explicit allow rules for TCP 80/443
- **Router Priority**: TCP packets often get higher priority than UDP
- **Rate Limiting**: ICMP rate limiting is often less aggressive for TCP-triggered responses
- **QoS Policies**: Quality of Service may treat TCP SYN favorably

---

## Performance Considerations

### UDP Performance
- **Faster**: Simpler packet structure
- **Less Overhead**: No connection state
- **Lower Network Impact**: Routers can quickly process/drop

### TCP Performance
- **Slightly Slower**: More complex packet crafting
- **Better Results**: More complete paths despite slight speed penalty
- **Worth the Trade-off**: 10-20% longer runtime for 30-50% more hop visibility

---

## Security Considerations

### UDP Traceroute
- ✅ Less likely to trigger security alerts
- ✅ Clearly identifiable as diagnostic traffic
- ❌ May be blocked by security policy

### TCP Traceroute
- ⚠️ May trigger IDS/IPS port scan alerts
- ⚠️ Rapid SYN packets might be flagged as SYN flood
- ✅ Uses legitimate ports, appears like normal traffic
- 💡 **Tip**: Add delays with lower `-npaths` to avoid triggering alerts

```powershell
# Gentler TCP trace - less likely to trigger IDS
dublin-traceroute -target example.com -tcp -dport 443 -npaths 2 -max-ttl 15
```

---

## Practical Recommendations

### Default Choice: UDP
Use UDP for general internet traceroute unless you encounter problems:
```powershell
dublin-traceroute -target example.com -npaths 4
```

### Switch to TCP When:
1. **High packet loss** (>70% timeouts)
2. **Behind corporate firewall**
3. **Testing specific service** (web, API, etc.)
4. **Need complete path visibility**
5. **UDP appears blocked**

```powershell
dublin-traceroute -target example.com -tcp -dport 443 -npaths 4
```

### Best Practice: Test Both
For comprehensive analysis, run both and compare:
```powershell
# UDP baseline
dublin-traceroute -target service.com -output-json udp-baseline.json

# TCP comparison
dublin-traceroute -target service.com -tcp -dport 443 -output-json tcp-baseline.json
```

---

## Port Selection Guide (TCP Mode)

| Port | Protocol | Use Case |
|------|----------|----------|
| **443** | HTTPS | Best default - rarely blocked, tests web traffic |
| **80** | HTTP | Second choice - some networks block unencrypted |
| **22** | SSH | Test admin access paths |
| **53** | DNS | Compare with UDP DNS (use UDP mode instead) |
| **3389** | RDP | Windows Remote Desktop paths |
| **8080** | HTTP Alt | Alternative web services |

**Most Recommended:** `-tcp -dport 443` (HTTPS)

---

## Troubleshooting Guide

### Problem: UDP Traceroute Shows Many Timeouts
```powershell
# Symptom: 70%+ packet loss with UDP
dublin-traceroute -target example.com
# Result: TTL=1 OK, TTL=2 timeout, TTL=3 timeout, ...

# Solution: Switch to TCP
dublin-traceroute -target example.com -tcp -dport 443
# Result: More hops respond, clearer path
```

### Problem: TCP Traceroute Triggers Security Alerts
```powershell
# Issue: IDS flagging rapid SYN packets

# Solution: Reduce flows and add delays
dublin-traceroute -target example.com -tcp -dport 443 -npaths 2 -max-ttl 15
```

### Problem: Neither Protocol Works Past First Hop
```powershell
# Possible causes:
# 1. Very restrictive firewall
# 2. ICMP responses blocked
# 3. Network doesn't support traceroute

# Try both protocols to different ports
dublin-traceroute -target example.com -tcp -dport 443
dublin-traceroute -target example.com -tcp -dport 80
```

---

## Examples Side-by-Side

### Basic Trace
```powershell
# UDP (traditional)
dublin-traceroute -target google.com

# TCP (better firewall traversal)
dublin-traceroute -target google.com -tcp -dport 443
```

### Load Balancing Detection
```powershell
# UDP
dublin-traceroute -target cdn.example.com -npaths 8

# TCP (may show different load balancing)
dublin-traceroute -target cdn.example.com -tcp -dport 443 -npaths 8
```

### Corporate Environment
```powershell
# UDP - likely many timeouts
dublin-traceroute -target vpn.company.com -npaths 4

# TCP - better visibility
dublin-traceroute -target vpn.company.com -tcp -dport 443 -npaths 4
```

---

## Summary

**Use UDP (default) for:**
- General internet troubleshooting
- Home/consumer networks
- Matching traditional traceroute behavior

**Use TCP (`-tcp -dport 443`) for:**
- Corporate/enterprise networks
- When UDP has high packet loss
- Testing web service paths
- Better hop visibility needed

**Test both when:**
- Diagnosing routing issues
- Creating baseline documentation
- Comparing protocol-specific paths
- Maximizing path discovery

---

## Quick Start

Most users should start with TCP for best results:

```powershell
# Recommended starting command (TCP HTTPS)
dublin-traceroute -target example.com -tcp -dport 443 -max-ttl 15 -npaths 4
```

This provides:
- ✅ Good firewall traversal
- ✅ Reasonable speed (~30-60 seconds)
- ✅ Adequate multipath detection
- ✅ Tests real web traffic path
