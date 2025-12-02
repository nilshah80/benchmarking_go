// Package benchmark provides benchmarking functionality
package benchmark

import (
	"fmt"
	"strings"

	"github.com/benchmarking_go/pkg/config"
)

// ThresholdResult represents the result of a single threshold check
type ThresholdResult struct {
	Name     string // Name of the threshold (e.g., "Max Error Rate")
	Passed   bool   // Whether the threshold passed
	Expected string // Expected value
	Actual   string // Actual value
	Message  string // Human-readable message
}

// ThresholdResults represents all threshold check results
type ThresholdResults struct {
	Results []ThresholdResult
	Passed  bool // Overall pass/fail
}

// EvaluateThresholds checks if the benchmark results meet the defined thresholds
func EvaluateThresholds(stats *Stats, thresholds *config.ThresholdConfig) (*ThresholdResults, error) {
	results := &ThresholdResults{
		Results: make([]ThresholdResult, 0),
		Passed:  true,
	}

	if thresholds == nil || !thresholds.HasThresholds() {
		return results, nil
	}

	// Check error rate
	if thresholds.MaxErrorRate > 0 {
		result := checkErrorRate(stats, thresholds.MaxErrorRate)
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check average latency
	if thresholds.MaxAvgLatency != "" {
		result, err := checkAvgLatency(stats, thresholds.MaxAvgLatency)
		if err != nil {
			return nil, err
		}
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check P50 latency
	if thresholds.MaxP50Latency != "" {
		result, err := checkPercentileLatency(stats, 50, thresholds.MaxP50Latency)
		if err != nil {
			return nil, err
		}
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check P75 latency
	if thresholds.MaxP75Latency != "" {
		result, err := checkPercentileLatency(stats, 75, thresholds.MaxP75Latency)
		if err != nil {
			return nil, err
		}
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check P90 latency
	if thresholds.MaxP90Latency != "" {
		result, err := checkPercentileLatency(stats, 90, thresholds.MaxP90Latency)
		if err != nil {
			return nil, err
		}
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check P99 latency
	if thresholds.MaxP99Latency != "" {
		result, err := checkPercentileLatency(stats, 99, thresholds.MaxP99Latency)
		if err != nil {
			return nil, err
		}
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check minimum requests per second
	if thresholds.MinRequestsPerSecond > 0 {
		result := checkMinRPS(stats, thresholds.MinRequestsPerSecond)
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	// Check maximum requests per second
	if thresholds.MaxRequestsPerSecond > 0 {
		result := checkMaxRPS(stats, thresholds.MaxRequestsPerSecond)
		results.Results = append(results.Results, result)
		if !result.Passed {
			results.Passed = false
		}
	}

	return results, nil
}

// checkErrorRate checks if error rate is within threshold
func checkErrorRate(stats *Stats, maxErrorRate float64) ThresholdResult {
	totalRequests := stats.SuccessCount + stats.FailureCount
	var actualErrorRate float64
	if totalRequests > 0 {
		actualErrorRate = float64(stats.FailureCount) / float64(totalRequests)
	}

	passed := actualErrorRate <= maxErrorRate
	return ThresholdResult{
		Name:     "Max Error Rate",
		Passed:   passed,
		Expected: fmt.Sprintf("≤ %.2f%%", maxErrorRate*100),
		Actual:   fmt.Sprintf("%.2f%%", actualErrorRate*100),
		Message:  formatResultMessage("Error Rate", passed, fmt.Sprintf("%.2f%%", actualErrorRate*100), fmt.Sprintf("≤ %.2f%%", maxErrorRate*100)),
	}
}

// checkAvgLatency checks if average latency is within threshold
func checkAvgLatency(stats *Stats, maxLatencyStr string) (ThresholdResult, error) {
	maxLatencyMicros, err := config.ParseLatency(maxLatencyStr)
	if err != nil {
		return ThresholdResult{}, err
	}

	avgLatencyMicros := stats.AverageResponseTime()
	passed := int64(avgLatencyMicros) <= maxLatencyMicros

	return ThresholdResult{
		Name:     "Max Avg Latency",
		Passed:   passed,
		Expected: fmt.Sprintf("≤ %s", maxLatencyStr),
		Actual:   formatMicroseconds(int64(avgLatencyMicros)),
		Message:  formatResultMessage("Avg Latency", passed, formatMicroseconds(int64(avgLatencyMicros)), "≤ "+maxLatencyStr),
	}, nil
}

// checkPercentileLatency checks if a specific percentile latency is within threshold
func checkPercentileLatency(stats *Stats, percentile int, maxLatencyStr string) (ThresholdResult, error) {
	maxLatencyMicros, err := config.ParseLatency(maxLatencyStr)
	if err != nil {
		return ThresholdResult{}, err
	}

	actualLatencyMicros := stats.GetLatencyPercentile(percentile)
	passed := actualLatencyMicros <= maxLatencyMicros

	name := fmt.Sprintf("Max P%d Latency", percentile)
	return ThresholdResult{
		Name:     name,
		Passed:   passed,
		Expected: fmt.Sprintf("≤ %s", maxLatencyStr),
		Actual:   formatMicroseconds(actualLatencyMicros),
		Message:  formatResultMessage(fmt.Sprintf("P%d Latency", percentile), passed, formatMicroseconds(actualLatencyMicros), "≤ "+maxLatencyStr),
	}, nil
}

// checkMinRPS checks if requests per second meets minimum threshold
func checkMinRPS(stats *Stats, minRPS float64) ThresholdResult {
	actualRPS := stats.RequestsPerSecond
	passed := actualRPS >= minRPS

	return ThresholdResult{
		Name:     "Min Requests/sec",
		Passed:   passed,
		Expected: fmt.Sprintf("≥ %.2f", minRPS),
		Actual:   fmt.Sprintf("%.2f", actualRPS),
		Message:  formatResultMessage("Requests/sec", passed, fmt.Sprintf("%.2f", actualRPS), fmt.Sprintf("≥ %.2f", minRPS)),
	}
}

// checkMaxRPS checks if requests per second is within maximum threshold
func checkMaxRPS(stats *Stats, maxRPS float64) ThresholdResult {
	actualRPS := stats.RequestsPerSecond
	passed := actualRPS <= maxRPS

	return ThresholdResult{
		Name:     "Max Requests/sec",
		Passed:   passed,
		Expected: fmt.Sprintf("≤ %.2f", maxRPS),
		Actual:   fmt.Sprintf("%.2f", actualRPS),
		Message:  formatResultMessage("Requests/sec", passed, fmt.Sprintf("%.2f", actualRPS), fmt.Sprintf("≤ %.2f", maxRPS)),
	}
}

// formatMicroseconds formats microseconds into a human-readable duration
func formatMicroseconds(micros int64) string {
	if micros < 1000 {
		return fmt.Sprintf("%dµs", micros)
	} else if micros < 1000000 {
		return fmt.Sprintf("%.2fms", float64(micros)/1000)
	} else {
		return fmt.Sprintf("%.2fs", float64(micros)/1000000)
	}
}

// formatResultMessage formats a threshold result message
func formatResultMessage(name string, passed bool, actual, expected string) string {
	status := "✓ PASS"
	if !passed {
		status = "✗ FAIL"
	}
	return fmt.Sprintf("%s: %s (actual: %s, expected: %s)", status, name, actual, expected)
}

// FormatResults returns a formatted string of all threshold results
func (r *ThresholdResults) FormatResults() string {
	if len(r.Results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n  Threshold Results:\n")

	for _, result := range r.Results {
		sb.WriteString("    ")
		sb.WriteString(result.Message)
		sb.WriteString("\n")
	}

	if r.Passed {
		sb.WriteString("\n  ✓ All thresholds passed\n")
	} else {
		sb.WriteString("\n  ✗ Some thresholds failed\n")
	}

	return sb.String()
}

// FailedCount returns the number of failed thresholds
func (r *ThresholdResults) FailedCount() int {
	count := 0
	for _, result := range r.Results {
		if !result.Passed {
			count++
		}
	}
	return count
}

// PassedCount returns the number of passed thresholds
func (r *ThresholdResults) PassedCount() int {
	count := 0
	for _, result := range r.Results {
		if result.Passed {
			count++
		}
	}
	return count
}
