# Final validation: real UDP over Docker bridge + tc/netem (no loopback).
param(
    [int]$Connections = 10,
    [string]$Duration = "3s"
)

$ErrorActionPreference = "Stop"
$Bench = Split-Path $PSScriptRoot -Parent
Push-Location $Bench

Write-Host "╔═══════════════════════════════════════════════════════════════╗"
Write-Host "║  QCP verify-docker — real network, all scenarios             ║"
Write-Host "╚═══════════════════════════════════════════════════════════════╝"

docker compose build
docker compose up -d server
Start-Sleep -Seconds 2

$failed = $false
foreach ($scenario in @("lan", "wifi", "4g", "3g", "congested", "extreme")) {
    Write-Host "`n── scenario: $scenario ──"
    docker compose run --rm -e "SCENARIO=$scenario" client `
        -verify-net -server "10.10.0.10:9000" -scenario $scenario `
        -duration $Duration -connections $Connections
    if ($LASTEXITCODE -ne 0) { $failed = $true }
}

docker compose down
Pop-Location
if ($failed) { exit 1 }
Write-Host "`n✓ ALL DOCKER NETWORK SCENARIOS PASS"
