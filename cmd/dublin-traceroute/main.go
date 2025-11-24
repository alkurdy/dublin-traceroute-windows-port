/* SPDX-License-Identifier: BSD-2-Clause */

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/atlanticbb/dublin-traceroute-windows/internal/platform"
	"github.com/atlanticbb/dublin-traceroute-windows/pkg/capture"
	"github.com/atlanticbb/dublin-traceroute-windows/pkg/probe"
	"github.com/atlanticbb/dublin-traceroute-windows/pkg/results"
)

const (
	version = "1.0.0-windows"
)

var (
	// Target parameters
	target = flag.String("target", "", "Target host or IP address (required)")
	
	// Protocol parameters
	useTCP = flag.Bool("tcp", false, "Use TCP SYN packets instead of UDP (better firewall traversal)")
	
	// Port parameters
	srcPort = flag.Uint("sport", 33434, "Starting source port")
	dstPort = flag.Uint("dport", 33434, "Destination port (UDP) or target port (TCP: 80, 443, etc.)")
	
	// TTL parameters
	minTTL = flag.Uint("min-ttl", 1, "Minimum TTL")
	maxTTL = flag.Uint("max-ttl", 30, "Maximum TTL")
	
	// Path parameters
	numPaths = flag.Uint("npaths", 4, "Number of paths to probe (parallel flows)")
	probeCount = flag.Uint("count", 1, "Number of probes per hop for MTR-style statistics (1-10, default: 1)")
	timeout = flag.Uint("timeout", 0, "Probe timeout in milliseconds (default: UDP=3000ms, TCP=1000ms)")
	
	// Output parameters
	outputJSON = flag.String("output-json", "", "Save results to JSON file")
	showVersion = flag.Bool("version", false, "Show version information")
	showAnalysis = flag.Bool("analyze", true, "Show detailed network analysis (default: true)")
	showHelp = flag.Bool("help-routing", false, "Explain return path routing and asymmetric paths")
	showTips = flag.Bool("tips", false, "Show tips for comparing routes over time")
	verbose = flag.Bool("verbose", false, "Show verbose output including timeouts")
	
	// Debug parameters
	listDevices = flag.Bool("list-devices", false, "List available network devices and exit")
	device = flag.String("device", "", "Network device to use for capture (auto-detect if not specified)")
)

func printBanner() {
	fmt.Printf("Dublin Traceroute for Windows v%s\n", version)
	fmt.Printf("Go %s on %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("NAT-aware multipath traceroute\n")
	fmt.Println()
}

func printUsage() {
	printBanner()
	fmt.Printf("Usage: %s [options] -target <host>\n\n", os.Args[0])
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  Basic UDP trace:")
	fmt.Println("    dublin-traceroute -target google.com")
	fmt.Println()
	fmt.Println("  TCP trace to port 443 (HTTPS):")
	fmt.Println("    dublin-traceroute -target google.com -tcp -dport 443")
	fmt.Println()
	fmt.Println("  TCP trace to port 80 (HTTP) - better firewall traversal:")
	fmt.Println("    dublin-traceroute -target example.com -tcp -dport 80 -npaths 6")
	fmt.Println()
	fmt.Println("  Detect load balancing with more paths:")
	fmt.Println("    dublin-traceroute -target 8.8.8.8 -npaths 8")
	fmt.Println()
	fmt.Println("  MTR mode - multiple probes per hop for statistics:")
	fmt.Println("    dublin-traceroute -target google.com -count 5 -max-ttl 15")
	fmt.Println()
	fmt.Println("  MTR mode with TCP for return path analysis:")
	fmt.Println("    dublin-traceroute -target example.com -tcp -dport 443 -count 3")
	fmt.Println()
	fmt.Println("  Save for later comparison:")
	fmt.Println("    dublin-traceroute -target example.com -output-json baseline.json")
	fmt.Println()
	fmt.Println("  Quick trace (fewer hops):")
	fmt.Println("    dublin-traceroute -target 192.168.1.1 -max-ttl 10")
	fmt.Println()
	fmt.Println("Help & Education:")
	fmt.Println("  -help-routing     Understand forward vs return paths")
	fmt.Println("  -tips             Tips for comparing routes over time")
	fmt.Println()
	fmt.Println("Requirements:")
	fmt.Println("  - Administrator privileges (required for raw sockets)")
	fmt.Println("  - Npcap installed (https://npcap.com)")
	fmt.Println()
}

func checkPrerequisites() error {
	// Check admin privileges
	isAdmin, err := platform.IsAdmin()
	if err != nil {
		return fmt.Errorf("failed to check administrator privileges: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf(
			"dublin-traceroute requires administrator privileges\n\n" +
			"Please run from an elevated PowerShell or Command Prompt:\n" +
			"1. Right-click PowerShell and select 'Run as administrator'\n" +
			"2. Navigate to the directory containing dublin-traceroute.exe\n" +
			"3. Run the command again\n")
	}

	// Check Npcap installation
	npcapInstalled, err := platform.CheckNpcapInstalled()
	if err != nil {
		return fmt.Errorf("failed to check Npcap installation: %w", err)
	}
	if !npcapInstalled {
		return fmt.Errorf("%s", platform.GetNpcapInstallMessage())
	}

	return nil
}

func validateParameters() error {
	if *target == "" && !*listDevices && !*showVersion {
		return fmt.Errorf("target host is required")
	}

	if *srcPort < 1 || *srcPort > 65535 {
		return fmt.Errorf("invalid source port: %d (must be 1-65535)", *srcPort)
	}

	if *dstPort < 1 || *dstPort > 65535 {
		return fmt.Errorf("invalid destination port: %d (must be 1-65535)", *dstPort)
	}

	if *minTTL < 1 || *minTTL > 255 {
		return fmt.Errorf("invalid min-ttl: %d (must be 1-255)", *minTTL)
	}

	if *maxTTL < 1 || *maxTTL > 255 {
		return fmt.Errorf("invalid max-ttl: %d (must be 1-255)", *maxTTL)
	}

	if *minTTL > *maxTTL {
		return fmt.Errorf("min-ttl (%d) cannot be greater than max-ttl (%d)", *minTTL, *maxTTL)
	}

	if *numPaths < 1 || *numPaths > 256 {
		return fmt.Errorf("invalid npaths: %d (must be 1-256)", *numPaths)
	}
	
	if *probeCount < 1 || *probeCount > 10 {
		return fmt.Errorf("invalid count: %d (must be 1-10)", *probeCount)
	}

	// Check if source port range is valid
	maxSrcPort := *srcPort + *numPaths - 1
	if maxSrcPort > 65535 {
		return fmt.Errorf("source port range overflow: %d-%d exceeds 65535", *srcPort, maxSrcPort)
	}

	return nil
}

func main() {
	// Parse command-line flags
	flag.Parse()

	// Handle version flag
	if *showVersion {
		printBanner()
		os.Exit(0)
	}
	
	// Handle help flags
	if *showHelp {
		printBanner()
		fmt.Println(results.ExplainReturnPath())
		os.Exit(0)
	}
	
	if *showTips {
		printBanner()
		results.PrintComparisonHelp()
		os.Exit(0)
	}

	// Handle list-devices flag
	if *listDevices {
		printBanner()
		fmt.Println("Checking prerequisites...")
		if err := checkPrerequisites(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
		
		if err := capture.PrintDeviceList(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to list devices: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Validate parameters
	if err := validateParameters(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n\n", err)
		printUsage()
		os.Exit(1)
	}

	// Print banner
	printBanner()

	// Check prerequisites
	fmt.Println("Checking prerequisites...")
	if err := checkPrerequisites(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓ Administrator privileges: OK")
	fmt.Println("✓ Npcap installation: OK")
	fmt.Println()

	// Create probe (UDP or TCP)
	fmt.Printf("Initializing probe to %s...\n", *target)
	
	var result *results.TracerouteResult
	
	if *useTCP {
		// Create TCP probe
		prober, err := probe.NewTCPProbe(
			*target,
			uint16(*srcPort),
			uint16(*dstPort),
			uint16(*numPaths),
			uint8(*minTTL),
			uint8(*maxTTL),
			int(*probeCount),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create TCP probe: %v\n", err)
			os.Exit(1)
		}
		defer prober.Close()
		
		// Set custom timeout if specified
		if *timeout > 0 {
			prober.SetTimeout(time.Duration(*timeout) * time.Millisecond)
		}
		
		fmt.Println("✓ Raw socket created")
		fmt.Println("✓ Packet capture initialized")
		fmt.Println()
		
		// Run TCP traceroute
		result, err = prober.Traceroute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Traceroute failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Create UDP probe
		prober, err := probe.NewUDPProbe(
			*target,
			uint16(*srcPort),
			uint16(*dstPort),
			uint16(*numPaths),
			uint8(*minTTL),
			uint8(*maxTTL),
			int(*probeCount),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to create UDP probe: %v\n", err)
			os.Exit(1)
		}
		defer prober.Close()
		
		// Set custom timeout if specified
		if *timeout > 0 {
			prober.SetTimeout(time.Duration(*timeout) * time.Millisecond)
		}
		
		fmt.Println("✓ Raw socket created")
		fmt.Println("✓ Packet capture initialized")
		fmt.Println()
		
		// Run UDP traceroute
		result, err = prober.Traceroute()
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Traceroute failed: %v\n", err)
			os.Exit(1)
		}
	}

	// Print summary or MTR-style output based on probe count
	if *probeCount > 1 {
		// MTR-style statistics table
		result.PrintMTRStyle()
	} else {
		// Traditional summary
		result.PrintSummary()
	}

	// Save to JSON if requested
	if *outputJSON != "" {
		jsonData, err := result.ToJSON()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to convert to JSON: %v\n", err)
		} else {
			err = os.WriteFile(*outputJSON, []byte(jsonData), 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WARNING: Failed to write JSON file: %v\n", err)
			} else {
				fmt.Printf("\nResults saved to: %s\n", *outputJSON)
			}
		}
	}

	os.Exit(0)
}
