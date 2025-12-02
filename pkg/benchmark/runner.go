// Package benchmark provides benchmarking functionality
package benchmark

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benchmarking_go/pkg/config"
	"github.com/benchmarking_go/pkg/progress"
)

// Runner executes benchmarks
type Runner struct {
	Config        *config.Config
	DurationSec   int
	TimeoutSec    int
	RampUpSec     int
	QuietMode     bool
	VerboseMode   bool
	Stats         *Stats
	client        *http.Client
	selector      *WeightedRequestSelector
	rateLimiter   *RateLimiter
	activeWorkers int32
}

// NewRunner creates a new benchmark runner
func NewRunner(cfg *config.Config, durationSec, timeoutSec, rampUpSec int, quietMode, verboseMode bool) *Runner {
	// Create stats with histogram settings from config
	useHdr := !cfg.Settings.DisableHdr
	showHistogram := cfg.Settings.ShowHistogram
	stats := NewStatsWithOptions(useHdr, showHistogram)

	return &Runner{
		Config:      cfg,
		DurationSec: durationSec,
		TimeoutSec:  timeoutSec,
		RampUpSec:   rampUpSec,
		QuietMode:   quietMode,
		VerboseMode: verboseMode,
		Stats:       stats,
		selector:    NewWeightedRequestSelector(cfg.Requests),
	}
}

// Run executes the benchmark
func (r *Runner) Run(ctx context.Context) *Stats {
	// Check if scenario mode
	if r.Config.IsScenarioMode() {
		return r.RunScenario(ctx)
	}

	var wg sync.WaitGroup
	stopwatch := time.Now()

	// Initialize rate limiter if configured
	if r.Config.Settings.RateLimit > 0 {
		r.rateLimiter = NewRateLimiter(r.Config.Settings.RateLimit)
		defer r.rateLimiter.Stop()
	}

	// Create cancellation context
	benchCtx, benchCancel := r.createBenchmarkContext(ctx)
	if r.DurationSec <= 0 {
		defer benchCancel()
	}

	totalRequests := r.calculateTotalRequests()
	var completedRequests int64 = 0

	// Console output
	if !r.QuietMode {
		r.printBenchmarkStart(totalRequests)
	}

	progressBar := progress.NewBarWithOptions(r.DurationSec > 0, r.QuietMode, r.Config.Settings.ShowLiveStats)
	defer progressBar.Close()

	// Start progress tracking
	r.startProgressTracking(benchCtx, stopwatch, &completedRequests, totalRequests, progressBar)

	// Create HTTP client
	r.createHTTPClient()

	// Start workers
	r.startWorkers(benchCtx, benchCancel, &wg, &completedRequests, totalRequests)

	wg.Wait()

	progressBar.ForceComplete(time.Since(stopwatch), int(completedRequests))

	// Calculate final statistics
	elapsed := time.Since(stopwatch)
	r.Stats.TotalRequests = completedRequests
	r.Stats.TotalDuration = elapsed.Seconds()
	r.Stats.RequestsPerSecond = float64(completedRequests) / r.Stats.TotalDuration

	if !r.QuietMode {
		fmt.Println(" Done!")
	}

	return r.Stats
}

// RunScenario executes the benchmark in scenario mode
func (r *Runner) RunScenario(ctx context.Context) *Stats {
	var wg sync.WaitGroup
	stopwatch := time.Now()

	// Create cancellation context
	benchCtx, benchCancel := r.createBenchmarkContext(ctx)
	if r.DurationSec <= 0 {
		defer benchCancel()
	}

	// In scenario mode, each "iteration" is one complete scenario run
	// Total requests = scenarios * steps per scenario
	totalScenarios := r.Config.Settings.ConcurrentUsers * r.Config.Settings.RequestsPerUser
	stepsPerScenario := len(r.Config.Steps)
	var completedScenarios int64 = 0

	// Console output
	if !r.QuietMode {
		r.printScenarioStart(totalScenarios, stepsPerScenario)
	}

	progressBar := progress.NewBarWithOptions(r.DurationSec > 0, r.QuietMode, r.Config.Settings.ShowLiveStats)
	defer progressBar.Close()

	// Create HTTP client
	r.createHTTPClient()

	// Start progress tracking for scenarios
	r.startScenarioProgressTracking(benchCtx, stopwatch, &completedScenarios, totalScenarios, progressBar)

	// Start scenario workers
	r.startScenarioWorkers(benchCtx, benchCancel, &wg, &completedScenarios, totalScenarios)

	wg.Wait()

	progressBar.ForceComplete(time.Since(stopwatch), int(completedScenarios))

	// Calculate final statistics
	elapsed := time.Since(stopwatch)
	r.Stats.TotalRequests = completedScenarios * int64(stepsPerScenario)
	r.Stats.TotalDuration = elapsed.Seconds()
	r.Stats.RequestsPerSecond = float64(r.Stats.TotalRequests) / r.Stats.TotalDuration

	if !r.QuietMode {
		fmt.Println(" Done!")
	}

	return r.Stats
}

// printScenarioStart prints the scenario benchmark configuration at start
func (r *Runner) printScenarioStart(totalScenarios, stepsPerScenario int) {
	fmt.Printf("Scenario: %s\n", r.Config.Name)
	if r.Config.Description != "" {
		fmt.Printf("Description: %s\n", r.Config.Description)
	}
	fmt.Printf("Steps: %d\n", stepsPerScenario)
	for i, step := range r.Config.Steps {
		fmt.Printf("  %d. %s: %s %s\n", i+1, step.Name, step.Method, step.URL)
	}
	fmt.Printf("Concurrent users: %d\n", r.Config.Settings.ConcurrentUsers)
	if r.DurationSec > 0 {
		fmt.Printf("Duration: %d seconds\n", r.DurationSec)
	} else {
		fmt.Printf("Scenarios per user: %d (total: %d scenarios, %d requests)\n",
			r.Config.Settings.RequestsPerUser, totalScenarios, totalScenarios*stepsPerScenario)
	}
	fmt.Println()
}

// startScenarioProgressTracking starts progress tracking for scenario mode
func (r *Runner) startScenarioProgressTracking(ctx context.Context, stopwatch time.Time, completedScenarios *int64, totalScenarios int, progressBar *progress.Bar) {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsedSeconds := time.Since(stopwatch).Seconds()
				completed := atomic.LoadInt64(completedScenarios)
				stepsPerScenario := len(r.Config.Steps)
				totalRequests := completed * int64(stepsPerScenario)

				currentRate := float64(0)
				if elapsedSeconds > 0 {
					currentRate = float64(totalRequests) / elapsedSeconds
					r.Stats.AddRequestRate(currentRate)
				}

				// Build live stats if enabled
				var liveStats *progress.LiveStats
				if r.Config.Settings.ShowLiveStats {
					liveStats = &progress.LiveStats{
						RequestsPerSec: currentRate,
						AvgLatencyUs:   r.Stats.AverageResponseTime(),
						ErrorCount:     atomic.LoadInt64(&r.Stats.FailureCount),
						SuccessCount:   atomic.LoadInt64(&r.Stats.SuccessCount),
					}
				}

				if r.DurationSec > 0 {
					progressPercent := math.Min(1.0, elapsedSeconds/float64(r.DurationSec))
					progressBar.ReportWithStats(progressPercent, int(completed), liveStats)
				} else if totalScenarios > 0 {
					progressBar.ReportWithStats(float64(completed)/float64(totalScenarios), int(completed), liveStats)
				}
			}
		}
	}()
}

// startScenarioWorkers starts scenario worker goroutines
func (r *Runner) startScenarioWorkers(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, completedScenarios *int64, totalScenarios int) {
	semaphore := make(chan struct{}, r.Config.Settings.ConcurrentUsers)

	// Calculate ramp-up delay per worker
	rampUpDelay := time.Duration(0)
	if r.RampUpSec > 0 && r.Config.Settings.ConcurrentUsers > 1 {
		rampUpDelay = time.Duration(r.RampUpSec) * time.Second / time.Duration(r.Config.Settings.ConcurrentUsers-1)
	}

	for i := 0; i < r.Config.Settings.ConcurrentUsers; i++ {
		wg.Add(1)
		workerIndex := i

		go func() {
			defer wg.Done()
			r.runScenarioWorker(ctx, cancel, workerIndex, rampUpDelay, semaphore, completedScenarios, totalScenarios)
		}()
	}
}

// runScenarioWorker runs a single scenario worker
func (r *Runner) runScenarioWorker(ctx context.Context, cancel context.CancelFunc, workerIndex int, rampUpDelay time.Duration, semaphore chan struct{}, completedScenarios *int64, totalScenarios int) {
	// Apply ramp-up delay
	if rampUpDelay > 0 && workerIndex > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(rampUpDelay * time.Duration(workerIndex)):
		}
	}

	atomic.AddInt32(&r.activeWorkers, 1)
	defer atomic.AddInt32(&r.activeWorkers, -1)

	if r.VerboseMode && !r.QuietMode {
		fmt.Printf("[verbose] Scenario worker %d started\n", workerIndex)
	}

	executor := NewScenarioExecutor(r.Config, r.client, r.TimeoutSec, r.VerboseMode, r.Stats)

	if r.DurationSec > 0 {
		// Duration mode
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				executor.ExecuteScenario(ctx)
				atomic.AddInt64(completedScenarios, 1)
				<-semaphore
			}
		}
	} else {
		// Fixed count mode
		for j := 0; j < r.Config.Settings.RequestsPerUser; j++ {
			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case <-ctx.Done():
				return
			case semaphore <- struct{}{}:
				executor.ExecuteScenario(ctx)
				atomic.AddInt64(completedScenarios, 1)
				<-semaphore

				completed := atomic.LoadInt64(completedScenarios)
				if completed >= int64(totalScenarios) {
					cancel()
					return
				}
			}
		}
	}
}

// createBenchmarkContext creates the benchmark context with optional duration timer
func (r *Runner) createBenchmarkContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if r.DurationSec > 0 {
		benchCtx, benchCancel := context.WithCancel(ctx)
		go func() {
			timer := time.NewTimer(time.Duration(r.DurationSec) * time.Second)
			defer timer.Stop()
			select {
			case <-timer.C:
				benchCancel()
			case <-ctx.Done():
				benchCancel()
			}
		}()
		return benchCtx, benchCancel
	}
	return context.WithCancel(ctx)
}

// calculateTotalRequests calculates the total number of requests for fixed-request mode
func (r *Runner) calculateTotalRequests() int {
	if r.DurationSec <= 0 {
		return r.Config.Settings.ConcurrentUsers * r.Config.Settings.RequestsPerUser
	}
	return -1
}

// startProgressTracking starts the goroutine that tracks progress and request rates
func (r *Runner) startProgressTracking(ctx context.Context, stopwatch time.Time, completedRequests *int64, totalRequests int, progressBar *progress.Bar) {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsedSeconds := time.Since(stopwatch).Seconds()
				currentRate := float64(0)
				if elapsedSeconds > 0 {
					currentRate = float64(atomic.LoadInt64(completedRequests)) / elapsedSeconds
					r.Stats.AddRequestRate(currentRate)
				}

				// Build live stats if enabled
				var liveStats *progress.LiveStats
				if r.Config.Settings.ShowLiveStats {
					liveStats = &progress.LiveStats{
						RequestsPerSec: currentRate,
						AvgLatencyUs:   r.Stats.AverageResponseTime(),
						ErrorCount:     atomic.LoadInt64(&r.Stats.FailureCount),
						SuccessCount:   atomic.LoadInt64(&r.Stats.SuccessCount),
					}
				}

				reqCount := int(atomic.LoadInt64(completedRequests))
				if r.DurationSec > 0 {
					progressPercent := math.Min(1.0, elapsedSeconds/float64(r.DurationSec))
					progressBar.ReportWithStats(progressPercent, reqCount, liveStats)
				} else if totalRequests > 0 {
					progressBar.ReportWithStats(float64(reqCount)/float64(totalRequests), reqCount, liveStats)
				}
			}
		}
	}()
}

// startWorkers starts all worker goroutines with optional ramp-up
func (r *Runner) startWorkers(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, completedRequests *int64, totalRequests int) {
	semaphore := make(chan struct{}, r.Config.Settings.ConcurrentUsers)

	// Calculate ramp-up delay per worker
	rampUpDelay := time.Duration(0)
	if r.RampUpSec > 0 && r.Config.Settings.ConcurrentUsers > 1 {
		rampUpDelay = time.Duration(r.RampUpSec) * time.Second / time.Duration(r.Config.Settings.ConcurrentUsers-1)
	}

	for i := 0; i < r.Config.Settings.ConcurrentUsers; i++ {
		wg.Add(1)
		workerIndex := i

		go func() {
			defer wg.Done()
			r.runWorker(ctx, cancel, workerIndex, rampUpDelay, semaphore, completedRequests, totalRequests)
		}()
	}
}

// runWorker runs a single worker goroutine
func (r *Runner) runWorker(ctx context.Context, cancel context.CancelFunc, workerIndex int, rampUpDelay time.Duration, semaphore chan struct{}, completedRequests *int64, totalRequests int) {
	// Apply ramp-up delay
	if rampUpDelay > 0 && workerIndex > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(rampUpDelay * time.Duration(workerIndex)):
		}
	}

	atomic.AddInt32(&r.activeWorkers, 1)
	defer atomic.AddInt32(&r.activeWorkers, -1)

	if r.VerboseMode && !r.QuietMode {
		fmt.Printf("[verbose] Worker %d started\n", workerIndex)
	}

	if r.DurationSec > 0 {
		r.runDurationWorker(ctx, semaphore, completedRequests)
	} else {
		r.runFixedWorker(ctx, cancel, semaphore, completedRequests, totalRequests)
	}
}

// runDurationWorker runs requests until the context is cancelled (duration mode)
func (r *Runner) runDurationWorker(ctx context.Context, semaphore chan struct{}, completedRequests *int64) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for rate limiter
		if r.rateLimiter != nil && !r.rateLimiter.Wait(ctx) {
			return
		}

		select {
		case <-ctx.Done():
			return
		case semaphore <- struct{}{}:
			reqConfig := r.selector.Select()
			r.processRequest(ctx, reqConfig)
			atomic.AddInt64(completedRequests, 1)
			<-semaphore
		}
	}
}

// runFixedWorker runs a fixed number of requests per worker
func (r *Runner) runFixedWorker(ctx context.Context, cancel context.CancelFunc, semaphore chan struct{}, completedRequests *int64, totalRequests int) {
	for j := 0; j < r.Config.Settings.RequestsPerUser; j++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Wait for rate limiter
		if r.rateLimiter != nil && !r.rateLimiter.Wait(ctx) {
			return
		}

		select {
		case <-ctx.Done():
			return
		case semaphore <- struct{}{}:
			reqConfig := r.selector.Select()
			r.processRequest(ctx, reqConfig)
			atomic.AddInt64(completedRequests, 1)
			<-semaphore

			completed := atomic.LoadInt64(completedRequests)
			if completed >= int64(totalRequests) {
				cancel()
				return
			}
		}
	}
}

// printBenchmarkStart prints the benchmark configuration at start
func (r *Runner) printBenchmarkStart(totalRequests int) {
	if r.DurationSec > 0 {
		if len(r.Config.Requests) == 1 {
			fmt.Printf("Benchmarking %s for %ds using %d connections\n",
				r.Config.Requests[0].URL, r.DurationSec, r.Config.Settings.ConcurrentUsers)
		} else {
			fmt.Printf("Benchmarking %d URLs for %ds using %d connections\n",
				len(r.Config.Requests), r.DurationSec, r.Config.Settings.ConcurrentUsers)
		}
	} else {
		if len(r.Config.Requests) == 1 {
			fmt.Printf("Benchmarking %s with %d requests using %d connections\n",
				r.Config.Requests[0].URL, totalRequests, r.Config.Settings.ConcurrentUsers)
		} else {
			fmt.Printf("Benchmarking %d URLs with %d requests using %d connections\n",
				len(r.Config.Requests), totalRequests, r.Config.Settings.ConcurrentUsers)
		}
	}

	// Print additional info in verbose mode
	if r.VerboseMode {
		if r.Config.Settings.RateLimit > 0 {
			fmt.Printf("  Rate limit: %d req/s\n", r.Config.Settings.RateLimit)
		}
		if r.RampUpSec > 0 {
			fmt.Printf("  Ramp-up: %ds\n", r.RampUpSec)
		}
		if r.Config.IsKeepAliveDisabled() {
			fmt.Println("  Keep-alive: disabled")
		}
	}
}
