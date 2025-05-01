# Go Benchmarking Tool

A powerful HTTP benchmarking tool written in Go, similar to bombardier and other load testing tools.

## Features

- Concurrent request execution with configurable number of users
- Support for custom HTTP methods, headers, and request bodies
- Two benchmarking modes: fixed number of requests or duration-based
- Detailed statistics including latency distribution and throughput
- Progress bar with real-time updates
- Graceful shutdown with Ctrl+C
- Configurable request timeout

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/benchmarking_go.git
cd benchmarking_go

# Build the binary
go build -o benchmarking_go
```

## Usage

```
Benchmarking Go HTTP Client
Usage: benchmarking_go [options]

Options:
  -u, --url <url>                  The URL to benchmark
  -c, --concurrent-users <number>  Number of concurrent users
  -r, --requests-per-user <number> Number of requests per user
  -d, --duration <seconds>         Duration in seconds for the benchmark
  -m, --method <GET|POST|PUT|...>  HTTP method to use
  -H, --header <header:value>      Custom header to include in the request
  -b, --body <text>                Request body for POST/PUT
  -t, --content-type <type>        Content-Type of the request body
  --timeout <seconds>              Timeout in seconds for each request (default: 30)
  -h, --help                       Display this help message
```

## Examples

### Basic GET request

```bash
./benchmarking_go -u https://example.com -c 10 -r 100
```

### POST request with JSON body

```bash
./benchmarking_go -u https://api.example.com/data -m POST -b '{"key":"value"}' -H "Authorization:Bearer token" -c 5 -d 30
```

### Running for a specific duration

```bash
./benchmarking_go -u https://example.com -c 20 -d 60
```

### Setting a custom timeout

```bash
./benchmarking_go -u https://example.com -c 10 -r 50 --timeout 10
```

## Output

The tool provides detailed statistics after the benchmark completes:

- Request rate (requests per second)
- Latency statistics (average, standard deviation, maximum)
- Latency distribution (50th, 75th, 90th, and 99th percentiles)
- HTTP status code summary
- Error details if any occurred
- Throughput in MB/s

## Comparison with Other Tools

This benchmarking tool is designed to be similar to popular tools like bombardier and wrk, but with a focus on detailed statistics and ease of use. It provides a comprehensive view of API performance with minimal setup.
