// Package benchmark provides benchmarking functionality
package benchmark

import (
	"fmt"
	"math"
	"strings"

	"github.com/HdrHistogram/hdrhistogram-go"
)

// HdrStats provides memory-efficient statistics using HdrHistogram
type HdrStats struct {
	histogram *hdrhistogram.Histogram
	// Min/max tracking (HdrHistogram tracks these but we want atomic access)
	minValue int64
	maxValue int64
	count    int64
}

// NewHdrStats creates a new HdrStats instance
// minValue: minimum trackable value in microseconds (typically 1)
// maxValue: maximum trackable value in microseconds (e.g., 60000000 for 60 seconds)
// sigFigs: number of significant figures (1-5, typically 3)
func NewHdrStats(minValue, maxValue int64, sigFigs int) (*HdrStats, error) {
	h := hdrhistogram.New(minValue, maxValue, sigFigs)
	return &HdrStats{
		histogram: h,
		minValue:  math.MaxInt64,
		maxValue:  0,
	}, nil
}

// RecordValue records a latency value in microseconds
func (h *HdrStats) RecordValue(value int64) error {
	err := h.histogram.RecordValue(value)
	if err != nil {
		return err
	}
	h.count++
	if value < h.minValue {
		h.minValue = value
	}
	if value > h.maxValue {
		h.maxValue = value
	}
	return nil
}

// Mean returns the mean value
func (h *HdrStats) Mean() float64 {
	return h.histogram.Mean()
}

// StdDev returns the standard deviation
func (h *HdrStats) StdDev() float64 {
	return h.histogram.StdDev()
}

// Min returns the minimum recorded value
func (h *HdrStats) Min() int64 {
	if h.minValue == math.MaxInt64 {
		return 0
	}
	return h.minValue
}

// Max returns the maximum recorded value
func (h *HdrStats) Max() int64 {
	return h.maxValue
}

// Percentile returns the value at the given percentile (0-100)
func (h *HdrStats) Percentile(percentile float64) int64 {
	return h.histogram.ValueAtQuantile(percentile)
}

// Count returns the total number of recorded values
func (h *HdrStats) Count() int64 {
	return h.histogram.TotalCount()
}

// HistogramBucket represents a bucket in the ASCII histogram
type HistogramBucket struct {
	RangeStart int64   // Start of range in microseconds
	RangeEnd   int64   // End of range in microseconds
	Count      int64   // Number of values in this bucket
	Percentage float64 // Percentage of total
}

// GetHistogramBuckets returns buckets for ASCII histogram display
func (h *HdrStats) GetHistogramBuckets() []HistogramBucket {
	return h.GetCustomBuckets(nil)
}

// GetCustomBuckets returns histogram buckets with custom boundaries
// If boundaries is nil, uses default boundaries
func (h *HdrStats) GetCustomBuckets(boundaries []int64) []HistogramBucket {
	// Default boundaries in microseconds
	if boundaries == nil {
		boundaries = []int64{
			1000,      // 1ms
			5000,      // 5ms
			10000,     // 10ms
			25000,     // 25ms
			50000,     // 50ms
			100000,    // 100ms
			250000,    // 250ms
			500000,    // 500ms
			1000000,   // 1s
			2500000,   // 2.5s
			5000000,   // 5s
			10000000,  // 10s
		}
	}

	totalCount := h.histogram.TotalCount()
	if totalCount == 0 {
		return nil
	}

	// Use Distribution() to get counts per bucket
	distribution := h.histogram.Distribution()
	buckets := make([]HistogramBucket, 0)

	// Create a map of boundary index -> count
	boundaryCounts := make(map[int]int64)
	for i := range boundaries {
		boundaryCounts[i] = 0
	}
	var overflowCount int64 = 0

	// Iterate through distribution and assign to buckets
	for _, bar := range distribution {
		value := bar.To
		count := bar.Count

		assigned := false
		for i, boundary := range boundaries {
			if value <= boundary {
				boundaryCounts[i] += count
				assigned = true
				break
			}
		}
		if !assigned {
			overflowCount += count
		}
	}

	// Build bucket list - only include buckets with data
	var prevBoundary int64 = 0
	for i, boundary := range boundaries {
		count := boundaryCounts[i]
		if count > 0 {
			percentage := float64(count) / float64(totalCount) * 100
			buckets = append(buckets, HistogramBucket{
				RangeStart: prevBoundary,
				RangeEnd:   boundary,
				Count:      count,
				Percentage: percentage,
			})
		}
		prevBoundary = boundary
	}

	// Add overflow bucket
	if overflowCount > 0 {
		percentage := float64(overflowCount) / float64(totalCount) * 100
		buckets = append(buckets, HistogramBucket{
			RangeStart: prevBoundary,
			RangeEnd:   -1, // -1 indicates "and above"
			Count:      overflowCount,
			Percentage: percentage,
		})
	}

	return buckets
}

// FormatDuration formats microseconds to human-readable string
func FormatDuration(us int64) string {
	if us < 1000 {
		return fmt.Sprintf("%dus", us)
	} else if us < 1000000 {
		ms := float64(us) / 1000
		if ms < 10 {
			return fmt.Sprintf("%.1fms", ms)
		}
		return fmt.Sprintf("%.0fms", ms)
	} else {
		s := float64(us) / 1000000
		if s < 10 {
			return fmt.Sprintf("%.2fs", s)
		}
		return fmt.Sprintf("%.1fs", s)
	}
}

// FormatDurationShort formats microseconds to shorter human-readable string
func FormatDurationShort(us int64) string {
	if us < 1000 {
		return fmt.Sprintf("%dus", us)
	} else if us < 1000000 {
		return fmt.Sprintf("%.0fms", float64(us)/1000)
	} else {
		return fmt.Sprintf("%.1fs", float64(us)/1000000)
	}
}

// RenderASCIIHistogram renders an ASCII histogram from buckets
func RenderASCIIHistogram(buckets []HistogramBucket, maxBarWidth int) string {
	if len(buckets) == 0 {
		return "  No data recorded\n"
	}

	var sb strings.Builder
	sb.WriteString("\nLatency Histogram:\n")

	// Find max percentage for scaling
	maxPct := float64(0)
	for _, b := range buckets {
		if b.Percentage > maxPct {
			maxPct = b.Percentage
		}
	}

	if maxPct == 0 {
		maxPct = 1
	}

	for _, bucket := range buckets {
		// Format range label with shorter format
		var rangeLabel string
		if bucket.RangeStart == 0 {
			rangeLabel = fmt.Sprintf("  < %s", FormatDurationShort(bucket.RangeEnd))
		} else if bucket.RangeEnd == -1 {
			rangeLabel = fmt.Sprintf("  > %s", FormatDurationShort(bucket.RangeStart))
		} else {
			rangeLabel = fmt.Sprintf("  %s - %s", FormatDurationShort(bucket.RangeStart), FormatDurationShort(bucket.RangeEnd))
		}

		// Pad range label to fixed width (20 chars)
		rangeLabel = fmt.Sprintf("%-20s", rangeLabel)

		// Calculate bar width
		barWidth := int(math.Round(bucket.Percentage / maxPct * float64(maxBarWidth)))
		if barWidth < 0 {
			barWidth = 0
		}
		if barWidth > maxBarWidth {
			barWidth = maxBarWidth
		}

		// Create bar using simple ASCII characters for better terminal compatibility
		bar := strings.Repeat("#", barWidth)
		padding := strings.Repeat(" ", maxBarWidth-barWidth)

		// Format line
		sb.WriteString(fmt.Sprintf("%s [%s%s] %6.2f%% (%d)\n",
			rangeLabel, bar, padding, bucket.Percentage, bucket.Count))
	}

	return sb.String()
}

// Export exports the histogram data for serialization
func (h *HdrStats) Export() *hdrhistogram.Snapshot {
	return h.histogram.Export()
}

// Merge merges another HdrStats into this one
func (h *HdrStats) Merge(other *HdrStats) {
	h.histogram.Merge(other.histogram)
	if other.minValue < h.minValue {
		h.minValue = other.minValue
	}
	if other.maxValue > h.maxValue {
		h.maxValue = other.maxValue
	}
	h.count += other.count
}

// Reset resets the histogram
func (h *HdrStats) Reset() {
	h.histogram.Reset()
	h.minValue = math.MaxInt64
	h.maxValue = 0
	h.count = 0
}

