# Dublin Traceroute - Windows Native Port

[![CI Build](https://github.com/alkurdy/dublin-traceroute-windows-port/actions/workflows/ci.yml/badge.svg)](https://github.com/alkurdy/dublin-traceroute-windows-port/actions)
[![Release](https://img.shields.io/github/v/release/alkurdy/dublin-traceroute-windows-port?include_prereleases&color=blue)](https://github.com/alkurdy/dublin-traceroute-windows-port/releases/latest)
[![Go Version](https://img.shields.io/github/go-mod/go-version/alkurdy/dublin-traceroute-windows-port)](https://golang.org)
[![License](https://img.shields.io/badge/License-BSD_2--Clause-orange.svg)](LICENSE)

**NAT-aware multipath traceroute tool for Windows (ECMP / MTR)**

---

## Attribution

This project is a native Windows port based on the original [dublin-traceroute](https://github.com/insomniacslk/dublin-traceroute) by [insomniacslk](https://github.com/insomniacslk).

### Download Pre-built Binary

- **Direct Executable Download**: **[Download dublin-traceroute.exe (Latest)](https://github.com/alkurdy/dublin-traceroute-windows-port/raw/main/cmd/dublin-traceroute/dublin-traceroute.exe)**
- **Releases Page**: **[GitHub Releases](https://github.com/alkurdy/dublin-traceroute-windows-port/releases)**

Dublin Traceroute is a sophisticated network diagnostic tool that traces multiple paths through a network, detects NAT traversal, and visualizes complex Equal-Cost Multi-Path (ECMP) load balancing. This native Windows port operates without WSL2, Docker, or Linux emulation.

---

## Key Features

- **Hop 0 Local Host & NAT WAN Egress**: Automatically resolves local machine IPv4 (Hop 0) and discovers the router's public NAT WAN egress IP via fast DNS reflection.
- **RFC 1918 Private Network Awareness**: Automatically recognizes private LAN destinations (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, loopback, and link-local) and suppresses external NAT queries for internal traces.
- **Multipath ECMP Detection**: Modifies UDP source ports across parallel flows to uncover Equal-Cost Multi-Path hashing and routing topology.
- **Flow Path Deduplication**: Consolidates flows traversing identical physical routers, clearly grouping active flows (`[Flows: 0, 1, 2, 3]`) while keeping divergent routes separate.
- **MTR Statistical Probing (`-count`)**: Probes each hop multiple times to compute per-hop packet loss %, min/avg/max latency, and jitter standard deviation.
- **Dual Protocol Support (UDP & TCP)**:
  - **UDP (Default)**: Optimized for residential, Wi-Fi, and NAT environments.
  - **TCP SYN (`-tcp`)**: Injects Layer 2 Ethernet frames via Npcap driver (bypassing Windows Winsock raw TCP restrictions) and captures TCP SYN-ACK / RST completions.
- **Return Path Analysis**: Differentiates router ICMP rate-limiting (token bucket throttling) from genuine network packet loss.
- **Zero Runtime Dependencies**: Single compiled `.exe` using Npcap and Windows APIs.
- **JSON Export (`-output-json`)**: Complete structured telemetry for integration into dashboards and monitoring pipelines.

---

## Protocol Selection: UDP vs TCP

| Feature | UDP (Default - Recommended) | TCP SYN (`-tcp -dport 443`) |
| :--- | :--- | :--- |
| **Home / Wi-Fi NAT Compatibility** | **100% Compatible** (NAT conntrack routes ICMP Time Exceeded back to LAN client) | **Blocked by Home NAT** (Residential firewalls drop ICMP Time Exceeded for non-established TCP sessions) |
| **Corporate / Cloud Environments** | May be dropped by restrictive egress firewalls | **Bypasses firewalls** on ports 80/443 |
| **Packet Crafting Method** | Windows Raw Sockets (`IP_HDRINCL`) | Npcap Layer 2 Ethernet Frame Injection |
| **Best Used For** | Home broadband, residential Wi-Fi, ISP diagnostics, LAN traces | Enterprise networks, cloud VMs, direct public IP routing, testing web servers |

> **Note for Home / Residential Users**: Residential NAT routers (e.g. FiOS, Comcast, Netgear) employ stateful NAT. If you use `-tcp`, intermediate routers return ICMP Time Exceeded, but your home router drops incoming ICMP because no full TCP 3-way handshake was established. **Always use UDP mode when running behind home NAT routers.**

---

## Requirements

- **Windows 10 or 11** (64-bit)
- **Administrator Privileges** (required for packet injection and raw capture)
- **Npcap** - [Download from npcap.com](https://npcap.com/#download) (install with *"WinPcap API-compatible mode"* enabled)
- **Go 1.21+** (only if building from source)

---

## Installation

### Option 1: Direct Download (Pre-built Standalone Binary)
1. Download **[dublin-traceroute.exe](https://github.com/alkurdy/dublin-traceroute-windows-port/raw/main/cmd/dublin-traceroute/dublin-traceroute.exe)**.
2. Install [Npcap](https://npcap.com/#download) with *"Install Npcap in WinPcap API-compatible Mode"* checked.
3. Open PowerShell as **Administrator** and run:
   ```powershell
   .\dublin-traceroute.exe 8.8.8.8
   ```

### Option 2: Build from Source
```powershell
# Clone repository
git clone https://github.com/alkurdy/dublin-traceroute-windows-port.git
cd dublin-traceroute-windows-port

# Download dependencies
go mod download

# Build binary
go build -o dublin-traceroute.exe ./cmd/dublin-traceroute

# Run (requires Administrator)
.\dublin-traceroute.exe 8.8.8.8
```

---

## Example Usage

### 1. Standard Internet Trace (with NAT WAN Egress & Hop 0)
```powershell
dublin-traceroute 1.1.1.1 -max-ttl 15
```
```text
Dublin Traceroute to 1.1.1.1 (1.1.1.1)
Local Host: 192.168.12.210 -> NAT WAN Egress: 100.15.226.237
Using UDP ports 33434-33437, TTL 1-15

================================================================================
Dublin Traceroute Results
================================================================================
Target:     1.1.1.1
Local Host: 192.168.12.210
NAT WAN IP: 100.15.226.237 (Router egress to ISP)
Duration:   1.84s

Discovered 1 unique path(s):

Path 0 (5 hops) [Flows: 0, 1, 2, 3]:
   0: 192.168.12.210 (Local Host)                           0.0ms
   1: 192.168.12.1                                          3.1ms
      ↳ NAT WAN Egress: 100.15.226.237                        ---
   2: lo0-100.WASHDC-VFTTP-377.verizon-gni.net (100.36.56.1) 8.8ms
   3: 140.222.2.14                                         11.2ms
   4: 1.1.1.1                                              11.5ms
```

### 2. Internal LAN / Private Network Trace (No NAT Traversal)
```powershell
dublin-traceroute 192.168.12.1 -max-ttl 2
```
```text
Target:     192.168.12.1 (Internal / Private LAN - No NAT Traversal)
Local Host: 192.168.12.210

Path 0 (1 hops) [Flows: 0, 1, 2, 3]:
   0: 192.168.12.210 (Local Host)                           0.0ms
   1: 192.168.12.1                                          2.6ms
```

### 3. Multi-Probe MTR Statistics Mode
```powershell
dublin-traceroute -target 1.1.1.1 -count 5 -max-ttl 15
```

### 4. TCP SYN Mode (for Cloud VMs & Corporate Firewalls)
```powershell
dublin-traceroute -target 1.1.1.1 -tcp -dport 443 -max-ttl 15
```

---

## Documentation Index

- [USER_GUIDE.md](docs/USER_GUIDE.md) - Complete user reference and flag documentation
- [TCP_VS_UDP.md](docs/TCP_VS_UDP.md) - Deep dive into protocol mechanics, stateful NAT, and Npcap packet injection
- [MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md) - Statistical probing and jitter analysis guide
- [EXAMPLES.md](EXAMPLES.md) - Real-world network diagnostic scenarios
- [BUILD_AND_TEST.md](BUILD_AND_TEST.md) - Building, packaging, and integration testing
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contributor guidelines and workflow
- [DEVELOPMENT.md](docs/DEVELOPMENT.md) - Architecture and Windows kernel/driver internals

---

## License

[BSD-2-Clause](LICENSE) (same as original dublin-traceroute).
