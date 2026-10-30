# Dublin Traceroute - Windows Native Port

**NAT-aware multipath traceroute tool for Windows**

## Attribution

This project is a native Windows port based on the original [dublin-traceroute](https://github.com/insomniacslk/dublin-traceroute) by [insomniacslk](https://github.com/insomniacslk).

### Download Pre-built Binary

- **Direct Executable Download**: **[Download dublin-traceroute.exe (Latest)](https://github.com/alkurdy/dublin-traceroute-windows-port/raw/main/cmd/dublin-traceroute/dublin-traceroute.exe)**
- **Releases Page**: **[GitHub Releases](https://github.com/alkurdy/dublin-traceroute-windows-port/releases)**

Dublin Traceroute is a sophisticated network diagnostic tool that can trace multiple paths through a network, detect NAT traversal, and visualize complex network topologies. This is a native Windows port of the original dublin-traceroute project.

## Features

- **Multipath Detection**: Discover all routes packets take through ECMP load-balanced networks
- **MTR Mode**: Continuous probing with per-hop statistics (packet loss %, min/avg/max latency, jitter StdDev)
- **TCP & UDP Support**: TCP SYN mode for stateful firewall traversal, traditional UDP mode
- **Hop 0 & NAT Egress**: Resolves local machine IP (Hop 0) and discovers router public NAT WAN IP
- **Return Path Analysis**: Statistical inference to detect router ICMP rate-limiting vs real network congestion
- **Windows Native**: Pure Windows socket and Npcap packet capture (no WSL2, Docker, or Linux emulation required)
- **JSON Export**: Machine-readable structured output for dashboards and tooling
- **Raw Socket Support**: Direct packet crafting for maximum control

## Requirements

- **Windows 10 or 11** (64-bit)
- **Administrator Privileges** (required for raw socket packet crafting)
- **Npcap** (WinPcap successor) - [Download](https://npcap.com/#download) (install with "WinPcap API-compatible mode" enabled)
- **Go 1.21+** (only if building from source)

## Installation

### Option 1: Direct Download (Pre-built)
1. Download **[dublin-traceroute.exe](https://github.com/alkurdy/dublin-traceroute-windows-port/raw/main/cmd/dublin-traceroute/dublin-traceroute.exe)**
2. Install [Npcap](https://npcap.com/#download) with "WinPcap API-compatible mode" checked.
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

## Quick Start

```powershell
# Quick multipath test
.\dublin-traceroute.exe -target google.com -max-ttl 12 -npaths 2

# MTR statistical mode (per-hop packet loss, latency, jitter)
.\dublin-traceroute.exe -target google.com -count 3 -max-ttl 15

# TCP SYN mode to bypass UDP blocking
.\dublin-traceroute.exe -target google.com -tcp -dport 443
```

See [MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md) for more usage patterns and [docs/USER_GUIDE.md](docs/USER_GUIDE.md) for the full user manual.


## Windows-Specific Notes

- Run as Administrator for raw socket access
- Install [Npcap](https://npcap.com/#download) (WinPcap API-compatible mode)
- Allow through Windows Firewall if prompted

See [docs/USER_GUIDE.md](docs/USER_GUIDE.md) for troubleshooting and advanced setup.


## Output Format

Results are exported in JSON format. See [docs/USER_GUIDE.md](docs/USER_GUIDE.md) for details and examples.


## Development

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for build, test, and code quality instructions.


## Project Structure

See [DOCUMENTATION_INDEX.md](DOCUMENTATION_INDEX.md) for a full documentation map and project structure.


## Credits

Based on the original [dublin-traceroute](https://github.com/insomniacslk/dublin-traceroute) by Andrea Barberio (@insomniacslk).


## License

BSD-2-Clause (same as original dublin-traceroute)


## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.


## See Also

- [Dublin Traceroute Website](https://dublin-traceroute.net)
- [Dublin Traceroute Blog](https://blog.dublin-traceroute.net)
- [Python Bindings](https://github.com/insomniacslk/python-dublin-traceroute)
