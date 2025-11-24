# Dublin Traceroute - Windows Native Port

**NAT-aware multipath traceroute tool for Windows**

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

```powershell
# Quick test (RECOMMENDED - completes in ~30 seconds)
.\dublin-traceroute.exe -target google.com -max-ttl 12 -npaths 2

# MTR mode - continuous probing with statistics (NEW!)
.\dublin-traceroute.exe -target google.com -count 3 -max-ttl 15

# MTR mode with TCP for better firewall traversal
.\dublin-traceroute.exe -target example.com -tcp -dport 443 -count 5 -max-ttl 12

# Full trace (takes 2-5 minutes - probes all 30 hops)
.\dublin-traceroute.exe -target google.com

# For nearby targets (local network, corporate servers)
.\dublin-traceroute.exe -target 192.168.1.1 -max-ttl 8 -npaths 2
```

**💡 Tip:** Lower `-max-ttl` and `-npaths` values make traces much faster.
**🆕 MTR Mode:** Use `-count 3` or higher for per-hop statistics (loss%, avg/min/max RTT, jitter)

## Usage

```
dublin-traceroute [OPTIONS] <target>

Options:
  -n, --npaths <num>      Number of paths to probe (default: 20)
  -t, --min-ttl <ttl>     Minimum TTL (default: 1)
  -T, --max-ttl <ttl>     Maximum TTL (default: 30)
  -d, --delay <ms>        Delay between probes in milliseconds (default: 10)
  -o, --output <file>     Save results to JSON file
  -s, --sport <port>      Source port (default: random)
  -p, --dport <port>      Destination port base (default: 33434)
  --use-sport             Use source port for multipath detection
  --timeout <ms>          Probe timeout in milliseconds (default: 3000)
  -h, --help              Show this help message
  -v, --version           Show version information
```

## Windows-Specific Notes

### Running as Administrator
Dublin Traceroute requires raw socket access, which is only available to administrators on Windows.

**Right-click** the executable and select "Run as administrator", or use:
```powershell
Start-Process .\dublin-traceroute.exe -Verb RunAs -ArgumentList "8.8.8.8"
```

### Installing Npcap
1. Download Npcap from [npcap.com](https://npcap.com/#download)
2. Run the installer
3. **Important**: Check "Install Npcap in WinPcap API-compatible Mode"
4. Restart your computer after installation

### Windows Firewall
Dublin Traceroute sends raw ICMP and UDP packets. Windows Firewall may prompt for permissions. Allow the application for both Private and Public networks.

### Troubleshooting
**"Access denied" or "Socket creation failed"**
- Ensure you're running as Administrator
- Check that Windows Defender hasn't blocked the executable

**"Npcap not found"**
- Install Npcap from the link above
- Ensure "WinPcap API-compatible Mode" was selected during installation

**"No route to host"**
- Check your network connection
- Verify the target IP/hostname is reachable with `ping`

## Output Format

Results are exported in JSON format compatible with the original dublin-traceroute:

```json
{
  "flows": {
    "33434": [
      {
        "hop": 1,
        "rtt_usec": 1234,
        "received": {
          "icmp": {
            "type": 11,
            "code": 0
          },
          "ip": {
            "src": "192.168.1.1"
          }
        }
      }
    ]
  }
}
```

## Development

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for development guidelines.

### Building
```powershell
go build -o dublin-traceroute.exe ./cmd/dublin-traceroute
```

### Testing
```powershell
# Unit tests (no admin required)
go test ./pkg/...

# Integration tests (requires admin)
go test -tags=integration ./...
```

### Code Quality
```powershell
# Format code
go fmt ./...

# Lint
golangci-lint run

# Static analysis
go vet ./...
```

## Project Structure

```
windows-port/
├── cmd/
│   └── dublin-traceroute/    # CLI entry point
├── pkg/
│   ├── capture/               # Packet capture (Npcap integration)
│   ├── probe/                 # UDP/TCP probe modules
│   ├── traceroute/           # Core traceroute logic
│   └── results/              # Results processing
├── internal/
│   └── platform/             # Windows-specific code
├── docs/                     # Documentation
└── test/                     # Integration tests
```

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
