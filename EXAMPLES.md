# Benchmarking Go - Usage Examples

This document provides comprehensive examples for using the `benchmarking_go` HTTP benchmarking tool.

## Table of Contents

- [Basic Usage](#basic-usage)
- [Duration-Based Benchmarks](#duration-based-benchmarks)
- [Rate Limiting](#rate-limiting)
- [Ramp-Up Period](#ramp-up-period)
- [Custom Headers & Authentication](#custom-headers--authentication)
- [POST/PUT Requests](#postput-requests)
- [Custom Percentiles](#custom-percentiles)
- [Output Formats](#output-formats)
- [JSON Configuration Files](#json-configuration-files)
- [Advanced Examples](#advanced-examples)
- [Docker Usage](#docker-usage)

---

## Basic Usage

### Simple GET Request

```bash
# Benchmark with 10 concurrent users, 100 requests each (1000 total)
./benchmarking_go -u https://example.com

# Same as above, explicit
./benchmarking_go -u https://example.com -c 10 -r 100
```

### Specify Concurrent Users

```bash
# 50 concurrent users
./benchmarking_go -u https://example.com -c 50 -r 100

# 100 concurrent users, 50 requests each
./benchmarking_go -u https://example.com -c 100 -r 50
```

### View Version and Help

```bash
# Display version
./benchmarking_go -v

# Display help
./benchmarking_go -h
```

---

## Duration-Based Benchmarks

### Run for Specific Duration

```bash
# Run for 30 seconds
./benchmarking_go -u https://example.com -c 20 -d 30

# Run for 5 minutes (300 seconds)
./benchmarking_go -u https://example.com -c 50 -d 300
```

### High Load Test

```bash
# 100 connections for 60 seconds
./benchmarking_go -u https://api.example.com/health -c 100 -d 60
```

---

## Rate Limiting

Control the number of requests per second to simulate realistic traffic.

### Basic Rate Limiting

```bash
# Limit to 100 requests per second
./benchmarking_go -u https://example.com -c 20 -d 60 --rate 100

# Shorthand
./benchmarking_go -u https://example.com -c 20 -d 60 -R 100
```

### Simulate User Traffic Patterns

```bash
# Low traffic: 10 req/s
./benchmarking_go -u https://example.com -c 5 -d 60 --rate 10

# Medium traffic: 100 req/s
./benchmarking_go -u https://example.com -c 20 -d 60 --rate 100

# High traffic: 1000 req/s
./benchmarking_go -u https://example.com -c 100 -d 60 --rate 1000
```

---

## Ramp-Up Period

Gradually increase load to avoid thundering herd effect.

### Basic Ramp-Up

```bash
# Start 50 workers gradually over 10 seconds
./benchmarking_go -u https://example.com -c 50 -d 60 --ramp-up 10
```

### Combined with Rate Limiting

```bash
# Ramp up to 100 workers over 30 seconds, then maintain 500 req/s
./benchmarking_go -u https://example.com -c 100 -d 120 --ramp-up 30 --rate 500
```

### Load Test Pattern

```bash
# Simulate gradual user increase
./benchmarking_go -u https://api.example.com -c 200 -d 300 --ramp-up 60
```

---

## Custom Headers & Authentication

### Single Header

```bash
./benchmarking_go -u https://api.example.com -H "Authorization:Bearer token123"
```

### Multiple Headers

```bash
./benchmarking_go -u https://api.example.com \
  -H "Authorization:Bearer token123" \
  -H "X-API-Key:my-api-key" \
  -H "Accept:application/json"
```

### API Key Authentication

```bash
./benchmarking_go -u https://api.example.com/data \
  -H "X-API-Key:your-api-key" \
  -c 10 -d 30
```

### Basic Authentication (via header)

```bash
# Base64 encoded "user:password"
./benchmarking_go -u https://api.example.com \
  -H "Authorization:Basic dXNlcjpwYXNzd29yZA=="
```

---

## POST/PUT Requests

### POST with JSON Body

```bash
./benchmarking_go -u https://api.example.com/users \
  -m POST \
  -b '{"name":"John","email":"john@example.com"}' \
  -t application/json \
  -c 10 -d 30
```

### PUT Request

```bash
./benchmarking_go -u https://api.example.com/users/123 \
  -m PUT \
  -b '{"name":"John Updated"}' \
  -t application/json \
  -c 5 -r 100
```

### DELETE Request

```bash
./benchmarking_go -u https://api.example.com/items/456 \
  -m DELETE \
  -H "Authorization:Bearer token" \
  -c 5 -r 50
```

### Form Data

```bash
./benchmarking_go -u https://example.com/login \
  -m POST \
  -b "username=user&password=pass" \
  -t application/x-www-form-urlencoded \
  -c 10 -r 100
```

---

## Custom Percentiles

### Standard Percentiles

```bash
# Default: 50, 75, 90, 99
./benchmarking_go -u https://example.com -c 10 -d 30
```

### Custom Percentile List

```bash
# Report p50, p90, p95, p99
./benchmarking_go -u https://example.com -c 10 -d 30 -p 50,90,95,99

# Include p99.9 for high precision
./benchmarking_go -u https://example.com -c 10 -d 30 -p 50,90,95,99

# Minimal (only median and p99)
./benchmarking_go -u https://example.com -c 10 -d 30 -p 50,99
```

---

## Output Formats

### Console Output (Default)

```bash
./benchmarking_go -u https://example.com -c 10 -d 30
```

**Sample Output:**
```
Statistics        Avg      Stdev        Max
  Reqs/sec       1523.45    125.32    1892.00
  Latency       12.35ms    3.21ms    45.67ms
  Latency Distribution
     50%    11.23ms
     75%    14.56ms
     90%    18.90ms
     99%    35.12ms
  HTTP codes:
    1xx - 0, 2xx - 15234, 3xx - 0, 4xx - 0, 5xx - 0
    others - 0
  Throughput:   12.45MB/s
```

### JSON Output

```bash
# To stdout
./benchmarking_go -u https://example.com -c 10 -d 30 -o json

# To file
./benchmarking_go -u https://example.com -c 10 -d 30 -o json --output-file results.json
```

### CSV Output

```bash
# To stdout
./benchmarking_go -u https://example.com -c 10 -d 30 -o csv

# To file for data analysis
./benchmarking_go -u https://example.com -c 10 -d 30 -o csv --output-file results.csv
```

### Quiet Mode

```bash
# Only show final summary line
./benchmarking_go -u https://example.com -c 10 -r 100 -q
```

**Output:**
```
Requests: 1000, Duration: 5.23s, Req/s: 191.20, Avg Latency: 52.30ms, Errors: 0
```

### Verbose Mode

```bash
# Show detailed request/response info
./benchmarking_go -u https://example.com -c 2 -r 5 -V
```

---

## JSON Configuration Files

### Basic Configuration

Create `benchmark.json`:
```json
{
  "name": "Simple API Test",
  "settings": {
    "concurrentUsers": 10,
    "duration": "30s",
    "timeout": "10s"
  },
  "requests": [
    {
      "name": "Homepage",
      "url": "https://example.com/",
      "method": "GET"
    }
  ]
}
```

Run:
```bash
./benchmarking_go --config benchmark.json
```

### Multiple URLs with Weights

Create `multi-url.json`:
```json
{
  "name": "E-commerce Load Test",
  "settings": {
    "concurrentUsers": 50,
    "duration": "5m",
    "rateLimit": 500
  },
  "variables": {
    "baseUrl": "https://api.shop.com"
  },
  "defaultHeaders": {
    "Accept": "application/json"
  },
  "requests": [
    {
      "name": "Browse Products",
      "url": "{{baseUrl}}/products",
      "method": "GET",
      "weight": 50
    },
    {
      "name": "View Product",
      "url": "{{baseUrl}}/products/1",
      "method": "GET",
      "weight": 30
    },
    {
      "name": "Add to Cart",
      "url": "{{baseUrl}}/cart",
      "method": "POST",
      "weight": 15,
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "productId": 1,
        "quantity": 1
      }
    },
    {
      "name": "Health Check",
      "url": "{{baseUrl}}/health",
      "method": "GET",
      "weight": 5
    }
  ]
}
```

### Using Environment Variables

Create `ci-config.json`:
```json
{
  "name": "CI/CD Benchmark",
  "settings": {
    "concurrentUsers": 20,
    "duration": "60s",
    "insecure": true
  },
  "variables": {
    "baseUrl": "{{env \"API_BASE_URL\"}}",
    "apiKey": "{{env \"API_KEY\"}}"
  },
  "defaultHeaders": {
    "X-API-Key": "{{apiKey}}"
  },
  "requests": [
    {
      "url": "{{baseUrl}}/api/health",
      "method": "GET"
    }
  ],
  "output": {
    "format": "json",
    "file": "benchmark-results.json"
  }
}
```

Run:
```bash
export API_BASE_URL="https://staging.example.com"
export API_KEY="your-api-key"
./benchmarking_go --config ci-config.json
```

### CLI Overrides with Config

```bash
# Override concurrent users
./benchmarking_go --config benchmark.json -c 100

# Override duration
./benchmarking_go --config benchmark.json -d 120

# Override output format
./benchmarking_go --config benchmark.json -o json

# Multiple overrides
./benchmarking_go --config benchmark.json -c 50 -d 60 --rate 200
```

---

## Advanced Examples

### Skip TLS Verification

```bash
# For self-signed certificates
./benchmarking_go -u https://localhost:8443 -k

# Or with config
./benchmarking_go --config config.json --insecure
```

### Disable Keep-Alive

```bash
# Test without connection reuse
./benchmarking_go -u https://example.com -c 10 -d 30 --disable-keepalive
```

### Custom Timeout

```bash
# 60 second timeout per request
./benchmarking_go -u https://slow-api.example.com -c 5 -d 60 --timeout 60
```

### CI/CD Pipeline Example

```bash
#!/bin/bash
# ci-benchmark.sh

# Run benchmark and save results
./benchmarking_go \
  -u https://staging.example.com/api/health \
  -c 20 \
  -d 60 \
  --rate 100 \
  -o json \
  --output-file benchmark-results.json

# Check results (example: fail if avg latency > 100ms)
AVG_LATENCY=$(jq -r '.latency.average' benchmark-results.json)
echo "Average latency: $AVG_LATENCY"
```

### Stress Test Pattern

```bash
# Gradually increase load to find breaking point
for users in 10 50 100 200 500; do
  echo "Testing with $users concurrent users..."
  ./benchmarking_go -u https://api.example.com \
    -c $users \
    -d 30 \
    --ramp-up 5 \
    -q
  sleep 5
done
```

### Compare Endpoints

```bash
# Test multiple endpoints separately
for endpoint in "/api/users" "/api/products" "/api/orders"; do
  echo "Testing $endpoint..."
  ./benchmarking_go -u "https://api.example.com$endpoint" \
    -c 10 \
    -d 30 \
    -o json \
    --output-file "results_$(echo $endpoint | tr '/' '_').json"
done
```

---

## Docker Usage

### Basic Docker Run

```bash
docker run --network host benchmarking_go -u https://example.com -c 10 -d 30
```

### With Config File

```bash
docker run --network host \
  -v $(pwd)/configs:/configs \
  benchmarking_go --config /configs/benchmark.json
```

### Save Results

```bash
docker run --network host \
  -v $(pwd)/results:/results \
  benchmarking_go \
    -u https://example.com \
    -c 10 \
    -d 30 \
    -o json \
    --output-file /results/benchmark.json
```

### Environment Variables

```bash
docker run --network host \
  -e API_BASE_URL=https://api.example.com \
  -e API_KEY=your-key \
  -v $(pwd)/configs:/configs \
  benchmarking_go --config /configs/ci-config.json
```

---

## Quick Reference

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-u` | `--url` | Target URL | (required) |
| `-c` | `--concurrent-users` | Number of concurrent connections | 10 |
| `-r` | `--requests-per-user` | Requests per user (fixed mode) | 100 |
| `-d` | `--duration` | Duration in seconds | - |
| `-m` | `--method` | HTTP method | GET |
| `-H` | `--header` | Custom header | - |
| `-b` | `--body` | Request body | - |
| `-t` | `--content-type` | Content-Type header | - |
| `-R` | `--rate` | Rate limit (req/s) | unlimited |
| `-` | `--ramp-up` | Ramp-up time (seconds) | 0 |
| `-k` | `--insecure` | Skip TLS verification | false |
| `-` | `--disable-keepalive` | Disable keep-alive | false |
| `-o` | `--output` | Output format (json/csv) | console |
| `-` | `--output-file` | Output file path | stdout |
| `-p` | `--percentiles` | Custom percentiles | 50,75,90,99 |
| `-q` | `--quiet` | Quiet mode | false |
| `-V` | `--verbose` | Verbose mode | false |
| `-` | `--timeout` | Request timeout (seconds) | 30 |
| `-` | `--config` | JSON config file path | - |
| `-v` | `--version` | Show version | - |
| `-h` | `--help` | Show help | - |

