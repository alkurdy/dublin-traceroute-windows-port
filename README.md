# Dublin Traceroute - Windows Native Port

**NAT-aware multipath traceroute tool for Windows**

## Attribution

This project is a native Windows port based on the original [dublin-traceroute](https://github.com/insomniacslk/dublin-traceroute) by [insomniacslk](https://github.com/insomniacslk).

Dublin Traceroute for Windows pre-built binary:

**[Download dublin-traceroute.exe](https://github.com/alkurdy/dublin-traceroute-windows-port/releases/download/v1.0.0-windows/dublin-traceroute.exe)**

Dublin Traceroute is a sophisticated network diagnostic tool that can trace multiple paths through a network, detect NAT traversal, and visualize complex network topologies. This is a native Windows port of the original dublin-traceroute project.

## Features

- **Multipath Detection**: Discover all routes packets take through ECMP load-balanced networks
- **MTR Mode (NEW!)**: Continuous probing with per-hop statistics (packet loss, latency, jitter)
- **TCP & UDP Support**: TCP SYN mode for better firewall traversal, traditional UDP mode
- **Return Path Analysis**: Statistical inference to detect ICMP filtering vs real network issues
- **NAT Detection**: Identify Network Address Translation along the path
- **Windows Native**: No WSL2, Docker, or Linux emulation required
- **JSON Export**: Machine-readable output for integration with other tools
- **Raw Socket Support**: Direct packet crafting for maximum control
- **Educational Features**: Built-in help for understanding routing, asymmetric paths, and diagnostics

## Requirements

- **Windows 10 or 11** (64-bit)
- **Administrator Privileges** (required for raw socket access)
- **Npcap** (WinPcap successor) - [Download](https://npcap.com/#download)
- **Go 1.21+** (for building from source)

## Installation

### Pre-built Binary (Coming Soon)
```powershell
# Download from releases page
# Extract dublin-traceroute.exe
# Run as Administrator
.\dublin-traceroute.exe 8.8.8.8
```

### Build from Source
```powershell
# Clone repository
git clone https://github.com/atlanticbb/dublin-traceroute-windows.git
cd dublin-traceroute-windows

# Download dependencies
go mod download

# Build
go build -o dublin-traceroute.exe ./cmd/dublin-traceroute

# Run (requires Administrator)
.\dublin-traceroute.exe 8.8.8.8
```

## Quick Start


## Quick Start

```powershell
# Quick test (recommended)
.\dublin-traceroute.exe -target google.com -max-ttl 12 -npaths 2
# MTR mode (per-hop stats)
.\dublin-traceroute.exe -target google.com -count 3 -max-ttl 15
```

See [MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md) for more usage patterns and [docs/USER_GUIDE.md](docs/USER_GUIDE.md) for a full manual.


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
