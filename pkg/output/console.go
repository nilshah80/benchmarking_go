// Package output handles benchmark result output in various formats
package output

import (
	"fmt"

	"github.com/benchmarking_go/pkg/benchmark"
	"github.com/benchmarking_go/pkg/config"
)

// WriteConsole outputs results to console
func WriteConsole(stats *benchmark.Stats, cfg *config.Config) {
	fmt.Println("\nStatistics        Avg      Stdev        Max")

	fmt.Printf("  Reqs/sec    %10.2f   %8.2f   %9.2f\n",
		stats.RequestsPerSecond,
		stats.RequestRateStdDev(),
		stats.MaxRequestRate())

	avgLatency := FormatLatency(stats.AverageResponseTime())
	stdevLatency := FormatLatency(stats.StandardDeviation())
	maxLatency := FormatLatency(float64(stats.MaxResponseTime()))

	fmt.Printf("  Latency      %8s   %8s    %7s\n", avgLatency, stdevLatency, maxLatency)

	// Use custom percentiles from config
	percentiles := cfg.Settings.Percentiles
	if len(percentiles) == 0 {
		percentiles = []int{50, 75, 90, 99}
	}

	fmt.Println("  Latency Distribution")
	for _, p := range percentiles {
		fmt.Printf("     %d%%    %s\n", p, FormatLatency(float64(stats.GetLatencyPercentile(p))))
	}

	fmt.Println("  HTTP codes:")
	fmt.Printf("    1xx - %d, 2xx - %d, 3xx - %d, 4xx - %d, 5xx - %d\n",
		stats.Http1xxCount, stats.Http2xxCount, stats.Http3xxCount, stats.Http4xxCount, stats.Http5xxCount)
	fmt.Printf("    others - %d\n", stats.OtherCount)

	errors := stats.GetErrors()
	if len(errors) > 0 {
		fmt.Println("  Errors:")
		for errMsg, count := range errors {
			fmt.Printf("    %s - %d\n", errMsg, count)
		}
	}

	fmt.Printf("  Throughput:   %5.2fMB/s\n", stats.ThroughputMBps())

	// Show per-request stats if multiple URLs
	stats.Lock()
	if len(stats.RequestStats) > 1 {
		fmt.Println("\n  Per-Request Statistics:")
		for _, rs := range stats.RequestStats {
			avgLatency := float64(0)
			if rs.RequestCount > 0 {
				avgLatency = float64(rs.TotalLatency) / float64(rs.RequestCount)
			}
			fmt.Printf("    %s (%s %s)\n", rs.Name, rs.Method, rs.URL)
			fmt.Printf("      Requests: %d, Success: %d, Failed: %d, Avg Latency: %s\n",
				rs.RequestCount, rs.SuccessCount, rs.FailureCount, FormatLatency(avgLatency))
		}
	}
	stats.Unlock()
}

// WriteConsoleQuiet outputs minimal results to console (quiet mode)
func WriteConsoleQuiet(stats *benchmark.Stats) {
	fmt.Printf("Requests: %d, Duration: %.2fs, Req/s: %.2f, Avg Latency: %s, Errors: %d\n",
		stats.TotalRequests,
		stats.TotalDuration,
		stats.RequestsPerSecond,
		FormatLatency(stats.AverageResponseTime()),
		stats.FailureCount)
}
