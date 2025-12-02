// Package benchmark provides benchmarking functionality
package benchmark

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
)

// Stats tracks statistics for the benchmark
type Stats struct {
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

	mutex             sync.Mutex
	totalResponseTime int64
	responseCount     int64
	minResponseTime   int64
	maxResponseTime   int64

	// For standard deviation calculation
	responseTimes []float64

	// For request rate statistics
	requestRates   []float64
	maxRequestRate float64

	// For error tracking
	errors map[string]int

	// Per-request stats (for multi-URL benchmarks)
	RequestStats map[string]*RequestStats
}

// RequestStats tracks statistics for individual request types
type RequestStats struct {
	Name         string
	URL          string
	Method       string
	RequestCount int64
	SuccessCount int64
	FailureCount int64
	TotalLatency int64
	Mutex        sync.Mutex
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		minResponseTime: math.MaxInt64,
		errors:          make(map[string]int),
		responseTimes:   make([]float64, 0),
		requestRates:    make([]float64, 0),
		RequestStats:    make(map[string]*RequestStats),
	}
}

// GetOrCreateRequestStats gets or creates stats for a specific request
func (s *Stats) GetOrCreateRequestStats(name, url, method string) *RequestStats {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if stats, ok := s.RequestStats[name]; ok {
		return stats
	}

	stats := &RequestStats{
		Name:   name,
		URL:    url,
		Method: method,
	}
	s.RequestStats[name] = stats
	return stats
}

// AddResponseTime adds a response time measurement
func (s *Stats) AddResponseTime(responseTimeMicros int64) {
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
func (s *Stats) AddError(errorMessage string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.errors[errorMessage]++
}

// GetErrors returns a copy of the error map
func (s *Stats) GetErrors() map[string]int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	errors := make(map[string]int)
	for k, v := range s.errors {
		errors[k] = v
	}
	return errors
}

// GetLatencyPercentile calculates the percentile of response times
func (s *Stats) GetLatencyPercentile(percentile int) int64 {
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
func (s *Stats) AverageResponseTime() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.responseCount > 0 {
		return float64(s.totalResponseTime) / float64(s.responseCount)
	}
	return 0
}

// MinResponseTime returns the minimum response time
func (s *Stats) MinResponseTime() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.minResponseTime == math.MaxInt64 {
		return 0
	}
	return s.minResponseTime
}

// MaxResponseTime returns the maximum response time
func (s *Stats) MaxResponseTime() int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxResponseTime
}

// StandardDeviation calculates the standard deviation of response times
func (s *Stats) StandardDeviation() float64 {
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
func (s *Stats) ThroughputMBps() float64 {
	if s.TotalBytes > 0 && s.TotalDuration > 0 {
		return (float64(s.TotalBytes) / 1024.0 / 1024.0) / s.TotalDuration
	}
	return 0
}

// AddRequestRate adds a request rate measurement
func (s *Stats) AddRequestRate(requestsPerSecond float64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.requestRates = append(s.requestRates, requestsPerSecond)
	if requestsPerSecond > s.maxRequestRate {
		s.maxRequestRate = requestsPerSecond
	}
}

// MaxRequestRate returns the maximum request rate
func (s *Stats) MaxRequestRate() float64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.maxRequestRate
}

// RequestRateStdDev calculates the standard deviation of request rates
func (s *Stats) RequestRateStdDev() float64 {
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
func (s *Stats) AddStatusCode(statusCode int) {
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
func (s *Stats) AddBytes(bytes int64) {
	atomic.AddInt64(&s.TotalBytes, bytes)
}

// IncrementSuccess increments the success counter
func (s *Stats) IncrementSuccess() {
	atomic.AddInt64(&s.SuccessCount, 1)
}

// IncrementFailure increments the failure counter
func (s *Stats) IncrementFailure() {
	atomic.AddInt64(&s.FailureCount, 1)
}

// Lock locks the stats mutex
func (s *Stats) Lock() {
	s.mutex.Lock()
}

// Unlock unlocks the stats mutex
func (s *Stats) Unlock() {
	s.mutex.Unlock()
}

