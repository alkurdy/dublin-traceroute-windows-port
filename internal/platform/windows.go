// +build windows

/* SPDX-License-Identifier: BSD-2-Clause */

//
// This file is based on code from the original dublin-traceroute project:
//   https://github.com/insomniacslk/dublin-traceroute
// Copyright (c) insomniacslk (https://github.com/insomniacslk)

package platform

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	// GetAdaptersAddresses flags
	GAA_FLAG_SKIP_ANYCAST   = 0x0002
	GAA_FLAG_SKIP_MULTICAST = 0x0004
)

var (
	modadvapi32             = windows.NewLazySystemDLL("advapi32.dll")
	procGetTokenInformation = modadvapi32.NewProc("GetTokenInformation")
	modshell32              = windows.NewLazySystemDLL("shell32.dll")
	procIsUserAnAdmin       = modshell32.NewProc("IsUserAnAdmin")
	modiphlpapi             = windows.NewLazySystemDLL("iphlpapi.dll")
	procSendARP             = modiphlpapi.NewProc("SendARP")
)

// IsAdmin checks if the current process is running with administrator privileges
// This is required for raw socket operations on Windows
func IsAdmin() (bool, error) {
	// Method 1: Use Shell32 IsUserAnAdmin (simple check)
	ret, _, _ := procIsUserAnAdmin.Call()
	if ret != 0 {
		return true, nil
	}

	// Method 2: Check token elevation level (more reliable)
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return false, fmt.Errorf("failed to open process token: %w", err)
	}
	defer token.Close()

	var elevation uint32
	var returnedLen uint32
	err = windows.GetTokenInformation(
		token,
		windows.TokenElevation,
		(*byte)(unsafe.Pointer(&elevation)),
		uint32(unsafe.Sizeof(elevation)),
		&returnedLen,
	)
	if err != nil {
		return false, fmt.Errorf("failed to get token information: %w", err)
	}

	return elevation != 0, nil
}

// RequireAdmin checks if running as admin and returns a helpful error if not
func RequireAdmin() error {
	isAdmin, err := IsAdmin()
	if err != nil {
		return fmt.Errorf("failed to check administrator privileges: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("dublin-traceroute requires administrator privileges on Windows\n" +
			"Please run from an elevated command prompt or right-click and 'Run as administrator'")
	}
	return nil
}

// CreateRawSocket creates a raw IP socket for packet crafting
// Requires administrator privileges on Windows
func CreateRawSocket(protocol int) (int, error) {
	// Verify admin privileges first
	if err := RequireAdmin(); err != nil {
		return 0, err
	}

	// Create raw socket using Windows syscalls
	// AF_INET = 2, SOCK_RAW = 3, IPPROTO_RAW = 255
	fd, err := windows.Socket(windows.AF_INET, windows.SOCK_RAW, protocol)
	if err != nil {
		return 0, fmt.Errorf("failed to create raw socket (protocol %d): %w", protocol, err)
	}

	// Set IP_HDRINCL socket option to manually construct IP headers
	err = windows.SetsockoptInt(fd, windows.IPPROTO_IP, windows.IP_HDRINCL, 1)
	if err != nil {
		windows.Close(fd)
		return 0, fmt.Errorf("failed to set IP_HDRINCL: %w", err)
	}

	return int(fd), nil
}

// CreateICMPSocket creates a raw socket specifically for ICMP
func CreateICMPSocket() (int, error) {
	return CreateRawSocket(windows.IPPROTO_ICMP)
}

// CreateUDPSocket creates a raw socket for UDP packet crafting
func CreateUDPSocket() (int, error) {
	return CreateRawSocket(windows.IPPROTO_UDP)
}

// SetSocketTimeout sets the receive timeout on a socket
func SetSocketTimeout(fd int, timeoutMs int) error {
	timeout := int32(timeoutMs)
	err := windows.SetsockoptInt(windows.Handle(fd), windows.SOL_SOCKET, windows.SO_RCVTIMEO, int(timeout))
	if err != nil {
		return fmt.Errorf("failed to set socket timeout: %w", err)
	}
	return nil
}

// SendPacket sends a raw IP packet
func SendPacket(fd int, packet []byte, dest *windows.SockaddrInet4) error {
	err := windows.Sendto(windows.Handle(fd), packet, 0, dest)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}
	return nil
}

// ReceivePacket receives a packet from a raw socket
func ReceivePacket(fd int, buffer []byte) (int, *windows.SockaddrInet4, error) {
	n, from, err := windows.Recvfrom(windows.Handle(fd), buffer, 0)
	if err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			if errno == windows.WSAETIMEDOUT {
				return 0, nil, fmt.Errorf("receive timeout")
			}
		}
		return 0, nil, fmt.Errorf("failed to receive packet: %w", err)
	}

	fromAddr, ok := from.(*windows.SockaddrInet4)
	if !ok {
		return 0, nil, fmt.Errorf("received non-IPv4 address")
	}

	return n, fromAddr, nil
}

// CloseSocket closes a raw socket
func CloseSocket(fd int) error {
	return windows.Close(windows.Handle(fd))
}

// GetLocalIPv4Address retrieves the local IPv4 address for the default route
func GetLocalIPv4Address() (string, error) {
	// Get adapter addresses
	var size uint32
	err := windows.GetAdaptersAddresses(windows.AF_INET, GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST, 0, nil, &size)
	if err != nil && err != windows.ERROR_BUFFER_OVERFLOW {
		return "", fmt.Errorf("failed to get adapter address size: %w", err)
	}

	buf := make([]byte, size)
	adapterAddresses := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	err = windows.GetAdaptersAddresses(windows.AF_INET, GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST, 0, adapterAddresses, &size)
	if err != nil {
		return "", fmt.Errorf("failed to get adapter addresses: %w", err)
	}

	// Find first operational adapter with a unicast address
	for adapter := adapterAddresses; adapter != nil; adapter = adapter.Next {
		if adapter.OperStatus != windows.IfOperStatusUp {
			continue
		}

		for unicast := adapter.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
			addr := unicast.Address.IP()
			if addr.To4() != nil && !addr.IsLoopback() {
				return addr.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no suitable IPv4 address found")
}

// CheckNpcapInstalled verifies that Npcap is installed
func CheckNpcapInstalled() (bool, error) {
	// Check for Npcap DLL in System32
	npcapPath := `C:\Windows\System32\Npcap\wpcap.dll`
	_, err := windows.LoadLibrary(npcapPath)
	if err != nil {
		// Try WinPcap compatibility location
		winpcapPath := `C:\Windows\System32\wpcap.dll`
		_, err2 := windows.LoadLibrary(winpcapPath)
		if err2 != nil {
			return false, nil
		}
	}
	return true, nil
}

// GetNpcapInstallMessage returns a helpful message about installing Npcap
func GetNpcapInstallMessage() string {
	return `Npcap is required but not installed.

Please install Npcap from: https://npcap.com/#download

Installation Instructions:
1. Download the latest Npcap installer
2. Run the installer as Administrator
3. IMPORTANT: Check "Install Npcap in WinPcap API-compatible Mode"
4. Restart your computer after installation

Npcap is the modern replacement for WinPcap and is required for
packet capture on Windows 10/11.`
}

// GetPublicNATIP attempts to discover the external/public NAT WAN IP address
// using fast DNS reflection (OpenDNS / Cloudflare) with a conservative timeout.
// Returns an empty string if offline or unavailable without failing the trace.
func GetPublicNATIP() string {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	// Method 1: Fast DNS reflection via OpenDNS resolver
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 400 * time.Millisecond}
			return d.DialContext(ctx, "udp", "208.67.222.222:53")
		},
	}
	ips, err := r.LookupHost(ctx, "myip.opendns.com")
	if err == nil && len(ips) > 0 {
		if parsed := net.ParseIP(ips[0]); parsed != nil && parsed.To4() != nil {
			return parsed.String()
		}
	}

	// Method 2: Fast DNS reflection via Cloudflare resolver
	r2 := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 400 * time.Millisecond}
			return d.DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}
	txts, err := r2.LookupTXT(ctx, "whoami.cloudflare")
	if err == nil && len(txts) > 0 {
		if parsed := net.ParseIP(txts[0]); parsed != nil && parsed.To4() != nil {
			return parsed.String()
		}
	}

	return ""
}

// ResolveMAC resolves the MAC address for an IPv4 address using the Windows SendARP API
func ResolveMAC(ip net.IP) (net.HardwareAddr, error) {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("IPv4 required for ARP resolution")
	}
	var mac [6]byte
	macLen := uint32(len(mac))
	ipUint := binary.LittleEndian.Uint32(ip4)

	ret, _, err := procSendARP.Call(
		uintptr(ipUint),
		0,
		uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&macLen)),
	)
	if ret != 0 {
		return nil, fmt.Errorf("SendARP failed with code %d: %w", ret, err)
	}
	return net.HardwareAddr(mac[:]), nil
}

// NetworkInfo contains Layer 2 and Layer 3 addressing for packet injection
type NetworkInfo struct {
	LocalIP    net.IP
	LocalMAC   net.HardwareAddr
	GatewayIP  net.IP
	GatewayMAC net.HardwareAddr
}

// GetNetworkInfo resolves local IP/MAC and default gateway IP/MAC
func GetNetworkInfo() (*NetworkInfo, error) {
	var size uint32
	err := windows.GetAdaptersAddresses(windows.AF_INET, GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST, 0, nil, &size)
	if err != nil && err != windows.ERROR_BUFFER_OVERFLOW {
		return nil, fmt.Errorf("failed to get adapter size: %w", err)
	}

	buf := make([]byte, size)
	adapterAddresses := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	err = windows.GetAdaptersAddresses(windows.AF_INET, GAA_FLAG_SKIP_ANYCAST|GAA_FLAG_SKIP_MULTICAST, 0, adapterAddresses, &size)
	if err != nil {
		return nil, fmt.Errorf("failed to get adapters: %w", err)
	}

	for adapter := adapterAddresses; adapter != nil; adapter = adapter.Next {
		if adapter.OperStatus != windows.IfOperStatusUp || adapter.PhysicalAddressLength == 0 {
			continue
		}

		var localIP net.IP
		for unicast := adapter.FirstUnicastAddress; unicast != nil; unicast = unicast.Next {
			addr := unicast.Address.IP()
			if addr.To4() != nil && !addr.IsLoopback() {
				localIP = addr
				break
			}
		}

		if localIP == nil {
			continue
		}

		localMAC := net.HardwareAddr(adapter.PhysicalAddress[:adapter.PhysicalAddressLength])

		var gatewayIP net.IP
		if adapter.FirstGatewayAddress != nil {
			gw := adapter.FirstGatewayAddress.Address.IP()
			if gw.To4() != nil {
				gatewayIP = gw
			}
		}

		var gatewayMAC net.HardwareAddr
		if gatewayIP != nil {
			gwMAC, err := ResolveMAC(gatewayIP)
			if err == nil {
				gatewayMAC = gwMAC
			}
		}

		return &NetworkInfo{
			LocalIP:    localIP,
			LocalMAC:   localMAC,
			GatewayIP:  gatewayIP,
			GatewayMAC: gatewayMAC,
		}, nil
	}

	return nil, fmt.Errorf("no operational network adapter found")
}
