// Package output handles benchmark result output in various formats
package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/benchmarking_go/pkg/benchmark"
	"github.com/benchmarking_go/pkg/config"
)

// WriteCSV outputs results in CSV format
func WriteCSV(stats *benchmark.Stats, cfg *config.Config) error {
	var output io.Writer = os.Stdout
	if cfg.Output.File != "" {
		file, err := os.Create(cfg.Output.File)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	writer := csv.NewWriter(output)
	defer writer.Flush()

	// Write header
	header := []string{
		"timestamp",
		"name",
		"duration_seconds",
		"total_requests",
		"success_count",
		"failure_count",
		"requests_per_second_avg",
		"requests_per_second_max",
		"latency_avg_us",
		"latency_min_us",
		"latency_max_us",
		"latency_std_dev_us",
	}

	// Add percentile headers
	for _, p := range cfg.Settings.Percentiles {
		header = append(header, fmt.Sprintf("latency_p%d_us", p))
	}

	header = append(header, []string{
		"http_1xx",
		"http_2xx",
		"http_3xx",
		"http_4xx",
		"http_5xx",
		"http_other",
		"throughput_bytes",
		"throughput_mb_per_sec",
	}...)

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	// Write data row
	row := []string{
		time.Now().UTC().Format(time.RFC3339),
		cfg.Name,
		strconv.FormatFloat(stats.TotalDuration, 'f', 3, 64),
		strconv.FormatInt(stats.TotalRequests, 10),
		strconv.FormatInt(stats.SuccessCount, 10),
		strconv.FormatInt(stats.FailureCount, 10),
		strconv.FormatFloat(stats.RequestsPerSecond, 'f', 2, 64),
		strconv.FormatFloat(stats.MaxRequestRate(), 'f', 2, 64),
		strconv.FormatFloat(stats.AverageResponseTime(), 'f', 2, 64),
		strconv.FormatInt(stats.MinResponseTime(), 10),
		strconv.FormatInt(stats.MaxResponseTime(), 10),
		strconv.FormatFloat(stats.StandardDeviation(), 'f', 2, 64),
	}

	// Add percentile values
	for _, p := range cfg.Settings.Percentiles {
		row = append(row, strconv.FormatInt(stats.GetLatencyPercentile(p), 10))
	}

	row = append(row, []string{
		strconv.FormatInt(stats.Http1xxCount, 10),
		strconv.FormatInt(stats.Http2xxCount, 10),
		strconv.FormatInt(stats.Http3xxCount, 10),
		strconv.FormatInt(stats.Http4xxCount, 10),
		strconv.FormatInt(stats.Http5xxCount, 10),
		strconv.FormatInt(stats.OtherCount, 10),
		strconv.FormatInt(stats.TotalBytes, 10),
		strconv.FormatFloat(stats.ThroughputMBps(), 'f', 4, 64),
	}...)

	if err := writer.Write(row); err != nil {
		return fmt.Errorf("error writing CSV data: %w", err)
	}

	return nil
}

// WriteCSVPerRequest outputs per-request results in CSV format
func WriteCSVPerRequest(stats *benchmark.Stats, cfg *config.Config) error {
	var output io.Writer = os.Stdout
	if cfg.Output.File != "" {
		file, err := os.Create(cfg.Output.File)
		if err != nil {
			return fmt.Errorf("error creating output file: %w", err)
		}
		defer file.Close()
		output = file
	}

	writer := csv.NewWriter(output)
	defer writer.Flush()

	// Write header
	header := []string{
		"timestamp",
		"benchmark_name",
		"request_name",
		"url",
		"method",
		"request_count",
		"success_count",
		"failure_count",
		"avg_latency_us",
		"errors",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing CSV header: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Write data rows for each request type
	stats.Lock()
	defer stats.Unlock()

	for _, rs := range stats.RequestStats {
		avgLatency := float64(0)
		if rs.RequestCount > 0 {
			avgLatency = float64(rs.TotalLatency) / float64(rs.RequestCount)
		}

		// Format errors as "error1:count1;error2:count2"
		errorStr := ""
		if len(rs.Errors) > 0 {
			first := true
			for errMsg, count := range rs.Errors {
				if !first {
					errorStr += ";"
				}
				errorStr += fmt.Sprintf("%s:%d", errMsg, count)
				first = false
			}
		}

		row := []string{
			timestamp,
			cfg.Name,
			rs.Name,
			rs.URL,
			rs.Method,
			strconv.FormatInt(rs.RequestCount, 10),
			strconv.FormatInt(rs.SuccessCount, 10),
			strconv.FormatInt(rs.FailureCount, 10),
			strconv.FormatFloat(avgLatency, 'f', 2, 64),
			errorStr,
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing CSV data: %w", err)
		}
	}

	return nil
}

