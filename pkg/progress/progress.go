// Package progress provides a console progress bar
package progress

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// Bar displays and updates a progress bar
type Bar struct {
	blockCount      int
	currentProgress float64
	startTime       time.Time
	currentText     string
	durationMode    bool
	mutex           sync.Mutex
	done            bool
	quiet           bool
}

// NewBar creates a new progress bar
func NewBar(durationMode bool, quiet bool) *Bar {
	p := &Bar{
		blockCount:   50,
		startTime:    time.Now(),
		durationMode: durationMode,
		quiet:        quiet,
	}

	if !quiet {
		fmt.Print("\033[?25l") // Hide cursor
		p.resetBar()
	}

	return p
}

// Report updates the progress bar
func (p *Bar) Report(value float64, requestCount int) {
	if p.quiet {
		return
	}

	if value >= 0.999 {
		value = 1.0
	}

	p.mutex.Lock()
	p.currentProgress = math.Max(0, math.Min(1, value))
	p.mutex.Unlock()

	progressBlockCount := int(p.currentProgress * float64(p.blockCount))
	percent := int(p.currentProgress * 100)

	var text string
	if requestCount > 0 {
		text = fmt.Sprintf(" %3d%% [%s%s] (%d requests)",
			percent,
			strings.Repeat("=", progressBlockCount),
			strings.Repeat(" ", p.blockCount-progressBlockCount),
			requestCount)
	} else {
		text = fmt.Sprintf(" %3d%% [%s%s]",
			percent,
			strings.Repeat("=", progressBlockCount),
			strings.Repeat(" ", p.blockCount-progressBlockCount))
	}

	p.updateText(text)
}

func (p *Bar) updateText(text string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	commonPrefixLength := 0
	commonLength := int(math.Min(float64(len(p.currentText)), float64(len(text))))

	for commonPrefixLength < commonLength && text[commonPrefixLength] == p.currentText[commonPrefixLength] {
		commonPrefixLength++
	}

	var outputBuilder strings.Builder
	for i := 0; i < len(p.currentText)-commonPrefixLength; i++ {
		outputBuilder.WriteRune('\b')
	}

	outputBuilder.WriteString(text[commonPrefixLength:])

	overlapCount := len(p.currentText) - len(text)
	if overlapCount > 0 {
		outputBuilder.WriteString(strings.Repeat(" ", overlapCount))
		outputBuilder.WriteString(strings.Repeat("\b", overlapCount))
	}

	fmt.Print(outputBuilder.String())
	p.currentText = text
}

func (p *Bar) resetBar() {
	p.updateText(fmt.Sprintf(" %3d%% [%s]", 0, strings.Repeat(" ", p.blockCount)))
}

// Close cleans up the progress bar
func (p *Bar) Close() {
	if p.quiet {
		return
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.done {
		p.done = true
		fmt.Print("\033[?25h") // Show cursor
	}
}

// ForceComplete forces the progress bar to show completion
func (p *Bar) ForceComplete(elapsed time.Duration, requestCount int) {
	if p.quiet {
		return
	}

	p.mutex.Lock()
	p.currentProgress = 1.0
	p.mutex.Unlock()

	progressBlockCount := p.blockCount

	text := fmt.Sprintf(" 100%% [%s] %.0fs (%d requests)",
		strings.Repeat("=", progressBlockCount),
		elapsed.Seconds(),
		requestCount)

	p.updateText(text)
	fmt.Println()
}

