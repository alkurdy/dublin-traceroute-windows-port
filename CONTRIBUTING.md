# Contributing to Dublin Traceroute for Windows

Thank you for your interest in contributing to Dublin Traceroute for Windows!

## Getting Started

1. **Prerequisites**:
   - Windows 10/11 (64-bit)
   - Go 1.21+ installed
   - Npcap installed with "WinPcap API-compatible mode" enabled ([Download Npcap](https://npcap.com/#download))
   - PowerShell running as Administrator (for live network probes)

2. **Clone Repository**:
   ```powershell
   git clone https://github.com/alkurdy/dublin-traceroute-windows-port.git
   cd dublin-traceroute-windows-port
   ```

3. **Build Binary**:
   ```powershell
   go build -v -o dublin-traceroute.exe ./cmd/dublin-traceroute
   ```

## Running Tests

Before submitting a Pull Request, ensure all tests pass:

- **Run Offline Unit Tests**:
  ```powershell
  go test -v -cover ./...
  ```

- **Run Live Integration Suite** (Requires Administrator):
  ```powershell
  .\test-dublin-traceroute.ps1
  ```

## Submitting Pull Requests

1. Create a feature branch: `git checkout -b feature/my-new-feature`
2. Commit your changes with clear, descriptive commit messages.
3. Add unit tests for any new functionality in the appropriate package.
4. Push your branch and open a Pull Request against the `main` branch.
