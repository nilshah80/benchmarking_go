// Package output handles benchmark result output in various formats
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/benchmarking_go/pkg/benchmark"
	"github.com/benchmarking_go/pkg/config"
)

// Result represents the JSON output format for benchmark results
type Result struct {
	Name           string              `json:"name,omitempty"`
	Timestamp      string              `json:"timestamp"`
	Duration       float64             `json:"duration_seconds"`
	TotalRequests  int64               `json:"total_requests"`
	SuccessCount   int64               `json:"success_count"`
	FailureCount   int64               `json:"failure_count"`
	RequestsPerSec RequestsPerSecStats `json:"requests_per_second"`
	Latency        LatencyStats        `json:"latency"`
	HTTPCodes      HTTPCodeStats       `json:"http_codes"`
	Throughput     ThroughputStats     `json:"throughput"`
	Errors         map[string]int      `json:"errors,omitempty"`
	Requests       []RequestResult     `json:"requests,omitempty"`
}

// RequestsPerSecStats contains request rate statistics
type RequestsPerSecStats struct {
	Average float64 `json:"average"`
	StdDev  float64 `json:"std_dev"`
	Max     float64 `json:"max"`
}

// LatencyStats contains latency statistics
type LatencyStats struct {
	Average     string            `json:"average"`
	StdDev      string            `json:"std_dev"`
	Min         string            `json:"min"`
	Max         string            `json:"max"`
	Percentiles map[string]string `json:"percentiles"`
}

// HTTPCodeStats contains HTTP status code counts
type HTTPCodeStats struct {
	Code1xx int64 `json:"1xx"`
	Code2xx int64 `json:"2xx"`
	Code3xx int64 `json:"3xx"`
	Code4xx int64 `json:"4xx"`
	Code5xx int64 `json:"5xx"`
	Other   int64 `json:"other"`
}

// ThroughputStats contains throughput statistics
type ThroughputStats struct {
	TotalBytes int64   `json:"total_bytes"`
	MBPerSec   float64 `json:"mb_per_second"`
}

// RequestResult contains per-request statistics
type RequestResult struct {
	Name         string `json:"name"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	RequestCount int64  `json:"request_count"`
	SuccessCount int64  `json:"success_count"`
	FailureCount int64  `json:"failure_count"`
	AvgLatency   string `json:"avg_latency"`
}

// ToJSONResult converts Stats to Result for JSON output
func ToJSONResult(stats *benchmark.Stats, cfg *config.Config) *Result {
	// Build percentiles map using custom percentiles from config
	percentiles := cfg.Settings.Percentiles
	if len(percentiles) == 0 {
		percentiles = []int{50, 75, 90, 99}
	}

	percentilesMap := make(map[string]string)
	for _, p := range percentiles {
		key := fmt.Sprintf("p%d", p)
		percentilesMap[key] = FormatLatency(float64(stats.GetLatencyPercentile(p)))
	}

	result := &Result{
		Name:          cfg.Name,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Duration:      stats.TotalDuration,
		TotalRequests: stats.TotalRequests,
		SuccessCount:  stats.SuccessCount,
		FailureCount:  stats.FailureCount,
		RequestsPerSec: RequestsPerSecStats{
			Average: stats.RequestsPerSecond,
			StdDev:  stats.RequestRateStdDev(),
			Max:     stats.MaxRequestRate(),
		},
		Latency: LatencyStats{
			Average:     FormatLatency(stats.AverageResponseTime()),
			StdDev:      FormatLatency(stats.StandardDeviation()),
			Min:         FormatLatency(float64(stats.MinResponseTime())),
			Max:         FormatLatency(float64(stats.MaxResponseTime())),
			Percentiles: percentilesMap,
		},
		HTTPCodes: HTTPCodeStats{
			Code1xx: stats.Http1xxCount,
			Code2xx: stats.Http2xxCount,
			Code3xx: stats.Http3xxCount,
			Code4xx: stats.Http4xxCount,
			Code5xx: stats.Http5xxCount,
			Other:   stats.OtherCount,
		},
		Throughput: ThroughputStats{
			TotalBytes: stats.TotalBytes,
			MBPerSec:   stats.ThroughputMBps(),
		},
		Errors: stats.GetErrors(),
	}

	// Add per-request stats
	stats.Lock()
	for _, rs := range stats.RequestStats {
		avgLatency := float64(0)
		if rs.RequestCount > 0 {
			avgLatency = float64(rs.TotalLatency) / float64(rs.RequestCount)
		}
		result.Requests = append(result.Requests, RequestResult{
			Name:         rs.Name,
			URL:          rs.URL,
			Method:       rs.Method,
			RequestCount: rs.RequestCount,
			SuccessCount: rs.SuccessCount,
			FailureCount: rs.FailureCount,
			AvgLatency:   FormatLatency(avgLatency),
		})
	}
	stats.Unlock()

	return result
}

// WriteJSON outputs results in JSON format
func WriteJSON(stats *benchmark.Stats, cfg *config.Config) error {
	result := ToJSONResult(stats, cfg)

	var output io.Writer = os.Stdout
	if cfg.Output.File != "" {
		file, err := os.Create(cfg.Output.File)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("error encoding JSON: %w", err)
	}

	return nil
}
