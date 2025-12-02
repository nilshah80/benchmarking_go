// Package output handles benchmark result output in various formats
package output

import (
	"fmt"
)

// FormatLatency formats latency values with appropriate units
func FormatLatency(microseconds float64) string {
	if microseconds >= 1_000_000 {
		return fmt.Sprintf("%.2fs", microseconds/1_000_000)
	} else if microseconds >= 1_000 {
		return fmt.Sprintf("%.2fms", microseconds/1_000)
	} else {
		return fmt.Sprintf("%.2fus", microseconds)
	}
}

