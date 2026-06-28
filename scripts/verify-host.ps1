# Final validation on host LAN NIC (no 127.0.0.1).
param(
    [int]$Connections = 20,
    [string]$Duration = "5s",
    [string]$Bind = "0.0.0.0"
)

$ErrorActionPreference = "Stop"
$Bench = Split-Path $PSScriptRoot -Parent

# Pick first non-loopback IPv4
$NicIP = (
    Get-NetIPAddress -AddressFamily IPv4 |
    Where-Object { $_.IPAddress -notlike "127.*" -and $_.PrefixOrigin -ne "WellKnown" } |
    Select-Object -First 1
).IPAddress

if (-not $NicIP) {
    Write-Error "No LAN IPv4 found. Use scripts/verify-docker.sh instead."
}

Write-Host "╔═══════════════════════════════════════════════════════════════╗"
Write-Host "║  QCP verify-host — real UDP via NIC $NicIP                  ║"
Write-Host "╚═══════════════════════════════════════════════════════════════╝"

Push-Location $Bench
go build -o qcp-bench.exe .

$serverJob = Start-Job -ScriptBlock {
    param($dir, $bind)
    Set-Location $dir
    & .\qcp-bench.exe -mode server -bind $bind
} -ArgumentList $Bench, $Bind

Start-Sleep -Seconds 2
$ServerAddr = "${NicIP}:9000"
$env:QCP_BENCH_SERVER = $ServerAddr

$failed = $false
foreach ($scenario in @("lan", "wifi", "4g", "3g", "congested", "extreme")) {
    Write-Host "`n── scenario: $scenario ──"
    $env:QCP_BENCH_SCENARIO = $scenario
    & .\qcp-bench.exe -verify-net -server $ServerAddr -scenario $scenario -duration $Duration -connections $Connections
    if ($LASTEXITCODE -ne 0) { $failed = $true }
}

Stop-Job $serverJob -ErrorAction SilentlyContinue
Remove-Job $serverJob -Force -ErrorAction SilentlyContinue
Pop-Location

if ($failed) { exit 1 }
Write-Host "`n✓ HOST NIC VERIFY PASS"
