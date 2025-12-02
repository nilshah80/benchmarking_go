// Package benchmark provides benchmarking functionality
package benchmark

import (
	"context"
	"math/rand"
	"time"

	"github.com/benchmarking_go/pkg/config"
)

// RateLimiter controls the rate of requests using a token bucket algorithm
type RateLimiter struct {
	rate   int           // requests per second
	tokens chan struct{} // token bucket
	done   chan struct{}
	ticker *time.Ticker
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(ratePerSecond int) *RateLimiter {
	if ratePerSecond <= 0 {
		return nil
	}

	rl := &RateLimiter{
		rate:   ratePerSecond,
		tokens: make(chan struct{}, ratePerSecond*2), // Buffer for burst
		done:   make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < ratePerSecond; i++ {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}

	// Start token refill goroutine
	interval := time.Second / time.Duration(ratePerSecond)
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	rl.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-rl.done:
				return
			case <-rl.ticker.C:
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Token bucket full, discard
				}
			}
		}
	}()

	return rl
}

// Wait waits for a token to become available
func (rl *RateLimiter) Wait(ctx context.Context) bool {
	if rl == nil {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-rl.tokens:
		return true
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	if rl == nil {
		return
	}
	close(rl.done)
	rl.ticker.Stop()
}

// WeightedRequestSelector selects requests based on their weights
type WeightedRequestSelector struct {
	requests          []config.RequestConfig
	totalWeight       int
	cumulativeWeights []int
}

// NewWeightedRequestSelector creates a new weighted request selector
func NewWeightedRequestSelector(requests []config.RequestConfig) *WeightedRequestSelector {
	selector := &WeightedRequestSelector{
		requests:          requests,
		cumulativeWeights: make([]int, len(requests)),
	}

	cumulative := 0
	for i, req := range requests {
		cumulative += req.Weight
		selector.cumulativeWeights[i] = cumulative
	}
	selector.totalWeight = cumulative

	return selector
}

// Select returns a random request based on weights
func (s *WeightedRequestSelector) Select() *config.RequestConfig {
	if len(s.requests) == 1 {
		return &s.requests[0]
	}

	r := rand.Intn(s.totalWeight)
	for i, cumWeight := range s.cumulativeWeights {
		if r < cumWeight {
			return &s.requests[i]
		}
	}
	return &s.requests[len(s.requests)-1]
}

