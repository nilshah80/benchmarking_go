# Benchmarking Go - Enhancement Plan

This document outlines planned features and enhancements for the `benchmarking_go` HTTP benchmarking tool.

---

## 🎯 High-Value Features

### 1. Output Format Options (JSON/CSV/HTML)
Export results programmatically for CI/CD integration or historical tracking.

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 10 -d 30 --output json > results.json
./benchmarking_go -u https://api.com -c 10 -d 30 --output csv >> benchmark_history.csv
```

**Priority:** High  
**Complexity:** Medium

---

### 2. Request Rate Limiting
Control requests per second to simulate realistic traffic patterns instead of "as fast as possible".

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 10 -d 60 --rate 100  # 100 req/sec
```

**Priority:** High  
**Complexity:** Medium

---

### 3. Ramp-Up / Warm-Up Period
Gradually increase connections to avoid thundering herd and get realistic cold-start metrics.

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 100 --ramp-up 10s  # Reach 100 users over 10 seconds
```

**Priority:** High  
**Complexity:** Medium

---

### 4. Multiple URL Support (via JSON Config)
Test multiple endpoints in a single run with individual configurations per URL. All URLs and their settings are defined in a JSON configuration file for easier maintenance.

**Usage:**
```bash
./benchmarking_go --config benchmark.json -c 10 -d 60
```

**Example Configuration (`benchmark.json`):**
```json
{
  "requests": [
    {
      "name": "Get Users",
      "url": "https://api.com/users",
      "method": "GET",
      "weight": 50
    },
    {
      "name": "Get Products",
      "url": "https://api.com/products",
      "method": "GET",
      "weight": 30
    },
    {
      "name": "Health Check",
      "url": "https://api.com/health",
      "method": "GET",
      "weight": 20
    }
  ]
}
```

**Note:** The `weight` field determines the distribution of requests (e.g., 50% Users, 30% Products, 20% Health).

**Priority:** High  
**Complexity:** Medium

---

### 5. Request Body Support (via JSON Config)
Support complex request bodies defined in the JSON configuration file. This avoids command-line escaping issues and allows for large payloads.

**Usage:**
```bash
./benchmarking_go --config benchmark.json -c 5 -d 30
```

**Example Configuration (`benchmark.json`):**
```json
{
  "requests": [
    {
      "name": "Create User",
      "url": "https://api.com/users",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "Authorization": "Bearer token123"
      },
      "body": {
        "username": "testuser",
        "email": "test@example.com",
        "role": "admin"
      }
    },
    {
      "name": "Upload Data",
      "url": "https://api.com/upload",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "bodyFile": "./payloads/large_payload.json"
    }
  ]
}
```

**Note:** Use `body` for inline JSON objects or `bodyFile` to reference an external file for large payloads.

**Priority:** High  
**Complexity:** Low

---

## 📊 Enhanced Statistics

### 6. Histogram Output ✅ COMPLETED
ASCII histogram of latency distribution for visual analysis.

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 10 -d 30 --histogram
```

**Example Output:**
```
Latency Histogram:
  0-1.00ms      |                                        |   0.00% (0)
  250.00ms-500.00ms|████████████████████████████████████████|  66.67% (2)
  500.00ms-1.00s|████████████████████                    |  33.33% (1)
```

**Priority:** Medium  
**Complexity:** Low  
**Status:** ✅ Implemented in Phase 4

---

### 7. Real-Time Stats Display ✅ COMPLETED
Live updating statistics during the benchmark (similar to `htop` style).

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 10 -d 30 --live
```

**Example Output:**
```
 66% [=================================] Reqs: 1523 | Rate: 1523.4/s | Avg: 12.3ms | Err: 0
```

**Priority:** Medium  
**Complexity:** Medium  
**Status:** ✅ Implemented in Phase 4

---

### 8. Percentile Configuration
Allow custom percentiles (p95, p99.9, etc.).

**Usage:**
```bash
./benchmarking_go -u https://api.com --percentiles 50,90,95,99,99.9
```

**Priority:** Medium  
**Complexity:** Low

---

### 9. Memory-Efficient Statistics (HdrHistogram) ✅ COMPLETED
Replace the in-memory `[]float64` with [HdrHistogram](https://github.com/HdrHistogram/hdrhistogram-go) for constant memory usage regardless of test duration.

**Usage:**
```bash
# HdrHistogram is enabled by default
./benchmarking_go -u https://api.com -c 10 -d 30

# To disable and use legacy in-memory stats
./benchmarking_go -u https://api.com -c 10 -d 30 --no-hdr
```

**Features:**
- Constant memory usage regardless of test duration
- Accurate percentile calculations
- Efficient histogram bucketing
- Configurable precision (3 significant figures)
- Range: 1 microsecond to 60 seconds

**Priority:** Medium  
**Complexity:** Medium  
**Status:** ✅ Implemented in Phase 4

---

## 🔧 Protocol & Connection Features

### 10. HTTP/2 Support ✅ COMPLETED
Explicit HTTP/2 with multiplexing for modern API testing.

**Usage:**
```bash
./benchmarking_go -u https://api.com --http2
```

**Features:**
- Uses `golang.org/x/net/http2` transport
- Automatic connection multiplexing
- Only works with HTTPS endpoints

**Priority:** High  
**Complexity:** Medium  
**Status:** ✅ Implemented in Phase 4

---

### 11. Keep-Alive Configuration
Control connection reuse behavior.

**Usage:**
```bash
./benchmarking_go -u https://api.com --disable-keepalive
./benchmarking_go -u https://api.com --connections 50  # Connection pool size
```

**Priority:** Medium  
**Complexity:** Low

---

### 12. TLS/Certificate Options
Skip verification or use custom certificates.

**Usage:**
```bash
./benchmarking_go -u https://internal.api --insecure
./benchmarking_go -u https://api.com --cert client.crt --key client.key
```

**Priority:** High  
**Complexity:** Low

---

### 13. Proxy Support
Route traffic through HTTP/SOCKS proxies.

**Usage:**
```bash
./benchmarking_go -u https://api.com --proxy http://proxy:8080
```

**Priority:** Medium  
**Complexity:** Medium

---

## 🔄 Advanced Request Features

### 14. Dynamic Variables / Templates (via JSON Config)
Parameterize requests with random or sequential values using template syntax within the JSON configuration.

**Usage:**
```bash
./benchmarking_go --config benchmark.json -c 10 -d 60
```

**Example Configuration (`benchmark.json`):**
```json
{
  "variables": {
    "baseUrl": "https://api.com",
    "apiVersion": "v2"
  },
  "requests": [
    {
      "name": "Get Random User",
      "url": "{{baseUrl}}/{{apiVersion}}/users/{{randomInt 1 1000}}",
      "method": "GET"
    },
    {
      "name": "Create Sequential Order",
      "url": "{{baseUrl}}/orders",
      "method": "POST",
      "body": {
        "orderId": "{{seq}}",
        "timestamp": "{{timestamp}}",
        "userId": "{{randomUUID}}"
      }
    }
  ]
}
```

**Supported Template Functions:**
| Function | Description | Example |
|----------|-------------|---------|
| `{{seq}}` | Sequential counter (1, 2, 3...) | `"id": "{{seq}}"` |
| `{{randomInt min max}}` | Random integer in range | `{{randomInt 1 1000}}` |
| `{{randomString length}}` | Random alphanumeric string | `{{randomString 16}}` |
| `{{randomUUID}}` | Random UUID v4 | `{{randomUUID}}` |
| `{{timestamp}}` | Current Unix timestamp | `{{timestamp}}` |
| `{{timestampMs}}` | Current timestamp in milliseconds | `{{timestampMs}}` |
| `{{isoDate}}` | Current ISO 8601 date | `{{isoDate}}` |
| `{{env "VAR_NAME"}}` | Environment variable | `{{env "API_KEY"}}` |

**Priority:** Medium  
**Complexity:** High

---

### 15. Request Sequence / Scenarios (via JSON Config)
Define multi-step workflows (login → action → logout) with response extraction and chaining. All scenarios are defined in a JSON configuration file for maintainability.

**Usage:**
```bash
./benchmarking_go --config scenario.json -c 5 -d 60 --scenario-mode
```

**Example Configuration (`scenario.json`):**
```json
{
  "name": "User Login Flow",
  "description": "Simulates user login, data fetch, and logout",
  "baseUrl": "https://api.com",
  "variables": {
    "username": "testuser",
    "password": "testpass123"
  },
  "steps": [
    {
      "name": "Login",
      "url": "{{baseUrl}}/auth/login",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "username": "{{username}}",
        "password": "{{password}}"
      },
      "extract": {
        "authToken": "$.data.access_token",
        "userId": "$.data.user.id",
        "refreshToken": "$.data.refresh_token"
      },
      "validate": {
        "status": 200,
        "jsonPath": {
          "$.success": true
        }
      }
    },
    {
      "name": "Get User Profile",
      "url": "{{baseUrl}}/users/{{userId}}",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{authToken}}"
      },
      "validate": {
        "status": 200
      }
    },
    {
      "name": "Update Settings",
      "url": "{{baseUrl}}/users/{{userId}}/settings",
      "method": "PUT",
      "headers": {
        "Authorization": "Bearer {{authToken}}",
        "Content-Type": "application/json"
      },
      "body": {
        "theme": "dark",
        "notifications": true
      },
      "validate": {
        "status": 200,
        "bodyContains": "success"
      }
    },
    {
      "name": "Logout",
      "url": "{{baseUrl}}/auth/logout",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer {{authToken}}"
      },
      "validate": {
        "status": 200
      }
    }
  ]
}
```

**Extraction Features:**
- **JSONPath extraction**: Extract values from JSON responses using JSONPath syntax
- **Header extraction**: Extract values from response headers
- **Regex extraction**: Extract values using regular expressions

**Priority:** Medium  
**Complexity:** High

---

### 16. Response Validation (via JSON Config)
Verify responses meet expectations beyond just status codes. Validation rules are defined per-request in the JSON configuration.

**Usage:**
```bash
./benchmarking_go --config benchmark.json -c 10 -d 30
```

**Example Configuration (`benchmark.json`):**
```json
{
  "requests": [
    {
      "name": "Get User API",
      "url": "https://api.com/users/1",
      "method": "GET",
      "validate": {
        "status": [200, 201],
        "statusRange": {
          "min": 200,
          "max": 299
        },
        "bodyContains": "success",
        "bodyNotContains": "error",
        "jsonPath": {
          "$.data.id": 1,
          "$.data.active": true,
          "$.meta.count": "> 0"
        },
        "headers": {
          "Content-Type": "application/json"
        },
        "responseTime": {
          "max": "500ms"
        }
      }
    }
  ]
}
```

**Validation Options:**
| Option | Description |
|--------|-------------|
| `status` | Expected status code(s) |
| `statusRange` | Status code range (min/max) |
| `bodyContains` | Response body must contain string |
| `bodyNotContains` | Response body must NOT contain string |
| `jsonPath` | JSONPath assertions on response body |
| `headers` | Expected response headers |
| `responseTime` | Maximum allowed response time |

**Priority:** Medium  
**Complexity:** Medium

---

### 17. Cookie/Session Support
Maintain session state across requests.

**Usage:**
```bash
./benchmarking_go -u https://api.com --enable-cookies
```

**Priority:** Medium  
**Complexity:** Medium

---

## 📈 Reporting & Analysis

### 18. Comparison Mode
Compare results between runs or against baseline.

**Usage:**
```bash
./benchmarking_go -u https://api.com --compare baseline.json
```

**Example Output:**
```
Latency: +15% (regression)
Throughput: -5%
```

**Priority:** Low  
**Complexity:** Medium

---

### 19. Time-Series Data Export
Export metrics over time for Grafana/Prometheus visualization.

**Usage:**
```bash
./benchmarking_go -u https://api.com --prometheus-push http://pushgateway:9091
```

**Priority:** Low  
**Complexity:** High

---

### 20. HTML Report Generation ✅ COMPLETED
Generate a self-contained HTML report with visual presentation.

**Usage:**
```bash
./benchmarking_go -u https://api.com -c 10 -d 30 -o html --output-file report.html
```

**Features:**
- Modern dark theme design
- Summary cards for key metrics
- Latency percentile table
- Visual histogram with bar charts
- HTTP status code breakdown
- Per-request statistics for multi-URL tests
- Error summary
- Configuration overview
- Self-contained (no external dependencies)

**Priority:** Medium  
**Complexity:** High  
**Status:** ✅ Implemented in Phase 4

---

## 🛡️ Reliability Features

### 21. Retry Logic
Automatic retries for transient failures.

**Usage:**
```bash
./benchmarking_go -u https://api.com --retries 3 --retry-delay 100ms
```

**Priority:** Medium  
**Complexity:** Low

---

### 22. Circuit Breaker
Stop test early if error rate exceeds threshold.

**Usage:**
```bash
./benchmarking_go -u https://api.com --max-errors 100 --max-error-rate 0.1
```

**Priority:** Medium  
**Complexity:** Medium

---

### 23. DNS Caching Control
Option to bypass DNS caching for testing load balancer distribution.

**Usage:**
```bash
./benchmarking_go -u https://api.com --no-dns-cache
```

**Priority:** Low  
**Complexity:** Medium

---

## 🖥️ User Experience

### 24. JSON Configuration File Support (Core Feature)
Save and reuse benchmark configurations using a comprehensive JSON configuration file. This is the **central feature** that enables features 4, 5, 14, 15, and 16.

**Usage:**
```bash
# Basic usage with config file
./benchmarking_go --config benchmark.json

# Override config file settings via CLI
./benchmarking_go --config benchmark.json -c 50 -d 120

# Validate config file without running
./benchmarking_go --config benchmark.json --validate-only
```

**Complete JSON Configuration Schema:**
```json
{
  "$schema": "https://benchmarking-go/schema/v1.json",
  "name": "API Benchmark Suite",
  "description": "Production API load test",
  
  "settings": {
    "concurrentUsers": 20,
    "duration": "60s",
    "requestsPerUser": 100,
    "rampUp": "10s",
    "timeout": "30s",
    "rateLimit": 1000,
    "http2": false,
    "insecure": false,
    "keepAlive": true,
    "maxConnections": 100,
    "proxy": null,
    "enableCookies": false
  },
  
  "variables": {
    "baseUrl": "https://api.com",
    "apiKey": "{{env \"API_KEY\"}}",
    "version": "v2"
  },
  
  "defaultHeaders": {
    "User-Agent": "benchmarking_go/1.0",
    "Accept": "application/json",
    "X-API-Key": "{{apiKey}}"
  },
  
  "requests": [
    {
      "name": "Get Users",
      "url": "{{baseUrl}}/{{version}}/users",
      "method": "GET",
      "weight": 40,
      "headers": {
        "Cache-Control": "no-cache"
      },
      "validate": {
        "status": 200,
        "responseTime": {"max": "200ms"}
      }
    },
    {
      "name": "Create User",
      "url": "{{baseUrl}}/{{version}}/users",
      "method": "POST",
      "weight": 30,
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "username": "user_{{randomString 8}}",
        "email": "{{randomString 8}}@test.com"
      },
      "validate": {
        "status": [200, 201]
      }
    },
    {
      "name": "Search Products",
      "url": "{{baseUrl}}/{{version}}/products?q={{randomString 5}}",
      "method": "GET",
      "weight": 30
    }
  ],
  
  "output": {
    "format": "json",
    "file": "results/benchmark_{{timestamp}}.json",
    "htmlReport": "results/report_{{timestamp}}.html"
  },

  "thresholds": {
    "maxErrorRate": 0.01,
    "maxP99Latency": "1s",
    "minRequestsPerSecond": 500
  }
}
```

**Configuration Sections:**

| Section | Description |
|---------|-------------|
| `settings` | Global benchmark parameters (can be overridden via CLI) |
| `variables` | Reusable variables and environment references |
| `defaultHeaders` | Headers applied to all requests |
| `requests` | Array of request definitions with weights |
| `output` | Output format and file paths |
| `thresholds` | Pass/fail criteria for CI/CD integration |

**CLI Override Precedence:**
1. CLI arguments (highest priority)
2. Environment variables
3. JSON config file
4. Default values (lowest priority)

**Priority:** High  
**Complexity:** Medium

---

### 25. Quiet / Verbose Modes
Control output verbosity.

**Usage:**
```bash
./benchmarking_go -u https://api.com -q  # Only final stats
./benchmarking_go -u https://api.com -v  # Debug output including headers
```

**Priority:** Medium  
**Complexity:** Low

---

### 26. Color Output Control
Disable colors for CI/CD log compatibility.

**Usage:**
```bash
./benchmarking_go -u https://api.com --no-color  # For CI logs
```

**Priority:** Low  
**Complexity:** Low

---

## 🏆 Development Roadmap

### Phase 1: JSON Configuration Foundation (High Priority) ✅ COMPLETED
Core infrastructure that enables most other features.

| Feature | Complexity | Status |
|---------|------------|--------|
| **JSON config file support (#24)** | Medium | ✅ Done |
| Multiple URL support via JSON (#4) | Medium | ✅ Done |
| Request body via JSON (#5) | Low | ✅ Done |
| JSON output format (#1) | Medium | ✅ Done |
| Insecure TLS (`--insecure`) (#12) | Low | ✅ Done |

### Phase 2: Usability & Basic Features ✅ COMPLETED
| Feature | Complexity | Status |
|---------|------------|--------|
| Rate limiting (#2) | Medium | ✅ Done |
| Ramp-up period (#3) | Medium | ✅ Done |
| Quiet/Verbose modes (#25) | Low | ✅ Done |
| Custom percentiles (#8) | Low | ✅ Done |
| Keep-alive configuration (#11) | Low | ✅ Done |
| CSV output format (#1) | Medium | ✅ Done |

### Phase 3: Advanced JSON Features
| Feature | Complexity | Status |
|---------|------------|--------|
| Dynamic variables/templates (#14) | High | ⬜ Planned |
| Response validation via JSON (#16) | Medium | ⬜ Planned |
| Request scenarios/sequences (#15) | High | ⬜ Planned |
| Thresholds for CI/CD | Medium | ⬜ Planned |

### Phase 4: Performance & Analysis ✅ COMPLETED
| Feature | Complexity | Status |
|---------|------------|--------|
| HTTP/2 support (#10) | Medium | ✅ COMPLETED |
| HdrHistogram integration (#9) | Medium | ✅ COMPLETED |
| HTML report generation (#20) | High | ✅ COMPLETED |
| Histogram output (#6) | Low | ✅ COMPLETED |
| Real-time stats display (#7) | Medium | ✅ COMPLETED |

### Phase 5: Enterprise Features
| Feature | Complexity | Status |
|---------|------------|--------|
| Prometheus integration (#19) | High | ⬜ Planned |
| Client certificate auth (#12) | Low | ⬜ Planned |
| Proxy support (#13) | Medium | ⬜ Planned |
| Circuit breaker (#22) | Medium | ⬜ Planned |
| Cookie/session support (#17) | Medium | ⬜ Planned |
| Comparison mode (#18) | Medium | ⬜ Planned |

---

## 📁 Example JSON Configuration Files

### Example 1: Simple Multi-URL Benchmark (`simple.json`)
```json
{
  "name": "Simple API Benchmark",
  "settings": {
    "concurrentUsers": 10,
    "duration": "30s"
  },
  "requests": [
    {
      "name": "Homepage",
      "url": "https://example.com/",
      "method": "GET"
    },
    {
      "name": "API Health",
      "url": "https://example.com/api/health",
      "method": "GET"
    }
  ]
}
```

### Example 2: Weighted Distribution Benchmark (`weighted.json`)
```json
{
  "name": "E-commerce API Load Test",
  "settings": {
    "concurrentUsers": 50,
    "duration": "5m",
    "rateLimit": 500
  },
  "variables": {
    "baseUrl": "https://api.shop.com/v1"
  },
  "defaultHeaders": {
    "Accept": "application/json",
    "X-Client-ID": "benchmark-tool"
  },
  "requests": [
    {
      "name": "Browse Products",
      "url": "{{baseUrl}}/products?page={{randomInt 1 100}}",
      "method": "GET",
      "weight": 50
    },
    {
      "name": "View Product Detail",
      "url": "{{baseUrl}}/products/{{randomInt 1 10000}}",
      "method": "GET",
      "weight": 30
    },
    {
      "name": "Search",
      "url": "{{baseUrl}}/search?q={{randomString 5}}",
      "method": "GET",
      "weight": 15
    },
    {
      "name": "Add to Cart",
      "url": "{{baseUrl}}/cart/add",
      "method": "POST",
      "weight": 5,
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "productId": "{{randomInt 1 10000}}",
        "quantity": 1
      }
    }
  ],
  "output": {
    "format": "json",
    "file": "results/ecommerce_{{isoDate}}.json"
  }
}
```

### Example 3: Full Scenario with Authentication (`scenario.json`)
```json
{
  "name": "User Journey Scenario",
  "description": "Complete user flow: login, browse, action, logout",
  "settings": {
    "concurrentUsers": 10,
    "duration": "2m",
    "enableCookies": true
  },
  "variables": {
    "baseUrl": "https://api.example.com",
    "testUser": "loadtest@example.com",
    "testPass": "{{env \"TEST_PASSWORD\"}}"
  },
  "steps": [
    {
      "name": "Login",
      "url": "{{baseUrl}}/auth/login",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "email": "{{testUser}}",
        "password": "{{testPass}}"
      },
      "extract": {
        "token": "$.data.accessToken",
        "userId": "$.data.user.id"
      },
      "validate": {
        "status": 200,
        "jsonPath": {
          "$.success": true
        }
      }
    },
    {
      "name": "Get Dashboard",
      "url": "{{baseUrl}}/users/{{userId}}/dashboard",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{token}}"
      },
      "validate": {
        "status": 200,
        "responseTime": {"max": "500ms"}
      }
    },
    {
      "name": "Create Item",
      "url": "{{baseUrl}}/items",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer {{token}}",
        "Content-Type": "application/json"
      },
      "body": {
        "name": "Test Item {{seq}}",
        "description": "Created by benchmark at {{isoDate}}"
      },
      "extract": {
        "itemId": "$.data.id"
      },
      "validate": {
        "status": [200, 201]
      }
    },
    {
      "name": "Delete Item",
      "url": "{{baseUrl}}/items/{{itemId}}",
      "method": "DELETE",
      "headers": {
        "Authorization": "Bearer {{token}}"
      },
      "validate": {
        "status": [200, 204]
      }
    },
    {
      "name": "Logout",
      "url": "{{baseUrl}}/auth/logout",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer {{token}}"
      }
    }
  ],
  "thresholds": {
    "maxErrorRate": 0.01,
    "maxP99Latency": "2s"
  }
}
```

### Example 4: CI/CD Pipeline Configuration (`ci-benchmark.json`)
```json
{
  "name": "CI/CD Performance Gate",
  "description": "Benchmark to run in CI pipeline with pass/fail thresholds",
  "settings": {
    "concurrentUsers": 20,
    "duration": "1m",
    "timeout": "10s",
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
      "name": "Critical Endpoint",
      "url": "{{baseUrl}}/api/critical",
      "method": "GET",
      "validate": {
        "status": 200,
        "responseTime": {"max": "200ms"},
        "bodyContains": "ok"
      }
    }
  ],
  "output": {
    "format": "json",
    "file": "benchmark-results.json"
  },
  "thresholds": {
    "maxErrorRate": 0.001,
    "maxP99Latency": "500ms",
    "minRequestsPerSecond": 100
  }
}
```

---

## 📝 Implementation Notes

### Dependencies to Consider
- `github.com/HdrHistogram/hdrhistogram-go` - Memory-efficient histogram
- `github.com/fatih/color` - Colored terminal output
- `golang.org/x/net/http2` - HTTP/2 support
- `github.com/PaesslerAG/jsonpath` - JSONPath extraction for response validation
- `github.com/google/uuid` - UUID generation for dynamic variables

### Code Structure Recommendations
Consider refactoring to a multi-file structure as features grow:
```
benchmarking_go/
├── cmd/
│   └── main.go              # Entry point
├── pkg/
│   ├── benchmark/
│   │   ├── runner.go        # Benchmark execution
│   │   ├── stats.go         # Statistics tracking
│   │   └── scenario.go      # Scenario/sequence execution
│   ├── config/
│   │   ├── config.go        # JSON configuration parsing
│   │   ├── schema.go        # Config validation
│   │   └── variables.go     # Variable/template resolution
│   ├── output/
│   │   ├── console.go       # Console output
│   │   ├── json.go          # JSON export
│   │   └── html.go          # HTML reports
│   ├── http/
│   │   └── client.go        # HTTP client wrapper
│   └── validation/
│       ├── validator.go     # Response validation
│       └── jsonpath.go      # JSONPath utilities
├── configs/
│   ├── examples/
│   │   ├── simple.json      # Simple benchmark example
│   │   ├── multi-url.json   # Multiple URL example
│   │   └── scenario.json    # Scenario/workflow example
│   └── schema.json          # JSON schema for validation
├── go.mod
├── go.sum
├── Dockerfile
├── README.md
└── plan.md
```

### JSON Configuration Go Structs

```go
// Config represents the root configuration
type Config struct {
    Schema       string            `json:"$schema,omitempty"`
    Name         string            `json:"name,omitempty"`
    Description  string            `json:"description,omitempty"`
    Settings     Settings          `json:"settings,omitempty"`
    Variables    map[string]string `json:"variables,omitempty"`
    DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
    Requests     []RequestConfig   `json:"requests"`
    Steps        []StepConfig      `json:"steps,omitempty"`      // For scenario mode
    Output       OutputConfig      `json:"output,omitempty"`
    Thresholds   ThresholdConfig   `json:"thresholds,omitempty"`
}

// Settings contains global benchmark settings
type Settings struct {
    ConcurrentUsers  int    `json:"concurrentUsers,omitempty"`
    Duration         string `json:"duration,omitempty"`
    RequestsPerUser  int    `json:"requestsPerUser,omitempty"`
    RampUp           string `json:"rampUp,omitempty"`
    Timeout          string `json:"timeout,omitempty"`
    RateLimit        int    `json:"rateLimit,omitempty"`
    HTTP2            bool   `json:"http2,omitempty"`
    Insecure         bool   `json:"insecure,omitempty"`
    KeepAlive        bool   `json:"keepAlive,omitempty"`
    MaxConnections   int    `json:"maxConnections,omitempty"`
    Proxy            string `json:"proxy,omitempty"`
    EnableCookies    bool   `json:"enableCookies,omitempty"`
}

// RequestConfig represents a single request definition
type RequestConfig struct {
    Name       string            `json:"name"`
    URL        string            `json:"url"`
    Method     string            `json:"method,omitempty"`
    Headers    map[string]string `json:"headers,omitempty"`
    Body       interface{}       `json:"body,omitempty"`       // Can be object or string
    BodyFile   string            `json:"bodyFile,omitempty"`   // Path to external file
    Weight     int               `json:"weight,omitempty"`     // Distribution weight
    Validate   ValidationConfig  `json:"validate,omitempty"`
}

// StepConfig represents a step in a scenario (extends RequestConfig)
type StepConfig struct {
    RequestConfig
    Extract    map[string]string `json:"extract,omitempty"`    // JSONPath extractions
    Delay      string            `json:"delay,omitempty"`      // Delay before next step
}

// ValidationConfig defines response validation rules
type ValidationConfig struct {
    Status          interface{}       `json:"status,omitempty"`          // int or []int
    StatusRange     *StatusRange      `json:"statusRange,omitempty"`
    BodyContains    string            `json:"bodyContains,omitempty"`
    BodyNotContains string            `json:"bodyNotContains,omitempty"`
    JSONPath        map[string]interface{} `json:"jsonPath,omitempty"`
    Headers         map[string]string `json:"headers,omitempty"`
    ResponseTime    *ResponseTimeConfig `json:"responseTime,omitempty"`
}

// OutputConfig defines output settings
type OutputConfig struct {
    Format     string `json:"format,omitempty"`     // json, csv, html
    File       string `json:"file,omitempty"`
    HTMLReport string `json:"htmlReport,omitempty"`
}

// ThresholdConfig defines pass/fail criteria
type ThresholdConfig struct {
    MaxErrorRate        float64 `json:"maxErrorRate,omitempty"`
    MaxP99Latency       string  `json:"maxP99Latency,omitempty"`
    MinRequestsPerSecond float64 `json:"minRequestsPerSecond,omitempty"`
}
```

---

## 🔗 References

- [bombardier](https://github.com/codesenberg/bombardier) - Fast HTTP benchmarking tool
- [wrk](https://github.com/wg/wrk) - Modern HTTP benchmarking tool
- [hey](https://github.com/rakyll/hey) - HTTP load generator
- [k6](https://github.com/grafana/k6) - Modern load testing tool
- [vegeta](https://github.com/tsenart/vegeta) - HTTP load testing tool

