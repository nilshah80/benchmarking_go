#!/bin/bash
# Offline Build Script for benchmarking_go
# This script builds the application using vendored dependencies

echo "=== Benchmarking Go - Offline Build ==="
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "ERROR: Go is not installed or not in PATH"
    echo "Please install Go from: https://golang.org/dl/"
    exit 1
fi

echo "Go version: $(go version)"

# Check if vendor directory exists
if [ ! -d "vendor" ]; then
    echo "ERROR: vendor directory not found!"
    echo "This build requires vendored dependencies."
    echo "Run 'go mod vendor' on a machine with internet access first."
    exit 1
fi

echo "Vendor directory found"

# Determine output name based on OS
OUTPUT="benchmarking_go"
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    OUTPUT="benchmarking_go.exe"
fi

# Build with vendor flag
echo ""
echo "Building $OUTPUT..."

CGO_ENABLED=0 go build -mod=vendor -ldflags="-s -w" -o "$OUTPUT" ./cmd

if [ $? -eq 0 ]; then
    SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
    echo ""
    echo "=== Build Successful ==="
    echo "Output: $OUTPUT"
    echo "Size: $SIZE"
    echo ""
    echo "Run with: ./$OUTPUT -h"
else
    echo "Build failed!"
    exit 1
fi

