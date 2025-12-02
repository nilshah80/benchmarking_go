// Package benchmark provides benchmarking functionality
package benchmark

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/benchmarking_go/pkg/config"
	"golang.org/x/net/http2"
)

// createHTTPClient creates and configures the HTTP client
func (r *Runner) createHTTPClient() {
	// Base TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: r.Config.Settings.Insecure,
	}

	// Check if HTTP/2 is enabled
	if r.Config.Settings.HTTP2 {
		r.createHTTP2Client(tlsConfig)
		return
	}

	// Standard HTTP/1.1 transport
	transport := &http.Transport{
		MaxIdleConns:        r.Config.Settings.ConcurrentUsers,
		MaxIdleConnsPerHost: r.Config.Settings.ConcurrentUsers,
		MaxConnsPerHost:     r.Config.Settings.ConcurrentUsers,
		DisableCompression:  false,
		DisableKeepAlives:   r.Config.IsKeepAliveDisabled(),
		TLSClientConfig:     tlsConfig,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	r.client = &http.Client{
		Timeout:   time.Duration(r.TimeoutSec) * time.Second,
		Transport: transport,
	}
}

// createHTTP2Client creates an HTTP/2 enabled client
func (r *Runner) createHTTP2Client(tlsConfig *tls.Config) {
	// HTTP/2 transport
	transport := &http2.Transport{
		TLSClientConfig: tlsConfig,
		AllowHTTP:       false, // Only allow HTTPS for HTTP/2
		ReadIdleTimeout: 30 * time.Second,
		PingTimeout:     15 * time.Second,
	}

	r.client = &http.Client{
		Timeout:   time.Duration(r.TimeoutSec) * time.Second,
		Transport: transport,
	}
}

// processRequest processes a single HTTP request and records statistics
func (r *Runner) processRequest(ctx context.Context, reqConfig *config.RequestConfig) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	requestStart := time.Now()

	reqCtx, cancel := context.WithTimeout(context.Background(), time.Duration(r.TimeoutSec)*time.Second)
	defer cancel()

	// Prepare body
	body, err := config.PrepareRequestBody(reqConfig)
	if err != nil {
		r.Stats.IncrementFailure()
		r.Stats.AddError(err.Error())
		return
	}

	// Resolve URL variables
	url := config.ResolveVariables(reqConfig.URL, r.Config.Variables)

	// Create request
	var req *http.Request
	if body != "" {
		req, err = http.NewRequestWithContext(reqCtx, reqConfig.Method, url, bytes.NewBufferString(body))
	} else {
		req, err = http.NewRequestWithContext(reqCtx, reqConfig.Method, url, nil)
	}

	if err != nil {
		select {
		case <-ctx.Done():
			return
		default:
			r.Stats.IncrementFailure()
			r.Stats.AddError(err.Error())
			return
		}
	}

	// Add headers
	r.addHeaders(req, reqConfig, body)

	// Verbose logging
	if r.VerboseMode {
		fmt.Printf("[verbose] %s %s\n", reqConfig.Method, url)
	}

	// Send request
	resp, err := r.client.Do(req)

	select {
	case <-ctx.Done():
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return
	default:
	}

	if err != nil {
		r.Stats.IncrementFailure()
		if !strings.Contains(err.Error(), "context") {
			r.Stats.AddError(err.Error())
		}
		return
	}
	defer resp.Body.Close()

	// Record response
	r.recordResponse(ctx, resp, reqConfig, requestStart)
}

// addHeaders adds all required headers to the request
func (r *Runner) addHeaders(req *http.Request, reqConfig *config.RequestConfig, body string) {
	// Add default headers
	for key, value := range r.Config.DefaultHeaders {
		req.Header.Set(key, config.ResolveVariables(value, r.Config.Variables))
	}

	// Add request-specific headers
	for key, value := range reqConfig.Headers {
		req.Header.Set(key, config.ResolveVariables(value, r.Config.Variables))
	}

	// Set default content type for body
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set user agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "benchmarking_go/2.1")
	}
}

// recordResponse records the response statistics
func (r *Runner) recordResponse(ctx context.Context, resp *http.Response, reqConfig *config.RequestConfig, requestStart time.Time) {
	r.Stats.AddStatusCode(resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)

	select {
	case <-ctx.Done():
		return
	default:
	}

	if err != nil {
		r.Stats.IncrementFailure()
		if !strings.Contains(err.Error(), "context") {
			r.Stats.AddError(err.Error())
		}
		return
	}

	r.Stats.AddBytes(int64(len(respBody)))

	responseTime := time.Since(requestStart).Microseconds()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		r.Stats.IncrementSuccess()
	} else {
		r.Stats.IncrementFailure()
	}

	r.Stats.AddResponseTime(responseTime)

	// Verbose response logging
	if r.VerboseMode {
		url := config.ResolveVariables(reqConfig.URL, r.Config.Variables)
		fmt.Printf("[verbose] %s %s -> %d (%s)\n", reqConfig.Method, url, resp.StatusCode, time.Duration(responseTime)*time.Microsecond)
	}

	// Update per-request stats
	r.updateRequestStats(reqConfig, resp.StatusCode, responseTime)
}

// updateRequestStats updates the per-request statistics
func (r *Runner) updateRequestStats(reqConfig *config.RequestConfig, statusCode int, responseTime int64) {
	reqStats := r.Stats.GetOrCreateRequestStats(reqConfig.Name, reqConfig.URL, reqConfig.Method)
	reqStats.Mutex.Lock()
	reqStats.RequestCount++
	reqStats.TotalLatency += responseTime
	if statusCode >= 200 && statusCode < 300 {
		reqStats.SuccessCount++
	} else {
		reqStats.FailureCount++
	}
	reqStats.Mutex.Unlock()
}

