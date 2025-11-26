# Automated Test Script for dublin-traceroute-windows-port
# Run this script in PowerShell as Administrator

$exe = "cmd\dublin-traceroute\dublin-traceroute.exe"

# Check if running as administrator
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "WARNING: Not running as Administrator. Tests requiring admin privileges will be skipped." -ForegroundColor Yellow
    Write-Host "Run this script as Administrator for full test coverage." -ForegroundColor Yellow
    Write-Host ""
}

$tests = @(
    @{ Name = "Basic UDP Traceroute"; Args = "-target google.com"; RequiresAdmin = $true },
    @{ Name = "TCP Traceroute"; Args = "-target google.com -tcp -dport 443"; RequiresAdmin = $true },
    @{ Name = "Custom Ports"; Args = "-target 8.8.8.8 -sport 40000 -dport 53"; RequiresAdmin = $true },
    @{ Name = "TTL Range"; Args = "-target example.com -min-ttl 5 -max-ttl 6"; RequiresAdmin = $true },
    @{ Name = "Multipath Detection"; Args = "-target example.com -npaths 2"; RequiresAdmin = $true },
    @{ Name = "MTR Mode"; Args = "-target google.com -count 2"; RequiresAdmin = $true },
    @{ Name = "Custom Timeout"; Args = "-target google.com -timeout 500"; RequiresAdmin = $true },
    @{ Name = "Output to JSON"; Args = "-target google.com -output-json test_trace.json"; RequiresAdmin = $true },
    @{ Name = "Show Version"; Args = "-version"; RequiresAdmin = $false },
    @{ Name = "Show Help Routing"; Args = "-help-routing"; RequiresAdmin = $false },
    @{ Name = "Show Tips"; Args = "-tips"; RequiresAdmin = $false },
    @{ Name = "Verbose Output"; Args = "-target google.com -verbose"; RequiresAdmin = $true },
    @{ Name = "List Devices"; Args = "-list-devices"; RequiresAdmin = $true },
    @{ Name = "Error: Missing Target"; Args = ""; RequiresAdmin = $false; ExpectedError = "target host is required" },
    @{ Name = "Error: Invalid Port"; Args = "-target google.com -dport 99999"; RequiresAdmin = $false; ExpectedError = "invalid destination port" }
)

$results = @()

foreach ($test in $tests) {
    Write-Host "Running: $($test.Name) ..." -ForegroundColor Cyan
    
    # Skip admin-required tests if not running as admin
    if ($test.RequiresAdmin -and -not $isAdmin) {
        Write-Host "[SKIP] $($test.Name) (requires Administrator privileges)" -ForegroundColor Yellow
        $results += [PSCustomObject]@{
            Test = $test.Name
            Args = $test.Args
            Success = $null  # Null indicates skipped
            Output = "Skipped: requires Administrator privileges"
        }
        continue
    }
    
    $output = & $exe @($test.Args -split ' ') 2>&1
    $exitCode = $LASTEXITCODE
    
    # Check success based on test type
    if ($test.ExpectedError) {
        # For error tests, success if output contains expected error
        $fullOutput = $output -join "`n"
        $success = $fullOutput -match [regex]::Escape($test.ExpectedError)
    } else {
        # For normal tests, success if exit code is 0
        $success = $exitCode -eq 0
    }
    
    $results += [PSCustomObject]@{
        Test = $test.Name
        Args = $test.Args
        Success = $success
        Output = $output -join "`n"
    }
    
    if ($success) {
        Write-Host "[PASS] $($test.Name)" -ForegroundColor Green
    } elseif ($test.ExpectedError) {
        Write-Host "[FAIL] $($test.Name) (expected error not found)" -ForegroundColor Red
    } else {
        Write-Host "[FAIL] $($test.Name)" -ForegroundColor Red
    }
}

# Summary
Write-Host "`n--- Test Summary ---" -ForegroundColor Yellow
foreach ($r in $results) {
    $status = switch ($r.Success) {
        $null { "SKIP" }
        $true { "PASS" }
        default { "FAIL" }
    }
    Write-Host ("{0,-25} {1}" -f $r.Test, $status)
}

# Optional: Show details for failed tests
$failed = $results | Where-Object { $_.Success -eq $false }
if ($failed) {
    Write-Host "`n--- Failed Test Details ---" -ForegroundColor Red
    foreach ($f in $failed) {
        Write-Host "`n[$($f.Test)] Output:" -ForegroundColor Red
        Write-Host $f.Output
    }
}

# Show skipped tests if any
$skipped = $results | Where-Object { $_.Success -eq $null }
if ($skipped) {
    Write-Host "`n--- Skipped Tests ---" -ForegroundColor Yellow
    foreach ($s in $skipped) {
        Write-Host $s.Test
    }
    Write-Host "`nRun this script as Administrator to test these features." -ForegroundColor Yellow
}
