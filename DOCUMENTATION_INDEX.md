# Dublin Traceroute - Documentation Index

Welcome to Dublin Traceroute for Windows! This guide will help you find the right documentation for your needs.

## 🚀 Getting Started

**New to Dublin Traceroute?** Start here:

1. **[README.md](README.md)** - Installation, requirements, and quick start
2. **[EXAMPLES.md](EXAMPLES.md)** - 20+ practical usage examples
3. **[docs/USER_GUIDE.md](docs/USER_GUIDE.md)** - Comprehensive user manual

## 📚 Core Documentation

### For End Users

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[README.md](README.md)** | Quick start and installation | First time setup |
| **[EXAMPLES.md](EXAMPLES.md)** | Practical usage examples | Learning by example |
| **[MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md)** | Quick reference card for MTR mode | Quick lookup while using MTR |
| **[docs/USER_GUIDE.md](docs/USER_GUIDE.md)** | Complete user manual | Deep dive into features |
| **[docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)** | TCP vs UDP mode comparison | Choosing the right protocol |
| **[docs/MTR_MODE.md](docs/MTR_MODE.md)** | Complete MTR mode guide | Learning continuous probing |

### For Developers

| Document | Purpose | When to Use |
|----------|---------|-------------|
| **[BUILD_AND_TEST.md](BUILD_AND_TEST.md)** | Build instructions and testing | Contributing or building from source |
| **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** | Development guidelines | Contributing code |
| **[docs/STATUS.md](docs/STATUS.md)** | Project status and roadmap | Understanding project state |
| **[RELEASE_NOTES_MTR.md](RELEASE_NOTES_MTR.md)** | MTR mode release notes | Understanding MTR implementation |

## 🎯 Find What You Need

### "I want to..."

#### Basic Usage
- **Run my first traceroute** → [README.md - Quick Start](README.md#quick-start)
- **See usage examples** → [EXAMPLES.md](EXAMPLES.md)
- **Understand all the flags** → Run `dublin-traceroute -help`
- **Save results to JSON** → [EXAMPLES.md - Example 15](EXAMPLES.md)

#### TCP Mode
- **Use TCP instead of UDP** → [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)
- **Bypass firewalls** → [EXAMPLES.md - TCP Examples](EXAMPLES.md)
- **Test HTTPS paths** → [EXAMPLES.md - Example 3b](EXAMPLES.md)

#### MTR Mode (Continuous Probing)
- **Understand MTR mode** → [docs/MTR_MODE.md](docs/MTR_MODE.md)
- **Quick MTR reference** → [MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md)
- **Diagnose packet loss** → [docs/MTR_MODE.md - Interpreting Results](docs/MTR_MODE.md#interpreting-results)
- **Detect return path issues** → [docs/MTR_MODE.md - Return Path Analysis](docs/MTR_MODE.md#return-path-analysis-strategy)

#### Troubleshooting
- **"100% packet loss"** → [docs/MTR_MODE.md - Scenario 1](docs/MTR_MODE.md#scenario-1-everything-shows-100-loss)
- **"High loss but app works"** → [docs/MTR_MODE.md - Scenario 2](docs/MTR_MODE.md#scenario-2-high-loss-but-application-works-fine)
- **Network diagnostics** → [docs/USER_GUIDE.md - Troubleshooting](docs/USER_GUIDE.md)
- **Understanding forward vs return paths** → Run `dublin-traceroute -help-routing`

#### Advanced Usage
- **Detect load balancing** → [EXAMPLES.md - Example 4](EXAMPLES.md)
- **Compare routes over time** → Run `dublin-traceroute -tips`
- **Create baselines** → [docs/MTR_MODE.md - Best Practices](docs/MTR_MODE.md#best-practices)
- **Continuous monitoring** → [docs/MTR_MODE.md - Advanced Techniques](docs/MTR_MODE.md#advanced-techniques)

## 📖 Documentation by Topic

### Network Analysis
- **Forward Path Tracing** - [docs/USER_GUIDE.md](docs/USER_GUIDE.md)
- **Return Path Analysis** - [docs/MTR_MODE.md - Return Path Analysis](docs/MTR_MODE.md#return-path-analysis-strategy)
- **Asymmetric Routing** - Run `dublin-traceroute -help-routing`
- **ICMP Filtering Detection** - [docs/MTR_MODE.md - Return Path Issues](docs/MTR_MODE.md#2-return-path-issues-icmp-filtering)

### Performance & Optimization
- **Performance Tips** - [EXAMPLES.md - Performance Tips](EXAMPLES.md#-performance-tips)
- **Timeout Configuration** - [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)
- **Optimal Settings** - [MTR_QUICK_REFERENCE.md - Recommended Settings](MTR_QUICK_REFERENCE.md#recommended-settings)

### Statistics & Monitoring
- **MTR Mode Overview** - [docs/MTR_MODE.md](docs/MTR_MODE.md)
- **Packet Loss Analysis** - [docs/MTR_MODE.md - Interpreting Results](docs/MTR_MODE.md#interpreting-results)
- **Jitter Detection** - [docs/MTR_MODE.md - High Jitter](docs/MTR_MODE.md#3-high-jitter-network-congestion)
- **Baseline Comparison** - [docs/MTR_MODE.md - Best Practices](docs/MTR_MODE.md#5-save-baselines-for-comparison)

### Protocol Selection
- **TCP vs UDP Comparison** - [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)
- **When to Use TCP** - [docs/TCP_VS_UDP.md - When to Use TCP](docs/TCP_VS_UDP.md)
- **When to Use UDP** - [docs/TCP_VS_UDP.md - When to Use UDP](docs/TCP_VS_UDP.md)
- **Firewall Traversal** - [docs/TCP_VS_UDP.md - Scenarios](docs/TCP_VS_UDP.md)

## 🆘 Quick Help

### Command-Line Help
```powershell
# Show all flags and options
dublin-traceroute -help

# Show version information
dublin-traceroute -version

# Explain forward vs return paths
dublin-traceroute -help-routing

# Tips for route comparison
dublin-traceroute -tips

# List network devices
dublin-traceroute -list-devices
```

### Quick Examples
```powershell
# Basic trace
dublin-traceroute -target google.com

# MTR mode (recommended)
dublin-traceroute -target google.com -count 3 -max-ttl 12

# TCP mode for firewalls
dublin-traceroute -target example.com -tcp -dport 443

# Fast trace with timeout
dublin-traceroute -target 1.1.1.1 -max-ttl 8 -timeout 500
```

## 📊 Documentation Maturity

| Area | Status | Documentation |
|------|--------|---------------|
| **Basic Traceroute** | ✅ Complete | README, EXAMPLES, USER_GUIDE |
| **MTR Mode** | ✅ Complete | MTR_MODE, MTR_QUICK_REFERENCE |
| **TCP Support** | ✅ Complete | TCP_VS_UDP, EXAMPLES |
| **Return Path Analysis** | ✅ Complete | MTR_MODE |
| **Network Diagnostics** | ✅ Complete | MTR_MODE, USER_GUIDE |
| **JSON Output** | ✅ Complete | USER_GUIDE, EXAMPLES |
| **Educational Features** | ✅ Complete | Built-in help flags |
| **Development** | ✅ Complete | BUILD_AND_TEST, DEVELOPMENT |

## 🔗 External Resources

- **Npcap Download**: https://npcap.com/#download
- **Original Dublin Traceroute**: https://github.com/insomniacslk/dublin-traceroute
- **Go Installation**: https://go.dev/dl/

## 📝 Documentation Conventions

Throughout this documentation:
- **PowerShell commands** are shown with `PS>` prompt or code blocks
- **File paths** use Windows backslashes `\` or forward slashes `/`
- **Flags** are shown with hyphen prefix: `-target`, `-count`, etc.
- **Examples** include expected output and interpretation
- **⚠️ Warnings** highlight important caveats
- **💡 Tips** provide helpful suggestions
- **✅ Success** indicates expected good results
- **❌ Problems** highlight issues to investigate

## 🆕 Latest Features

### MTR Mode (v1.1.0)
- **Continuous probing** with multiple probes per hop
- **Per-hop statistics**: Loss%, Min/Avg/Max/StdDev RTT
- **Jitter detection** and congestion analysis
- **Return path inference** through statistical patterns
- **See**: [docs/MTR_MODE.md](docs/MTR_MODE.md)

### TCP Support
- **TCP SYN traceroute** for better firewall traversal
- **Configurable destination port** (80, 443, etc.)
- **Faster timeouts** (1s default vs 3s for UDP)
- **See**: [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md)

## 📧 Getting Help

1. **Check documentation** - Use this index to find relevant docs
2. **Run built-in help** - `dublin-traceroute -help`, `-help-routing`, `-tips`
3. **Search examples** - [EXAMPLES.md](EXAMPLES.md) has 20+ scenarios
4. **Review troubleshooting** - [docs/MTR_MODE.md](docs/MTR_MODE.md#troubleshooting-common-scenarios)
5. **Create GitHub issue** - Include version, OS, command, and full output

## 🎓 Learning Path

### Beginner
1. Read [README.md](README.md) - Installation and basic concepts
2. Try [EXAMPLES.md - Basic Examples](EXAMPLES.md#basic-examples)
3. Run `dublin-traceroute -help-routing` - Understand routing
4. Experiment with different targets

### Intermediate
1. Read [docs/TCP_VS_UDP.md](docs/TCP_VS_UDP.md) - Protocol selection
2. Try [EXAMPLES.md - TCP Examples](EXAMPLES.md)
3. Learn [docs/USER_GUIDE.md](docs/USER_GUIDE.md) - Full feature set
4. Start using `-output-json` for analysis

### Advanced
1. Read [docs/MTR_MODE.md](docs/MTR_MODE.md) - Statistical analysis
2. Use [MTR_QUICK_REFERENCE.md](MTR_QUICK_REFERENCE.md) - Quick reference
3. Practice return path inference
4. Create baselines and monitor changes
5. Integrate with monitoring systems

### Expert
1. Read [RELEASE_NOTES_MTR.md](RELEASE_NOTES_MTR.md) - Implementation details
2. Study [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) - Code structure
3. Build from source [BUILD_AND_TEST.md](BUILD_AND_TEST.md)
4. Contribute improvements

---

**Last Updated**: November 24, 2025  
**Version**: 1.1.0 (MTR Mode Release)  
**Maintainer**: Atlantic Broadband Network Engineering
