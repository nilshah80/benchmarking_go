# Go Benchmarking Tool

A powerful HTTP benchmarking tool written in Go, similar to bombardier and other load testing tools. This tool is designed for API performance testing with detailed metrics and flexible configuration options.

## Features

- **JSON Configuration**: Define complex benchmarks in JSON files for easy maintenance and version control
- **Multiple URL Support**: Test multiple endpoints in a single run with weighted distribution
- **Concurrent Execution**: Configurable number of concurrent users/connections
- **Flexible Modes**: Fixed number of requests or duration-based benchmarking
- **Custom Requests**: Support for custom HTTP methods, headers, and request bodies
- **Rate Limiting**: Control requests per second (`--rate`)
- **Ramp-Up Period**: Gradually start workers (`--ramp-up`)
- **JSON/CSV Output**: Machine-readable output for CI/CD integration and data analysis
- **Custom Percentiles**: Configure which latency percentiles to report (`-p`)
- **TLS Options**: Skip certificate verification for self-signed certs (`--insecure`)
- **Keep-Alive Control**: Disable HTTP keep-alive connections (`--disable-keepalive`)
- **Quiet/Verbose Modes**: Control output verbosity (`-q`, `-V`)
- **Detailed Statistics**: Latency distribution, percentiles, throughput metrics
- **Progress Bar**: Real-time progress updates
- **Graceful Shutdown**: Clean shutdown with Ctrl+C
- **Docker Support**: Containerized execution

## Installation

### Local Build

```bash
# Clone the repository
git clone https://github.com/yourusername/benchmarking_go.git
cd benchmarking_go

# Build the binary
go build -o benchmarking_go ./cmd/

# Or on Windows
go build -o benchmarking_go.exe ./cmd/
```

### Offline Build (No Internet Required)

If Go module downloads are blocked in your environment:

```bash
# Build using vendored dependencies
go build -mod=vendor -o benchmarking_go ./cmd/

# Or use the build scripts
./build-offline.sh      # Linux/Mac
.\build-offline.ps1     # Windows
```

📖 See **[OFFLINE-BUILD.md](OFFLINE-BUILD.md)** for complete offline deployment options.

### Docker Build

```bash
# Build the Docker image
docker build -t benchmarking_go .

# Run the benchmarking tool with Docker
docker run --network host benchmarking_go -u https://example.com -c 10 -d 5
```

Note: The `--network host` flag allows the container to use the host's network, which is important for accurate benchmarking.

## Usage

```
Benchmarking Go HTTP Client v2.1
Usage: benchmarking_go [options]

Options:
  -u, --url <url>                  The URL to benchmark
  -c, --concurrent-users <number>  Number of concurrent users (default: 10)
  -r, --requests-per-user <number> Number of requests per user (default: 100)
  -d, --duration <seconds>         Duration in seconds for the benchmark
  -m, --method <GET|POST|PUT|...>  HTTP method to use (default: GET)
  -H, --header <header:value>      Custom header to include in the request
  -b, --body <text>                Request body for POST/PUT
  -t, --content-type <type>        Content-Type of the request body
  --timeout <seconds>              Timeout in seconds for each request (default: 30)
  --config <file>                  Path to JSON configuration file
  -o, --output <format>            Output format: json, csv, html, or empty for console
  --output-file <file>             Output file path (default: stdout)
  -k, --insecure                   Skip TLS certificate verification

Rate & Connection Options:
  -R, --rate <number>              Rate limit in requests per second (0 = unlimited)
  --ramp-up <seconds>              Gradually start workers over this duration
  --disable-keepalive              Disable HTTP keep-alive connections

Output Options:
  -q, --quiet                      Quiet mode - only show final summary line
  -V, --verbose                    Verbose mode - show detailed request info
  -p, --percentiles <list>         Custom percentiles (e.g., '50,90,95,99')
  --histogram                      Show ASCII latency histogram in output
  --live                           Show real-time stats during benchmark

Protocol Options:
  --http2                          Enable HTTP/2 protocol

Statistics Options:
  --no-hdr                         Disable HdrHistogram (use legacy in-memory stats)

Other:
  -v, --version                    Display version
  -h, --help                       Display this help message
```

## Examples

### Basic GET Request

```bash
./benchmarking_go -u https://example.com -c 10 -r 100
```

### POST Request with JSON Body

```bash
./benchmarking_go -u https://api.example.com/data -m POST -b '{"key":"value"}' -H "Authorization:Bearer token" -c 5 -d 30
```

### Duration-Based Benchmark

```bash
./benchmarking_go -u https://example.com -c 20 -d 60
```

### Using JSON Configuration File

```bash
./benchmarking_go --config benchmark.json
```

### JSON Output for CI/CD

```bash
./benchmarking_go -u https://example.com -c 10 -d 30 -o json > results.json
```

### Skip TLS Verification (Self-Signed Certs)

```bash
./benchmarking_go -u https://localhost:8443 -k
```

### Rate Limiting

```bash
# Limit to 100 requests per second
./benchmarking_go -u https://example.com -c 20 -d 60 --rate 100
```

### Ramp-Up Period

```bash
# Gradually start 50 workers over 10 seconds
./benchmarking_go -u https://example.com -c 50 -d 60 --ramp-up 10
```

### Custom Percentiles

```bash
# Report p50, p90, p95, p99 percentiles
./benchmarking_go -u https://example.com -c 10 -d 30 -p 50,90,95,99
```

### CSV Output

```bash
./benchmarking_go -u https://example.com -c 10 -d 30 -o csv > results.csv
```

### Quiet Mode

```bash
# Only show final summary line
./benchmarking_go -u https://example.com -c 10 -r 100 -q
```

### Latency Histogram

```bash
# Show ASCII histogram of latency distribution
./benchmarking_go -u https://example.com -c 10 -d 30 --histogram
```

**Example Output:**
```
Latency Histogram:
  0-1.00ms      |                                        |   0.00% (0)
  250.00ms-500.00ms|████████████████████████████████████████|  66.67% (2)
  500.00ms-1.00s|████████████████████                    |  33.33% (1)
```

### Live Stats Display

```bash
# Show real-time statistics during benchmark
./benchmarking_go -u https://example.com -c 10 -d 30 --live
```

**Example Output:**
```
 66% [=================================] Reqs: 1523 | Rate: 1523.4/s | Avg: 12.3ms | Err: 0
```

### HTTP/2 Protocol

```bash
# Enable HTTP/2 for modern APIs
./benchmarking_go -u https://example.com -c 10 -d 30 --http2
```

### HTML Report

```bash
# Generate a visual HTML report
./benchmarking_go -u https://example.com -c 10 -d 30 -o html --output-file report.html
```

### Using Docker

```bash
# With fixed number of requests
docker run --network host benchmarking_go -u https://example.com -c 10 -r 100

# With duration-based benchmarking
docker run --network host benchmarking_go -u https://example.com -c 10 -d 30

# With JSON config (mount config directory)
docker run --network host -v $(pwd)/configs:/configs benchmarking_go --config /configs/benchmark.json
```

## JSON Configuration

For complex benchmarks, you can use a JSON configuration file. This is especially useful for:
- Testing multiple endpoints with weighted distribution
- Storing request bodies without command-line escaping
- Reusing configurations across environments
- CI/CD pipelines

### Simple Example

```json
{
  "name": "Simple API Benchmark",
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

### Multiple URLs with Weights

```json
{
  "name": "Multi-URL Benchmark",
  "settings": {
    "concurrentUsers": 20,
    "duration": "60s"
  },
  "variables": {
    "baseUrl": "https://api.example.com"
  },
  "defaultHeaders": {
    "Accept": "application/json"
  },
  "requests": [
    {
      "name": "Get Users",
      "url": "{{baseUrl}}/users",
      "method": "GET",
      "weight": 50
    },
    {
      "name": "Get Products",
      "url": "{{baseUrl}}/products",
      "method": "GET",
      "weight": 30
    },
    {
      "name": "Health Check",
      "url": "{{baseUrl}}/health",
      "method": "GET",
      "weight": 20
    }
  ]
}
```

### POST Request with Body

```json
{
  "name": "POST Request Benchmark",
  "settings": {
    "concurrentUsers": 5,
    "requestsPerUser": 100
  },
  "requests": [
    {
      "name": "Create User",
      "url": "https://api.example.com/users",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer token123"
      },
      "body": {
        "username": "testuser",
        "email": "test@example.com"
      }
    }
  ]
}
```

### Using Environment Variables

```json
{
  "variables": {
    "baseUrl": "{{env \"API_BASE_URL\"}}",
    "apiKey": "{{env \"API_KEY\"}}"
  },
  "defaultHeaders": {
    "X-API-Key": "{{apiKey}}"
  },
  "requests": [
    {
      "url": "{{baseUrl}}/api/endpoint",
      "method": "GET"
    }
  ]
}
```

### JSON Output Configuration

```json
{
  "name": "CI/CD Benchmark",
  "settings": {
    "concurrentUsers": 20,
    "duration": "60s",
    "insecure": true
  },
  "requests": [
    {
      "url": "https://api.example.com/health",
      "method": "GET"
    }
  ],
  "output": {
    "format": "json",
    "file": "benchmark-results.json"
  }
}
```

## Output Formats

### Console Output (Default)

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

```json
{
  "name": "API Benchmark",
  "timestamp": "2024-01-15T10:30:00Z",
  "duration_seconds": 30.5,
  "total_requests": 15234,
  "success_count": 15234,
  "failure_count": 0,
  "requests_per_second": {
    "average": 1523.45,
    "std_dev": 125.32,
    "max": 1892.00
  },
  "latency": {
    "average": "12.35ms",
    "std_dev": "3.21ms",
    "min": "5.12ms",
    "max": "45.67ms",
    "percentiles": {
      "p50": "11.23ms",
      "p75": "14.56ms",
      "p90": "18.90ms",
      "p99": "35.12ms"
    }
  },
  "http_codes": {
    "1xx": 0,
    "2xx": 15234,
    "3xx": 0,
    "4xx": 0,
    "5xx": 0,
    "other": 0
  },
  "throughput": {
    "total_bytes": 15234000,
    "mb_per_second": 12.45
  }
}
```

## Project Structure

```
benchmarking_go/
├── cmd/
│   ├── main.go                  # Application entry point
│   ├── cli.go                   # CLI flag parsing and configuration
│   └── help.go                  # Help text and examples
├── pkg/
│   ├── config/
│   │   └── config.go            # Configuration loading and parsing
│   ├── benchmark/
│   │   ├── stats.go             # Statistics tracking (with HdrHistogram)
│   │   ├── histogram.go         # Histogram rendering and HdrHistogram wrapper
│   │   ├── runner.go            # Benchmark execution logic
│   │   ├── request.go           # HTTP request processing (HTTP/1.1 & HTTP/2)
│   │   └── selector.go          # Weighted request selector & rate limiter
│   ├── output/
│   │   ├── format.go            # Latency formatting utilities
│   │   ├── console.go           # Console output
│   │   ├── json.go              # JSON output
│   │   ├── csv.go               # CSV output
│   │   └── html.go              # HTML report generation
│   └── progress/
│       └── progress.go          # Progress bar with live stats
├── configs/examples/
│   ├── simple.json              # Simple benchmark example
│   ├── multi-url.json           # Multiple URL example
│   ├── post-request.json        # POST request example
│   └── ci-benchmark.json        # CI/CD configuration example
├── vendor/                      # Vendored dependencies (for offline builds)
│   ├── github.com/HdrHistogram/ # HdrHistogram library
│   └── golang.org/x/            # HTTP/2 and text processing
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── Dockerfile                   # Docker build configuration
├── build-offline.ps1            # Windows offline build script
├── build-offline.sh             # Linux/Mac offline build script
├── README.md                    # This documentation
├── EXAMPLES.md                  # Comprehensive usage examples
├── OFFLINE-BUILD.md             # Offline deployment guide
└── plan.md                      # Feature roadmap
```

## Technical Details

### Error Handling

The tool includes robust error handling:
- Context-related errors are filtered from the output statistics
- Each request uses an independent context with a reasonable timeout
- Multiple cancellation checks throughout request processing ensure clean shutdown

### TLS Configuration

Use `--insecure` or `-k` flag to skip TLS certificate verification. This is useful for:
- Self-signed certificates
- Internal/development servers
- Testing environments

### Docker Support

The included Dockerfile creates a minimal Alpine-based image with:
- Multi-stage build for smaller image size (~15MB)
- CA certificates included for HTTPS support
- Proper entrypoint configuration

## Comparison with Other Tools

| Feature | benchmarking_go | bombardier | wrk | hey |
|---------|-----------------|------------|-----|-----|
| JSON Config | ✅ | ❌ | ❌ | ❌ |
| Multiple URLs | ✅ | ❌ | ❌ | ❌ |
| Weighted Distribution | ✅ | ❌ | ❌ | ❌ |
| JSON Output | ✅ | ✅ | ❌ | ❌ |
| Custom Headers | ✅ | ✅ | ✅ | ✅ |
| Request Body | ✅ | ✅ | ❌ | ✅ |
| Duration Mode | ✅ | ✅ | ✅ | ✅ |
| Docker | ✅ | ✅ | ❌ | ❌ |

## License

MIT License
