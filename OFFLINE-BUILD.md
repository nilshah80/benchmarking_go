# Offline Build & Deployment Guide

This guide explains how to build and run `benchmarking_go` in environments without internet access or where Go module downloads are blocked.

## Table of Contents

- [Quick Start](#quick-start)
- [Option 1: Pre-built Binary](#option-1-pre-built-binary-easiest)
- [Option 2: Build with Vendored Dependencies](#option-2-build-with-vendored-dependencies)
- [Option 3: Docker Image](#option-3-docker-image)
- [Option 4: Corporate Proxy](#option-4-corporate-proxy)
- [Option 5: Module Cache Transfer](#option-5-module-cache-transfer)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

**Fastest option - just use the pre-built binary:**

```bash
# Windows
.\benchmarking_go.exe -u https://example.com -c 10 -r 100

# Linux/Mac
./benchmarking_go -u https://example.com -c 10 -r 100
```

No Go installation required!

---

## Option 1: Pre-built Binary (Easiest)

The compiled binary is completely standalone and requires no dependencies at runtime.

### For Windows

```powershell
# Just copy and run
.\benchmarking_go.exe -u https://your-api.com -c 10 -d 30
```

### For Linux

```bash
# Build on machine with internet access
GOOS=linux GOARCH=amd64 go build -o benchmarking_go_linux ./cmd

# Copy to target and run
chmod +x benchmarking_go_linux
./benchmarking_go_linux -u https://your-api.com -c 10 -d 30
```

### For macOS

```bash
# For Intel Mac
GOOS=darwin GOARCH=amd64 go build -o benchmarking_go_mac ./cmd

# For Apple Silicon (M1/M2/M3)
GOOS=darwin GOARCH=arm64 go build -o benchmarking_go_mac_arm ./cmd
```

### Cross-compilation Reference

| Target OS | Architecture | Command |
|-----------|--------------|---------|
| Windows | 64-bit | `GOOS=windows GOARCH=amd64 go build -o benchmarking_go.exe ./cmd` |
| Linux | 64-bit | `GOOS=linux GOARCH=amd64 go build -o benchmarking_go_linux ./cmd` |
| Linux | ARM64 | `GOOS=linux GOARCH=arm64 go build -o benchmarking_go_linux_arm ./cmd` |
| macOS | Intel | `GOOS=darwin GOARCH=amd64 go build -o benchmarking_go_mac ./cmd` |
| macOS | Apple Silicon | `GOOS=darwin GOARCH=arm64 go build -o benchmarking_go_mac_arm ./cmd` |

---

## Option 2: Build with Vendored Dependencies

All dependencies are included in the `vendor/` directory. This allows building without any network access.

### Prerequisites

- Go 1.21+ installed (no internet needed after installation)
- This repository with the `vendor/` folder

### Included Dependencies

```
vendor/
├── github.com/
│   └── HdrHistogram/hdrhistogram-go/   # Memory-efficient statistics
└── golang.org/x/
    ├── net/http2/                       # HTTP/2 protocol support
    └── text/                            # Unicode text processing
```

### Build Commands

**Windows (PowerShell):**
```powershell
# Using the build script
.\build-offline.ps1

# Or manually
go build -mod=vendor -o benchmarking_go.exe ./cmd
```

**Linux/macOS:**
```bash
# Using the build script
chmod +x build-offline.sh
./build-offline.sh

# Or manually
go build -mod=vendor -o benchmarking_go ./cmd
```

### Optimized Build (Smaller Binary)

```bash
# Disable CGO and strip debug info
CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" -o benchmarking_go ./cmd
```

This reduces the binary size by ~30%.

---

## Option 3: Docker Image

Build once on a machine with internet, then distribute the image.

### Build the Image

```bash
docker build -t benchmarking_go:latest .
```

### Export for Offline Transfer

```bash
# Save to a tar file
docker save benchmarking_go:latest -o benchmarking_go_docker.tar

# Compress for transfer
gzip benchmarking_go_docker.tar
```

### Import on Target Machine

```bash
# Decompress if needed
gunzip benchmarking_go_docker.tar.gz

# Load the image
docker load -i benchmarking_go_docker.tar

# Verify
docker images | grep benchmarking_go
```

### Run from Docker

```bash
# Basic usage
docker run --network host benchmarking_go:latest -u https://example.com -c 10 -d 30

# With config file
docker run --network host -v $(pwd)/configs:/configs benchmarking_go:latest --config /configs/benchmark.json

# Save results to host
docker run --network host -v $(pwd)/results:/results benchmarking_go:latest \
  -u https://example.com -c 10 -d 30 -o json --output-file /results/benchmark.json
```

---

## Option 4: Corporate Proxy

If your organization has an internal Go module proxy:

### Configure Proxy

```bash
# Set proxy URL
export GOPROXY=https://goproxy.your-company.com

# Or with fallback
export GOPROXY=https://goproxy.your-company.com,direct

# For private modules
export GOPRIVATE=your-company.com/*
```

**Windows:**
```powershell
$env:GOPROXY = "https://goproxy.your-company.com"
```

### Common Proxy Solutions

| Solution | URL |
|----------|-----|
| Athens | https://github.com/gomods/athens |
| Artifactory | https://jfrog.com/artifactory/ |
| Nexus | https://www.sonatype.com/nexus |

---

## Option 5: Module Cache Transfer

Copy the Go module cache from a machine with internet access.

### On Source Machine (with internet)

```bash
# Download all dependencies
go mod download

# Find cache location
go env GOMODCACHE
# Usually: ~/go/pkg/mod (Linux/Mac) or %USERPROFILE%\go\pkg\mod (Windows)

# Create archive
tar -czvf go_mod_cache.tar.gz -C $(go env GOMODCACHE) .
```

### On Target Machine (no internet)

```bash
# Extract to target location
mkdir -p ~/go/pkg/mod
tar -xzvf go_mod_cache.tar.gz -C ~/go/pkg/mod

# Ensure GOMODCACHE points to correct location
export GOMODCACHE=~/go/pkg/mod

# Build
go build ./cmd
```

---

## Troubleshooting

### Error: "vendor directory not found"

```bash
# On a machine with internet access, run:
go mod vendor

# Then commit to version control:
git add vendor/
git commit -m "Add vendored dependencies"
```

### Error: "go.sum mismatch"

```bash
# Regenerate go.sum
go mod tidy
```

### Error: "cannot find module providing package"

Ensure you're using the `-mod=vendor` flag:
```bash
go build -mod=vendor ./cmd
```

### Build is slow

Use parallel compilation:
```bash
go build -mod=vendor -p 4 ./cmd
```

### Binary is too large

Strip debug symbols:
```bash
go build -mod=vendor -ldflags="-s -w" ./cmd
```

Use UPX compression (if available):
```bash
upx --best benchmarking_go
```

---

## Distribution Checklist

When distributing to team members without internet access:

### Option A: Binary Only
- [ ] Pre-built binary for target OS/architecture
- [ ] `README.md` for usage instructions
- [ ] `EXAMPLES.md` for usage examples
- [ ] Sample config files (`configs/examples/`)

### Option B: Full Source
- [ ] All source code (`cmd/`, `pkg/`)
- [ ] `vendor/` directory with all dependencies
- [ ] `go.mod` and `go.sum`
- [ ] Build scripts (`build-offline.ps1`, `build-offline.sh`)
- [ ] Documentation (`README.md`, `EXAMPLES.md`, `OFFLINE-BUILD.md`)

### Option C: Docker
- [ ] Docker image tar file
- [ ] Sample config files
- [ ] Docker run examples

---

## Version Information

| Dependency | Version | Purpose |
|------------|---------|---------|
| HdrHistogram/hdrhistogram-go | v1.2.0 | Memory-efficient latency statistics |
| golang.org/x/net | v0.47.0 | HTTP/2 protocol support |
| golang.org/x/text | v0.31.0 | Unicode text processing |

---

## Support

If you encounter issues with offline builds:

1. Verify Go version: `go version` (requires 1.21+)
2. Verify vendor directory exists: `ls vendor/`
3. Try verbose build: `go build -v -mod=vendor ./cmd`
4. Check for disk space issues
5. Ensure file permissions are correct

For bugs or feature requests, refer to `plan.md` for the development roadmap.

