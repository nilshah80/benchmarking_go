# Offline Build Script for benchmarking_go
# This script builds the application using vendored dependencies

Write-Host "=== Benchmarking Go - Offline Build ===" -ForegroundColor Cyan
Write-Host ""

# Check if Go is installed
$goVersion = go version 2>$null
if (-not $goVersion) {
    Write-Host "ERROR: Go is not installed or not in PATH" -ForegroundColor Red
    Write-Host "Please install Go from: https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

Write-Host "Go version: $goVersion" -ForegroundColor Green

# Check if vendor directory exists
if (-not (Test-Path "vendor")) {
    Write-Host "ERROR: vendor directory not found!" -ForegroundColor Red
    Write-Host "This build requires vendored dependencies." -ForegroundColor Yellow
    Write-Host "Run 'go mod vendor' on a machine with internet access first." -ForegroundColor Yellow
    exit 1
}

Write-Host "Vendor directory found" -ForegroundColor Green

# Build with vendor flag
Write-Host ""
Write-Host "Building benchmarking_go.exe..." -ForegroundColor Cyan

$env:CGO_ENABLED = "0"
go build -mod=vendor -ldflags="-s -w" -o benchmarking_go.exe ./cmd

if ($LASTEXITCODE -eq 0) {
    $fileInfo = Get-Item benchmarking_go.exe
    Write-Host ""
    Write-Host "=== Build Successful ===" -ForegroundColor Green
    Write-Host "Output: benchmarking_go.exe" -ForegroundColor White
    Write-Host "Size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor White
    Write-Host ""
    Write-Host "Run with: .\benchmarking_go.exe -h" -ForegroundColor Cyan
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

