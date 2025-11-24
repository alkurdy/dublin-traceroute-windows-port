# Dublin Traceroute Windows Port - Development Guide

## Architecture Overview

This Windows port follows the original Dublin Traceroute design but replaces Linux-specific networking with Windows-compatible alternatives.

### Key Design Decisions

1. **Raw Sockets via Windows API**: Use `golang.org/x/sys/windows` instead of Linux syscalls
2. **Npcap for Capture**: Use Npcap (modern WinPcap replacement) via `gopacket/pcap`
3. **Admin Requirement**: Windows requires Administrator privileges for raw sockets
4. **IPv4 Only**: Initial port focuses on IPv4 (IPv6 requires different Windows APIs)

### Module Breakdown

```
windows-port/
├── cmd/dublin-traceroute/       # CLI application entry point
│   └── main.go                  # Flag parsing, prerequisite checks, orchestration
├── pkg/
│   ├── capture/                 # Packet capture layer
│   │   └── windows.go           # Npcap-based ICMP capture
│   ├── probe/                   # Probing logic
│   │   └── udp.go               # UDP probe implementation
│   ├── results/                 # Data models and output
│   │   └── results.go           # TracerouteResult, path extraction, JSON export
│   └── traceroute/              # (future) High-level traceroute API
└── internal/
    └── platform/                # Windows-specific abstractions
        └── windows.go           # Raw sockets, admin checks, Npcap detection
```

## Windows-Specific Challenges and Solutions

### Challenge 1: Administrator Privileges
**Problem**: Windows requires admin rights for raw socket operations.

**Solution**: 
- Implemented `IsAdmin()` using two methods:
  1. Shell32's `IsUserAnAdmin()` (simple check)
  2. Token elevation check via `GetTokenInformation()` (more reliable)
- Early validation in `main()` with clear error messages
- Admin check in `CreateRawSocket()` to fail fast

**Code Location**: `internal/platform/windows.go` lines 25-58

### Challenge 2: Raw Socket Creation
**Problem**: Windows has different socket constants and requires `IP_HDRINCL`.

**Solution**:
- Use `windows.Socket()` with `AF_INET`, `SOCK_RAW`, and protocol
- Set `IP_HDRINCL` option to manually construct IP headers
- Separate functions for ICMP and UDP sockets

**Code Location**: `internal/platform/windows.go` lines 72-98

### Challenge 3: Packet Capture
**Problem**: Linux uses AF_PACKET sockets; Windows needs Npcap/WinPcap.

**Solution**:
- Use gopacket's `pcap.OpenLive()` for device capture
- Implement device enumeration via `pcap.FindAllDevs()`
- Auto-detect default device by matching local IP
- Apply BPF filters to isolate ICMP responses

**Code Location**: `pkg/capture/windows.go` lines 89-130

### Challenge 4: ICMP Response Matching
**Problem**: Need to match ICMP responses to specific outgoing probes.

**Solution**:
- For Time Exceeded messages, parse embedded IP header
- Match embedded src/dst IPs to our probe parameters
- Extract hop IP from outer IP header's source address
- Handle different ICMP types (Time Exceeded, Dest Unreachable, Echo Reply)

**Code Location**: `pkg/capture/windows.go` lines 145-195

### Challenge 5: Network Device Names
**Problem**: Windows device names are GUIDs like `\Device\NPF_{GUID}`.

**Solution**:
- Provide `-list-devices` flag to show available devices
- Auto-detect default device using `GetAdaptersAddresses()`
- Match device to local IP for automatic selection

**Code Location**: `internal/platform/windows.go` lines 154-183 and `pkg/capture/windows.go` lines 51-75

## Packet Flow

### Outbound Probe (UDP)
```
1. Craft IP header (TTL, src/dst IPs)
2. Craft UDP header (src port = base + flowID)
3. Add 8-byte payload (0xDEADBEEFCAFEBABE)
4. Serialize with gopacket
5. Send via raw socket (windows.Sendto)
```

**Code**: `pkg/probe/udp.go` lines 95-125

### Inbound Response (ICMP)
```
1. Capture with pcap.Handle (BPF filter: "icmp")
2. Parse ICMP layer (check type/code)
3. If Time Exceeded:
   a. Extract embedded IP header from ICMP payload
   b. Match embedded src/dst to our probe
4. Extract hop IP from outer IP header
5. Calculate RTT (recv_time - sent_time)
6. Reverse DNS lookup for hostname
```

**Code**: `pkg/capture/windows.go` lines 145-195

### Path Reconstruction
```
1. Group flows by TTL level
2. For each flow ID, trace responses across TTLs
3. Identify unique paths (different IPs at same TTL)
4. Build Path objects with ordered hops
```

**Code**: `pkg/results/results.go` lines 57-92

## Dublin Traceroute Algorithm

### Flow ID Encoding
Each probe uses a different source port:
```
src_port = base_port + flow_id
```

Example with `base_port=33434` and `npaths=4`:
- Flow 0: src_port 33434
- Flow 1: src_port 33435
- Flow 2: src_port 33436
- Flow 3: src_port 33437

Routers hash 5-tuple (src_ip, src_port, dst_ip, dst_port, protocol) to select path.
Different src_ports → different hash values → different paths.

### Multipath Detection
If two flows at the same TTL receive responses from different IPs, multiple paths exist.

Example:
```
TTL=5:
  Flow 0 → 10.1.1.1
  Flow 1 → 10.1.1.1
  Flow 2 → 10.2.2.2  <-- Different IP = different path
  Flow 3 → 10.2.2.2
```

Result: 2 paths detected (flows 0-1 on path A, flows 2-3 on path B)

## Error Handling Strategy

### Network Errors
- Timeouts: Recorded as flow error, continue with next probe
- ICMP errors: Captured and stored (type/code) in FlowResult
- Socket errors: Fatal, abort entire trace

### User Errors
- Missing admin rights: Early detection with clear instructions
- Invalid parameters: Flag validation before socket creation
- Missing Npcap: Detection with installation link

### Partial Results
- If some flows timeout, still report successful flows
- Statistics show success rate per trace
- JSON output includes error field per flow

## Testing Strategy

### Unit Testing (Future Enhancement)
```go
// Example test structure
func TestCraftUDPPacket(t *testing.T) {
    probe := &UDPProbe{
        SrcIP:   net.ParseIP("192.168.1.100"),
        Target:  net.ParseIP("8.8.8.8"),
        SrcPort: 33434,
        DstPort: 33434,
    }
    
    packet, err := probe.craftUDPPacket(10, 0)
    require.NoError(t, err)
    require.NotEmpty(t, packet)
    
    // Parse and verify packet structure
    // ...
}
```

### Integration Testing
Run against known targets:
1. **localhost (127.0.0.1)**: Single hop, immediate response
2. **LAN gateway**: 1-2 hops, fast response
3. **Public DNS (8.8.8.8)**: ~10-15 hops, multipath likely
4. **Unreachable IP**: Test timeout handling

### Manual Testing Checklist
- [ ] Compile without errors
- [ ] Run without admin → clear error
- [ ] Run without Npcap → clear error
- [ ] List devices shows adapters
- [ ] Traceroute to localhost succeeds
- [ ] Traceroute to LAN gateway succeeds
- [ ] Traceroute to internet host succeeds
- [ ] Multipath detection (test with 8 flows to Google)
- [ ] JSON output is valid
- [ ] Hostname resolution works

## Performance Optimization

### Current Performance
- ~10ms delay between probes (configurable)
- 3-second timeout per probe (configurable)
- Typical trace to internet host: 5-10 seconds

### Potential Improvements
1. **Parallel Probing**: Send all flows for a TTL simultaneously
2. **Adaptive Timeout**: Reduce timeout after first hop responds
3. **Batch Processing**: Send probes for multiple TTLs before waiting
4. **Response Prediction**: Stop early if pattern emerges

### Memory Usage
- Minimal per probe: ~200 bytes (packet buffer)
- Results structure: ~1KB per hop × flows
- Typical trace: <100KB total

## Security Considerations

### Raw Socket Risks
- Can craft arbitrary packets → requires admin
- Can capture network traffic → requires admin
- Mitigations:
  - Early privilege validation
  - No packet forgery (legitimate traceroute only)
  - BPF filter limits capture to ICMP

### Denial of Service
- Rapid probing could trigger rate limits
- Mitigations:
  - Default 10ms delay between probes
  - Maximum 256 parallel flows
  - Timeout prevents infinite loops

### Privacy
- Reveals network path to target
- Captured packets contain routing information
- No credentials or application data exposed

## Comparison with Linux Version

| Feature | Linux (Original) | Windows (This Port) |
|---------|------------------|---------------------|
| Raw Sockets | AF_PACKET + socket() | windows.Socket() |
| Packet Capture | AF_PACKET (same socket) | Npcap + gopacket |
| Admin Required | CAP_NET_RAW capability | Full Administrator |
| Device Names | eth0, wlan0 | \Device\NPF_{GUID} |
| IPv6 Support | Yes | Not yet |
| TCP Probes | Yes | Not yet |

## Future Work

### IPv6 Support
**Required Changes**:
- Add `CreateICMPv6Socket()` and `CreateUDPv6Socket()`
- Implement `craftUDPv6Packet()` with extension headers
- Handle ICMPv6 Time Exceeded (different type codes)
- Update device detection for IPv6 addresses

**Estimated Effort**: 2-3 days

### TCP Probes
**Required Changes**:
- Implement `TCPProbe` struct
- Craft TCP SYN packets with varying sequence numbers
- Capture TCP RST or SYN-ACK responses
- Handle stateful connection tracking

**Estimated Effort**: 1-2 days

### Real-Time Visualization
**Possible Approaches**:
- Terminal UI with github.com/gizak/termui
- Web UI with embedded HTTP server
- Live graph updates as hops are discovered

**Estimated Effort**: 3-5 days

## Debugging Tips

### Enable Verbose Logging
Add debug output to trace execution:
```go
// In pkg/probe/udp.go
fmt.Printf("DEBUG: Sending probe TTL=%d FlowID=%d\n", ttl, flowID)
```

### Capture PCAP Files
Modify `pkg/capture/windows.go` to dump packets:
```go
pcapWriter, _ := pcapgo.NewWriter(file)
pcapWriter.WriteFileHeader(65535, layers.LinkTypeEthernet)
```

### Check Npcap Service
```powershell
Get-Service npcap
# Should show Status: Running
```

### Verify Raw Socket Creation
```powershell
# Check if app is opening raw sockets
netstat -anob | Select-String "dublin-traceroute"
```

### Test Packet Crafting
Extract packet crafting to separate tool:
```go
// test-packet.go
func main() {
    probe := &UDPProbe{...}
    packet, _ := probe.craftUDPPacket(10, 0)
    
    // Print hex dump
    fmt.Println(hex.Dump(packet))
}
```

## Contributing Guidelines

1. **Code Style**: Follow Go conventions, run `gofmt`
2. **Error Handling**: Always check errors, return meaningful messages
3. **Documentation**: Add comments for exported functions
4. **Testing**: Add tests for new features
5. **Compatibility**: Test on Windows 10 and Windows 11
6. **License**: Maintain BSD-2-Clause headers

## Resources

- **Original Dublin Traceroute**: https://github.com/insomniacslk/dublin-traceroute
- **Go Windows API**: https://pkg.go.dev/golang.org/x/sys/windows
- **gopacket**: https://pkg.go.dev/github.com/google/gopacket
- **Npcap**: https://npcap.com/guide/
- **RFC 792**: ICMP specification
- **RFC 1071**: IP checksum calculation

---

**Status**: Development guide complete
**Maintainer**: Reference AGENT.md for project guidelines
**Last Updated**: 2025-01-22
