package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BenchmarkStats tracks statistics for the benchmark
type BenchmarkStats struct {
	TotalRequests     int64
	SuccessCount      int64
	FailureCount      int64
	TotalDuration     float64
	RequestsPerSecond float64

	// HTTP status code counters
	Http1xxCount int64
	Http2xxCount int64
	Http3xxCount int64
	Http4xxCount int64
	Http5xxCount int64
	OtherCount   int64

	// Throughput tracking
	TotalBytes int64

	mutex            sync.Mutex
	totalResponseTime int64
	responseCount    int64
	minResponseTime  int64
	maxResponseTime  int64

	// For standard deviation calculation
	responseTimes []float64

	// For request rate statistics
	requestRates   []float64
	maxRequestRate float64

	// For error tracking
	errors map[string]int
}

// NewBenchmarkStats creates a new BenchmarkStats instance
func NewBenchmarkStats() *BenchmarkStats {
	return &BenchmarkStats{
		minResponseTime: math.MaxInt64,
		errors:          make(map[string]int),
		responseTimes:   make([]float64, 0),
		requestRates:    make([]float64, 0),
	}
}

// AddResponseTime adds a response time measurement
func (s *BenchmarkStats) AddResponseTime(responseTimeMicros int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.totalResponseTime += responseTimeMicros
	s.responseCount++
	if responseTimeMicros < s.minResponseTime {
		s.minResponseTime = responseTimeMicros
	}
	if responseTimeMicros > s.maxResponseTime {
		s.maxResponseTime = responseTimeMicros
	}
	s.responseTimes = append(s.responseTimes, float64(responseTimeMicros))
}

// AddError tracks an error
func (s *BenchmarkStats) AddError(errorMessage string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.errors[errorMessage]++
}

// GetErrors returns a copy of the error map
func (s *BenchmarkStats) GetErrors() map[string]int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	errors := make(map[string]int)
	for k, v := range s.errors {
		errors[k] = v
	}
	return errors
}

// GetLatencyPercentile calculates the percentile of response times
func (s *BenchmarkStats) GetLatencyPercentile(percentile int) int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.responseTimes) == 0 {
		return 0
	}

	// Create a copy and sort
	times := make([]float64, len(s.responseTimes))
	copy(times, s.responseTimes)
	sort.Float64s(times)

	// Calculate the index for the percentile
	index := int(math.Ceil(float64(percentile)/100.0*float64(len(times)))) - 1
	
	// Ensure index is within bounds
	index = int(math.Max(0, math.Min(float64(len(times)-1), float64(index))))
	
	return int64(times[index])
}

// AverageResponseTime calculates the average response time
func (s *BenchmarkStats) AverageResponseTime() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.responseCount > 0 {
		return float64(s.totalResponseTime) / float64(s.responseCount)
	}
	return 0
}

// MinResponseTime returns the minimum response time
func (s *BenchmarkStats) MinResponseTime() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.minResponseTime == math.MaxInt64 {
		return 0
	}
	return s.minResponseTime
}

// MaxResponseTime returns the maximum response time
func (s *BenchmarkStats) MaxResponseTime() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxResponseTime
}

// StandardDeviation calculates the standard deviation of response times
func (s *BenchmarkStats) StandardDeviation() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.responseTimes) <= 1 {
		return 0
	}

	avg := float64(s.totalResponseTime) / float64(s.responseCount)
	var sum float64
	for _, time := range s.responseTimes {
		sum += math.Pow(time-avg, 2)
	}

	return math.Sqrt(sum / float64(len(s.responseTimes)-1))
}

// ThroughputMBps calculates the throughput in MB/s
func (s *BenchmarkStats) ThroughputMBps() float64 {
	if s.TotalBytes > 0 {
		return (float64(s.TotalBytes) / 1024.0 / 1024.0) / s.TotalDuration
	}
	return 0
}

// FormatLatency formats latency values with appropriate units
func FormatLatency(microseconds float64) string {
	if microseconds >= 1_000_000 {
		// Convert to seconds
		return fmt.Sprintf("%.2fs", microseconds/1_000_000)
	} else if microseconds >= 1_000 {
		// Convert to milliseconds
		return fmt.Sprintf("%.2fms", microseconds/1_000)
	} else {
		// Keep as microseconds
		return fmt.Sprintf("%.2fus", microseconds)
	}
}

// AddRequestRate adds a request rate measurement
func (s *BenchmarkStats) AddRequestRate(requestsPerSecond float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.requestRates = append(s.requestRates, requestsPerSecond)
	if requestsPerSecond > s.maxRequestRate {
		s.maxRequestRate = requestsPerSecond
	}
}

// MaxRequestRate returns the maximum request rate
func (s *BenchmarkStats) MaxRequestRate() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxRequestRate
}

// RequestRateStdDev calculates the standard deviation of request rates
func (s *BenchmarkStats) RequestRateStdDev() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.requestRates) <= 1 {
		return 0
	}

	var sum float64
	var avg float64
	
	for _, rate := range s.requestRates {
		sum += rate
	}
	avg = sum / float64(len(s.requestRates))
	
	sum = 0
	for _, rate := range s.requestRates {
		sum += math.Pow(rate-avg, 2)
	}

	return math.Sqrt(sum / float64(len(s.requestRates)-1))
}

// AddStatusCode increments the counter for the appropriate status code range
func (s *BenchmarkStats) AddStatusCode(statusCode int) {
	if statusCode >= 100 && statusCode < 200 {
		atomic.AddInt64(&s.Http1xxCount, 1)
	} else if statusCode >= 200 && statusCode < 300 {
		atomic.AddInt64(&s.Http2xxCount, 1)
	} else if statusCode >= 300 && statusCode < 400 {
		atomic.AddInt64(&s.Http3xxCount, 1)
	} else if statusCode >= 400 && statusCode < 500 {
		atomic.AddInt64(&s.Http4xxCount, 1)
	} else if statusCode >= 500 && statusCode < 600 {
		atomic.AddInt64(&s.Http5xxCount, 1)
	} else {
		atomic.AddInt64(&s.OtherCount, 1)
	}
}

// AddBytes adds to the total bytes counter
func (s *BenchmarkStats) AddBytes(bytes int64) {
	atomic.AddInt64(&s.TotalBytes, bytes)
}

// ProgressBar displays and updates a progress bar
type ProgressBar struct {
	blockCount     int
	currentProgress float64
	startTime      time.Time
	currentText    string
	durationMode   bool
	mutex          sync.Mutex
	done           bool
}

// NewProgressBar creates a new progress bar
func NewProgressBar(durationMode bool) *ProgressBar {
	p := &ProgressBar{
		blockCount:   50,
		startTime:    time.Now(),
		durationMode: durationMode,
	}
	
	// Initialize the console progress bar
	fmt.Print("\033[?25l") // Hide cursor
	p.resetBar()
	
	return p
}

// Report updates the progress bar
func (p *ProgressBar) Report(value float64, requestCount int) {
	// Ensure we can actually reach 100%
	if value >= 0.999 {
		value = 1.0
	}
	
	p.mutex.Lock()
	p.currentProgress = math.Max(0, math.Min(1, value))
	p.mutex.Unlock()
	
	// Create text with progress percentage, bar, and request count
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
	
	// Update the progress bar
	p.updateText(text)
}

// updateText updates the progress bar text
func (p *ProgressBar) updateText(text string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	// Get length of common portion
	commonPrefixLength := 0
	commonLength := int(math.Min(float64(len(p.currentText)), float64(len(text))))
	
	for commonPrefixLength < commonLength && text[commonPrefixLength] == p.currentText[commonPrefixLength] {
		commonPrefixLength++
	}
	
	// Backtrack to the first differing character
	var outputBuilder strings.Builder
	for i := 0; i < len(p.currentText)-commonPrefixLength; i++ {
		outputBuilder.WriteRune('\b')
	}
	
	// Output new suffix
	outputBuilder.WriteString(text[commonPrefixLength:])
	
	// If the new text is shorter than the old one: delete overlapping characters
	overlapCount := len(p.currentText) - len(text)
	if overlapCount > 0 {
		outputBuilder.WriteString(strings.Repeat(" ", overlapCount))
		outputBuilder.WriteString(strings.Repeat("\b", overlapCount))
	}
	
	fmt.Print(outputBuilder.String())
	p.currentText = text
}

// resetBar initializes the progress bar
func (p *ProgressBar) resetBar() {
	p.updateText(fmt.Sprintf(" %3d%% [%s]", 0, strings.Repeat(" ", p.blockCount)))
}

// Close cleans up the progress bar
func (p *ProgressBar) Close() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	
	if !p.done {
		p.done = true
		fmt.Print("\033[?25h") // Show cursor
	}
}

// ForceComplete forces the progress bar to show completion
func (p *ProgressBar) ForceComplete(elapsed time.Duration, requestCount int) {
	p.mutex.Lock()
	p.currentProgress = 1.0
	p.mutex.Unlock()
	
	// Force immediate update to show 100% with duration
	progressBlockCount := p.blockCount // Full bar
	
	// Always include request count in both modes
	text := fmt.Sprintf(" 100%% [%s] %.0fs (%d requests)", 
		strings.Repeat("=", progressBlockCount),
		elapsed.Seconds(),
		requestCount)
	
	p.updateText(text)
	
	// Add a newline to ensure clean output for statistics
	fmt.Println()
}

// Header represents an HTTP header
type Header struct {
	Key   string
	Value string
}

// headerSliceFlag is a custom flag type for handling multiple headers
type headerSliceFlag []Header

func (h *headerSliceFlag) String() string {
	return fmt.Sprintf("%v", *h)
}

func (h *headerSliceFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("header must be in format 'key:value'")
	}
	*h = append(*h, Header{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
	return nil
}

func main() {
	// Default configuration
	var url string
	var concurrentUsers int
	var requestsPerUser int
	var durationSeconds int
	var httpMethod string
	var headers headerSliceFlag
	var requestBody string
	var contentType string
	var showHelp bool
	var timeout int

	// Parse command line arguments
	flag.StringVar(&url, "url", "", "The URL to benchmark")
	flag.StringVar(&url, "u", "", "The URL to benchmark (shorthand)")
	
	flag.IntVar(&concurrentUsers, "concurrent-users", 10, "Number of concurrent users")
	flag.IntVar(&concurrentUsers, "c", 10, "Number of concurrent users (shorthand)")
	
	flag.IntVar(&requestsPerUser, "requests-per-user", 100, "Number of requests per user")
	flag.IntVar(&requestsPerUser, "r", 100, "Number of requests per user (shorthand)")
	
	flag.IntVar(&durationSeconds, "duration", 0, "Duration in seconds for the benchmark")
	flag.IntVar(&durationSeconds, "d", 0, "Duration in seconds for the benchmark (shorthand)")
	
	flag.StringVar(&httpMethod, "method", "GET", "HTTP method to use")
	flag.StringVar(&httpMethod, "m", "GET", "HTTP method to use (shorthand)")
	
	flag.Var(&headers, "header", "Custom header to include in the request (format: 'key:value')")
	flag.Var(&headers, "H", "Custom header to include in the request (shorthand) (format: 'key:value')")
	
	flag.StringVar(&requestBody, "body", "", "Request body for POST/PUT")
	flag.StringVar(&requestBody, "b", "", "Request body for POST/PUT (shorthand)")
	
	flag.StringVar(&contentType, "content-type", "", "Content-Type of the request body")
	flag.StringVar(&contentType, "t", "", "Content-Type of the request body (shorthand)")
	
	flag.IntVar(&timeout, "timeout", 30, "Timeout in seconds for each request")
	
	flag.BoolVar(&showHelp, "help", false, "Display help message")
	flag.BoolVar(&showHelp, "h", false, "Display help message (shorthand)")
	
	flag.Parse()

	// Display help if requested or if URL is not provided
	if showHelp || url == "" {
		displayHelp()
		return
	}

	// Print the configuration
	fmt.Printf("Starting benchmark: %s\n", url)
	fmt.Printf("Concurrent users: %d\n", concurrentUsers)
	fmt.Printf("HTTP method: %s\n", httpMethod)
	fmt.Printf("Request timeout: %d seconds\n", timeout)
	if durationSeconds > 0 {
		fmt.Printf("Duration: %d seconds\n", durationSeconds)
	} else {
		fmt.Printf("Requests per user: %d\n", requestsPerUser)
	}
	if len(headers) > 0 {
		fmt.Println("Headers:")
		for _, h := range headers {
			fmt.Printf("  %s: %s\n", h.Key, h.Value)
		}
	}
	if requestBody != "" {
		fmt.Printf("Request body: %s\n", requestBody)
		if contentType != "" {
			fmt.Printf("Content-Type: %s\n", contentType)
		}
	}
	fmt.Println()

	// Set up context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		fmt.Println("\nBenchmark interrupted, shutting down...")
		cancel()
	}()

	// Run the benchmark
	stats := runBenchmark(ctx, url, httpMethod, headers, requestBody, contentType, concurrentUsers, requestsPerUser, durationSeconds, timeout)

	// Ensure we have a clean line for statistics output
	fmt.Println("\nStatistics        Avg      Stdev        Max")
	
	// Display Requests/sec statistics with proper integer formatting for the average
	fmt.Printf("  Reqs/sec    %10.2f   %8.2f   %9.2f\n", 
		stats.RequestsPerSecond, 
		stats.RequestRateStdDev(), 
		stats.MaxRequestRate())
	
	// Format latency values with appropriate units
	avgLatency := FormatLatency(stats.AverageResponseTime())
	stdevLatency := FormatLatency(stats.StandardDeviation())
	maxLatency := FormatLatency(float64(stats.MaxResponseTime()))
	
	// Ensure consistent alignment of values
	fmt.Printf("  Latency      %8s   %8s    %7s\n", avgLatency, stdevLatency, maxLatency)
	
	// Add latency distribution
	fmt.Println("  Latency Distribution")
	fmt.Printf("     50%%    %s\n", FormatLatency(float64(stats.GetLatencyPercentile(50))))
	fmt.Printf("     75%%    %s\n", FormatLatency(float64(stats.GetLatencyPercentile(75))))
	fmt.Printf("     90%%    %s\n", FormatLatency(float64(stats.GetLatencyPercentile(90))))
	fmt.Printf("     99%%    %s\n", FormatLatency(float64(stats.GetLatencyPercentile(99))))
	
	// HTTP code summary
	fmt.Println("  HTTP codes:")
	fmt.Printf("    1xx - %d, 2xx - %d, 3xx - %d, 4xx - %d, 5xx - %d\n", 
		stats.Http1xxCount, stats.Http2xxCount, stats.Http3xxCount, stats.Http4xxCount, stats.Http5xxCount)
	fmt.Printf("    others - %d\n", stats.OtherCount)
	
	// Display errors if any
	errors := stats.GetErrors()
	if len(errors) > 0 {
		fmt.Println("  Errors:")
		for error, count := range errors {
			fmt.Printf("    %s - %d\n", error, count)
		}
	}
	
	// Throughput
	fmt.Printf("  Throughput:   %5.2fMB/s\n", stats.ThroughputMBps())
}

// displayHelp shows command-line help information
func displayHelp() {
	fmt.Println("Benchmarking Go HTTP Client")
	fmt.Println("Usage: benchmarking_go [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -u, --url <url>                  The URL to benchmark")
	fmt.Println("  -c, --concurrent-users <number>  Number of concurrent users")
	fmt.Println("  -r, --requests-per-user <number> Number of requests per user")
	fmt.Println("  -d, --duration <seconds>         Duration in seconds for the benchmark")
	fmt.Println("  -m, --method <GET|POST|PUT|...>  HTTP method to use")
	fmt.Println("  -H, --header <header:value>      Custom header to include in the request")
	fmt.Println("  -b, --body <text>                Request body for POST/PUT")
	fmt.Println("  -t, --content-type <type>        Content-Type of the request body")
	fmt.Println("  --timeout <seconds>              Timeout in seconds for each request (default: 30)")
	fmt.Println("  -h, --help                       Display this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  benchmarking_go --url https://example.com --concurrent-users 20 --requests-per-user 50")
	fmt.Println("  benchmarking_go -u https://example.com -c 20 -d 30  # Run for 30 seconds")
	fmt.Println("  benchmarking_go -u https://api.example.com -m POST -b '{\"key\":\"value\"}' -t application/json -c 5 -d 10")
}

// runBenchmark executes the benchmark with the given parameters
func runBenchmark(ctx context.Context, url, httpMethod string, headers headerSliceFlag, requestBody, contentType string, 
	concurrentUsers, requestsPerUser, durationSeconds, timeoutSeconds int) *BenchmarkStats {
	
	stats := NewBenchmarkStats()
	var wg sync.WaitGroup
	stopwatch := time.Now()
	
	// Create cancellation context for duration-based benchmarking
	var benchCtx context.Context
	var benchCancel context.CancelFunc
	
	if durationSeconds > 0 {
		// For duration-based benchmarking, use a timer instead of context timeout
		// to avoid context deadline exceeded errors
		benchCtx, benchCancel = context.WithCancel(ctx)
		go func() {
			timer := time.NewTimer(time.Duration(durationSeconds) * time.Second)
			defer timer.Stop()
			select {
			case <-timer.C:
				benchCancel()
			case <-ctx.Done():
				benchCancel()
			}
		}()
	} else {
		benchCtx, benchCancel = context.WithCancel(ctx)
		defer benchCancel()
	}

	totalRequests := -1
	if durationSeconds <= 0 {
		totalRequests = concurrentUsers * requestsPerUser
	}
	
	var completedRequests int64 = 0
	
	// Update the console output based on the selected mode
	if durationSeconds > 0 {
		fmt.Printf("Benchmarking %s for %ds using %d connections\n", url, durationSeconds, concurrentUsers)
	} else {
		fmt.Printf("Benchmarking %s with %d requests using %d connections\n", url, totalRequests, concurrentUsers)
	}
	
	progress := NewProgressBar(durationSeconds > 0)
	defer progress.Close()
	
	// Track requests per second over time and update the progress bar
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-benchCtx.Done():
				return
			case <-ticker.C:
				elapsedSeconds := time.Since(stopwatch).Seconds()
				if elapsedSeconds > 0 {
					currentRate := float64(atomic.LoadInt64(&completedRequests)) / elapsedSeconds
					stats.AddRequestRate(currentRate)
				}
				
				// Update the progress bar in BOTH modes, but differently
				if durationSeconds > 0 {
					progressPercent := math.Min(1.0, elapsedSeconds/float64(durationSeconds))
					progress.Report(progressPercent, int(atomic.LoadInt64(&completedRequests)))
				} else if totalRequests > 0 {
					// For fixed request mode, update based on completed requests
					progress.Report(float64(atomic.LoadInt64(&completedRequests))/float64(totalRequests), 
						int(atomic.LoadInt64(&completedRequests)))
				}
			}
		}
	}()
	
	// Create HTTP client with configurable timeout
	client := &http.Client{
		Timeout: time.Duration(timeoutSeconds) * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        concurrentUsers,
			MaxIdleConnsPerHost: concurrentUsers,
			MaxConnsPerHost:     concurrentUsers,
			DisableCompression:  false, // Enable compression for better performance
		},
	}
	
	// Create a channel to limit the number of concurrent requests
	semaphore := make(chan struct{}, concurrentUsers)
	
	// Start worker goroutines
	for i := 0; i < concurrentUsers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			// In duration mode, keep sending requests until cancelled
			if durationSeconds > 0 {
				for {
					select {
					case <-benchCtx.Done():
						return
					case semaphore <- struct{}{}:
						processSingleRequest(benchCtx, client, httpMethod, url, headers, requestBody, contentType, stats)
						atomic.AddInt64(&completedRequests, 1)
						<-semaphore
					}
				}
			} else {
				// In fixed request mode, send the specified number of requests
				for j := 0; j < requestsPerUser; j++ {
					select {
					case <-benchCtx.Done():
						return
					case semaphore <- struct{}{}:
						processSingleRequest(benchCtx, client, httpMethod, url, headers, requestBody, contentType, stats)
						atomic.AddInt64(&completedRequests, 1)
						<-semaphore
						
						// If we've completed all requests, cancel the benchmark
						completed := atomic.LoadInt64(&completedRequests)
						if completed >= int64(totalRequests) {
							benchCancel()
							return
						}
					}
				}
			}
		}()
	}
	
	// Wait for all goroutines to complete
	wg.Wait()
	
	// Ensure progress bar shows completion
	progress.ForceComplete(time.Since(stopwatch), int(completedRequests))
	
	elapsed := time.Since(stopwatch)
	
	stats.TotalRequests = completedRequests
	stats.TotalDuration = elapsed.Seconds()
	stats.RequestsPerSecond = float64(completedRequests) / stats.TotalDuration
	
	fmt.Println(" Done!")
	return stats
}

// processSingleRequest sends a single HTTP request and updates statistics
func processSingleRequest(ctx context.Context, client *http.Client, httpMethod, url string, headers headerSliceFlag, 
	requestBody, contentType string, stats *BenchmarkStats) {
	
	// Check if context is already done before starting
	select {
	case <-ctx.Done():
		// Context already canceled, don't attempt the request
		return
	default:
		// Continue with the request
	}
	
	requestStart := time.Now()
	
	// Create a new context with a shorter timeout for this individual request
	// This prevents the "context deadline exceeded" errors from propagating to the output
	reqCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Create request with the new context
	var req *http.Request
	var err error
	
	if requestBody != "" {
		req, err = http.NewRequestWithContext(reqCtx, httpMethod, url, bytes.NewBufferString(requestBody))
	} else {
		req, err = http.NewRequestWithContext(reqCtx, httpMethod, url, nil)
	}
	
	if err != nil {
		// Check if the parent context is done before recording error
		select {
		case <-ctx.Done():
			return
		default:
			atomic.AddInt64(&stats.FailureCount, 1)
			stats.AddError(err.Error())
			return
		}
	}
	
	// Add headers to the request
	for _, header := range headers {
		if !strings.EqualFold(header.Key, "Content-Type") {
			req.Header.Add(header.Key, header.Value)
		}
	}
	
	// Set content type if provided
	if requestBody != "" {
		effectiveContentType := contentType
		if effectiveContentType == "" {
			// Look for content type in headers
			for _, header := range headers {
				if strings.EqualFold(header.Key, "Content-Type") {
					effectiveContentType = header.Value
					break
				}
			}
			
			// Default to application/json if not specified
			if effectiveContentType == "" {
				effectiveContentType = "application/json"
			}
		}
		
		req.Header.Set("Content-Type", effectiveContentType)
	}
	
	// Add user-agent header if not already set
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "benchmarking_go/1.0")
	}
	
	// Send request
	resp, err := client.Do(req)
	
	// Check if the parent context is done before processing response
	select {
	case <-ctx.Done():
		// Benchmark is ending, don't record any errors
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return
	default:
		// Continue processing
	}
	
	if err != nil {
		atomic.AddInt64(&stats.FailureCount, 1)
		
		// Filter out context-related errors
		if !strings.Contains(err.Error(), "context") {
			stats.AddError(err.Error())
		}
		return
	}
	defer resp.Body.Close()
	
	// Track status code
	stats.AddStatusCode(resp.StatusCode)
	
	// Read and measure response content
	body, err := io.ReadAll(resp.Body)
	
	// Check again if the parent context is done
	select {
	case <-ctx.Done():
		return
	default:
		// Continue processing
	}
	
	if err != nil {
		atomic.AddInt64(&stats.FailureCount, 1)
		
		// Filter out context-related errors
		if !strings.Contains(err.Error(), "context") {
			stats.AddError(err.Error())
		}
		return
	}
	
	stats.AddBytes(int64(len(body)))
	
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		atomic.AddInt64(&stats.SuccessCount, 1)
	} else {
		atomic.AddInt64(&stats.FailureCount, 1)
	}
	
	// Update timing stats (convert to microseconds)
	responseTime := time.Since(requestStart).Microseconds()
	stats.AddResponseTime(responseTime)
}
