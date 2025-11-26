# Automated Test Script for dublin-traceroute-windows-port
# Run this script in PowerShell as Administrator

$exe = "cmd\dublin-traceroute\dublin-traceroute.exe"
$tests = @(
    @{ Name = "Basic UDP Traceroute"; Args = "-target google.com" },
    @{ Name = "TCP Traceroute"; Args = "-target google.com -tcp -dport 443" },
    @{ Name = "Custom Ports"; Args = "-target 8.8.8.8 -sport 40000 -dport 53" },
    @{ Name = "TTL Range"; Args = "-target example.com -min-ttl 5 -max-ttl 6" },
    @{ Name = "Multipath Detection"; Args = "-target example.com -npaths 2" },
    @{ Name = "MTR Mode"; Args = "-target google.com -count 2" },
    @{ Name = "Custom Timeout"; Args = "-target google.com -timeout 500" },
    @{ Name = "Output to JSON"; Args = "-target google.com -output-json test_trace.json" },
    @{ Name = "Show Version"; Args = "-version" },
    @{ Name = "Show Help Routing"; Args = "-help-routing" },
    @{ Name = "Show Tips"; Args = "-tips" },
    @{ Name = "Verbose Output"; Args = "-target google.com -verbose" },
    @{ Name = "List Devices"; Args = "-list-devices" },
    @{ Name = "Error: Missing Target"; Args = "" },
    @{ Name = "Error: Invalid Port"; Args = "-target google.com -dport 99999" }
)

$results = @()

foreach ($test in $tests) {
    Write-Host "Running: $($test.Name) ..." -ForegroundColor Cyan
    $output = & $exe @($test.Args -split ' ') 2>&1
    $success = $LASTEXITCODE -eq 0
    $results += [PSCustomObject]@{
        Test = $test.Name
        Args = $test.Args
        Success = $success
        Output = $output -join "`n"
    }
    if ($success) {
        Write-Host "[PASS] $($test.Name)" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] $($test.Name)" -ForegroundColor Red
    }
}

# Summary
Write-Host "`n--- Test Summary ---" -ForegroundColor Yellow
foreach ($r in $results) {
    $status = if ($r.Success) { "PASS" } else { "FAIL" }
    Write-Host ("{0,-25} {1}" -f $r.Test, $status)
}

# Optional: Show details for failed tests
$failed = $results | Where-Object { -not $_.Success }
if ($failed) {
    Write-Host "`n--- Failed Test Details ---" -ForegroundColor Red
    foreach ($f in $failed) {
        Write-Host "`n[$($f.Test)] Output:" -ForegroundColor Red
        Write-Host $f.Output
    }
}
