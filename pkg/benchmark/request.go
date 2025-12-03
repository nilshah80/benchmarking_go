// Package benchmark provides benchmarking functionality
package benchmark

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/benchmarking_go/pkg/config"
	"golang.org/x/net/http2"
)

// extractErrorMessage extracts error messages from response body
func extractErrorMessage(body []byte, contentType string) string {
	if len(body) == 0 {
		return ""
	}

	// Limit message length
	const maxMessageLength = 100

	// Try JSON parsing first if content type suggests JSON
	if strings.Contains(contentType, "json") || (len(body) > 0 && body[0] == '{') {
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			// Try common error message fields
			for _, key := range []string{"error", "message", "msg", "detail", "error_description", "errorMessage"} {
				if msg, ok := result[key].(string); ok && msg != "" {
					if len(msg) > maxMessageLength {
						return msg[:maxMessageLength-3] + "..."
					}
					return msg
				}
			}
			// Try nested error object
			if errorObj, ok := result["error"].(map[string]interface{}); ok {
				if msg, ok := errorObj["message"].(string); ok && msg != "" {
					if len(msg) > maxMessageLength {
						return msg[:maxMessageLength-3] + "..."
					}
					return msg
				}
			}
		}
	}

	// Try plain text parsing
	bodyStr := string(body)
	bodyStr = strings.TrimSpace(bodyStr)

	// Remove HTML tags if present
	if strings.Contains(bodyStr, "<") {
		htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
		bodyStr = htmlTagRegex.ReplaceAllString(bodyStr, " ")
		bodyStr = strings.TrimSpace(bodyStr)
	}

	// Normalize whitespace (replace multiple spaces/newlines with single space)
	whitespaceRegex := regexp.MustCompile(`\s+`)
	bodyStr = whitespaceRegex.ReplaceAllString(bodyStr, " ")
	bodyStr = strings.TrimSpace(bodyStr)

	// If body is short enough and meaningful, use it
	if len(bodyStr) > 0 && len(bodyStr) <= maxMessageLength {
		// Only return if it looks like an error message (contains some text)
		if len(bodyStr) > 5 && !strings.HasPrefix(bodyStr, "{") && !strings.HasPrefix(bodyStr, "[") {
			return bodyStr
		}
	} else if len(bodyStr) > maxMessageLength {
		// Truncate to max length
		return bodyStr[:maxMessageLength-3] + "..."
	}

	return ""
}

// categorizeError normalizes error messages for proper grouping
func categorizeError(err error) string {
	errStr := err.Error()

	// Connection/network errors
	if strings.Contains(errStr, "connection refused") {
		return "Connection refused"
	}
	if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
		return "DNS lookup failed"
	}
	if strings.Contains(errStr, "connection reset") {
		return "Connection reset by peer"
	}
	if strings.Contains(errStr, "broken pipe") {
		return "Broken pipe"
	}
	if strings.Contains(errStr, "network is unreachable") {
		return "Network unreachable"
	}
	if strings.Contains(errStr, "i/o timeout") {
		return "I/O timeout"
	}
	if strings.Contains(errStr, "TLS handshake") {
		return "TLS handshake error"
	}
	if strings.Contains(errStr, "certificate") {
		return "Certificate error"
	}
	if strings.Contains(errStr, "EOF") {
		return "Connection closed (EOF)"
	}
	if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "context canceled") {
		return "Request timeout"
	}

	// Truncate long messages but keep them informative
	if len(errStr) > 80 {
		return errStr[:77] + "..."
	}
	return errStr
}

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
// Note: This function will complete the full request cycle regardless of stopSending signal
// to ensure all started requests are properly recorded in statistics
func (r *Runner) processRequest(ctx context.Context, reqConfig *config.RequestConfig) {
	requestStart := time.Now()

	reqCtx, cancel := context.WithTimeout(context.Background(), time.Duration(r.TimeoutSec)*time.Second)
	defer cancel()

	// Prepare body
	body, err := config.PrepareRequestBody(reqConfig)
	if err != nil {
		errMsg := categorizeError(err)
		r.Stats.IncrementFailure()
		r.Stats.AddError(errMsg)
		r.Stats.AddStatusCode(0) // Track as 'other' for non-HTTP failure
		r.updateRequestStats(reqConfig, 0, time.Since(requestStart).Microseconds(), errMsg)
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
		errMsg := categorizeError(err)
		r.Stats.IncrementFailure()
		r.Stats.AddError(errMsg)
		r.Stats.AddStatusCode(0) // Track as 'other' for non-HTTP failure
		r.updateRequestStats(reqConfig, 0, time.Since(requestStart).Microseconds(), errMsg)
		return
	}

	// Add headers
	r.addHeaders(req, reqConfig, body)

	// Verbose logging
	if r.VerboseMode {
		fmt.Printf("[verbose] %s %s\n", reqConfig.Method, url)
	}

	// Send request
	resp, err := r.client.Do(req)
	if err != nil {
		errMsg := categorizeError(err)
		r.Stats.IncrementFailure()
		r.Stats.AddStatusCode(0) // Track as 'other' for connection/timeout errors
		r.Stats.AddError(errMsg)
		r.updateRequestStats(reqConfig, 0, time.Since(requestStart).Microseconds(), errMsg)
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
	if err != nil {
		errMsg := categorizeError(err)
		r.Stats.IncrementFailure()
		r.Stats.AddError(errMsg)
		r.updateRequestStats(reqConfig, 0, time.Since(requestStart).Microseconds(), errMsg)
		return
	}

	r.Stats.AddBytes(int64(len(respBody)))

	responseTime := time.Since(requestStart).Microseconds()

	var errMsg string
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		r.Stats.IncrementSuccess()
	} else {
		// Include HTTP status text for better error reporting
		statusText := http.StatusText(resp.StatusCode)
		if statusText != "" {
			errMsg = fmt.Sprintf("HTTP %d %s", resp.StatusCode, statusText)
		} else {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}

		// Try to extract error message from response body
		if len(respBody) > 0 && len(respBody) < 10000 { // Only parse reasonable sized responses
			bodyMsg := extractErrorMessage(respBody, resp.Header.Get("Content-Type"))
			if bodyMsg != "" {
				// Append body message to status text
				errMsg = fmt.Sprintf("%s: %s", errMsg, bodyMsg)
			}
		}

		r.Stats.IncrementFailure()
		r.Stats.AddError(errMsg)
	}

	r.Stats.AddResponseTime(responseTime)

	// Verbose response logging
	if r.VerboseMode {
		url := config.ResolveVariables(reqConfig.URL, r.Config.Variables)
		fmt.Printf("[verbose] %s %s -> %d (%s)\n", reqConfig.Method, url, resp.StatusCode, time.Duration(responseTime)*time.Microsecond)
	}

	// Update per-request stats
	r.updateRequestStats(reqConfig, resp.StatusCode, responseTime, errMsg)
}

// updateRequestStats updates the per-request statistics
func (r *Runner) updateRequestStats(reqConfig *config.RequestConfig, statusCode int, responseTime int64, errMsg string) {
	reqStats := r.Stats.GetOrCreateRequestStats(reqConfig.Name, reqConfig.URL, reqConfig.Method)
	reqStats.Mutex.Lock()
	reqStats.RequestCount++
	reqStats.TotalLatency += responseTime
	if statusCode >= 200 && statusCode < 300 {
		reqStats.SuccessCount++
	} else {
		reqStats.FailureCount++
		// Track error per endpoint
		if errMsg != "" {
			reqStats.Errors[errMsg]++
		}
	}
	reqStats.Mutex.Unlock()
}

