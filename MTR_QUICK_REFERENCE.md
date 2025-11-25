
# MTR Mode - Quick Reference Card

This card provides a fast lookup for MTR mode usage in Dublin Traceroute. For full details, see [docs/MTR_MODE.md](docs/MTR_MODE.md).

## Enable MTR Mode

```powershell
dublin-traceroute -target <host> -count 3
```

## Recommended Settings

| Scenario | Command |
|----------|---------|
| **Quick MTR** | `dublin-traceroute -target <host> -count 3 -max-ttl 12 -npaths 2` |
| **TCP MTR** | `dublin-traceroute -target <host> -tcp -dport 443 -count 3 -max-ttl 12` |
| **Deep Analysis** | `dublin-traceroute -target <host> -count 5 -max-ttl 15 -npaths 4` |
| **Internal Network** | `dublin-traceroute -target <host> -count 3 -max-ttl 8 -npaths 2 -timeout 1000` |

## Key Flags

| Flag | Default | Range | Description |
|------|---------|-------|-------------|
| `-count` | 1 | 1-10 | Probes per hop (MTR mode) |
| `-max-ttl` | 30 | 1-255 | Maximum hops to trace |
| `-npaths` | 4 | 1-256 | Parallel flows (multipath) |
| `-timeout` | UDP: 3000<br>TCP: 1000 | - | Milliseconds per probe |
| `-tcp` | false | - | Use TCP SYN instead of UDP |
| `-dport` | 33434 | 1-65535 | TCP: target port (80, 443)<br>UDP: dest port |

## Output Columns (see [docs/MTR_MODE.md](docs/MTR_MODE.md) for full explanation)

| Column | Meaning |
|--------|---------|
| **Loss%** | Packet loss per hop |
| **Min/Avg/Max** | Fastest/average/slowest RTT |
| **StdDev** | Jitter (RTT variability) |

## Troubleshooting Shortcuts

- **High loss everywhere:** Likely ICMP filtering, not a real problem
- **Loss at specific hop:** Real congestion or routing issue
- **High jitter:** Network congestion or variable routing
- **Validate with:** `Test-NetConnection` or application testing

## Performance Tips

- Lower `-max-ttl` and `-npaths` for faster results
- Use TCP mode if UDP is blocked or unreliable
- Save baselines for later comparison

## More Help

- Full MTR Guide: [docs/MTR_MODE.md](docs/MTR_MODE.md)
- Usage Examples: [EXAMPLES.md](EXAMPLES.md)
- TCP vs UDP: [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)
- Release Notes: [RELEASE_NOTES_MTR.md](RELEASE_NOTES_MTR.md)
