// Package benchmark provides benchmarking functionality
package benchmark

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/benchmarking_go/pkg/config"
	"github.com/tidwall/gjson"
)

// Global counter for unique iteration IDs
var iterationCounter int64

// ScenarioResult represents the result of a single scenario execution
type ScenarioResult struct {
	Success       bool
	StepResults   []StepResult
	TotalDuration time.Duration
	Variables     map[string]string // Final state of variables after scenario
}

// StepResult represents the result of a single step
type StepResult struct {
	StepName       string
	Success        bool
	StatusCode     int
	ResponseTime   time.Duration
	Error          string
	ExtractedVars  map[string]string
	ValidationErrs []string
}

// ScenarioExecutor executes scenario sequences
type ScenarioExecutor struct {
	config      *config.Config
	client      *http.Client
	timeoutSec  int
	verboseMode bool
	stats       *Stats
}

// NewScenarioExecutor creates a new scenario executor
func NewScenarioExecutor(cfg *config.Config, client *http.Client, timeoutSec int, verboseMode bool, stats *Stats) *ScenarioExecutor {
	return &ScenarioExecutor{
		config:      cfg,
		client:      client,
		timeoutSec:  timeoutSec,
		verboseMode: verboseMode,
		stats:       stats,
	}
}

// ExecuteScenario runs all steps in the scenario sequence
func (e *ScenarioExecutor) ExecuteScenario(ctx context.Context) *ScenarioResult {
	result := &ScenarioResult{
		Success:     true,
		StepResults: make([]StepResult, 0, len(e.config.Steps)),
		Variables:   copyVariables(e.config.Variables),
	}

	scenarioStart := time.Now()

	for i, step := range e.config.Steps {
		select {
		case <-ctx.Done():
			result.Success = false
			return result
		default:
		}

		// Handle step delay
		if step.Delay != "" {
			if delay, err := time.ParseDuration(step.Delay); err == nil {
				time.Sleep(delay)
			}
		}

		stepResult := e.executeStep(ctx, &step, result.Variables, i)
		result.StepResults = append(result.StepResults, stepResult)

		// Merge extracted variables
		for k, v := range stepResult.ExtractedVars {
			result.Variables[k] = v
		}

		if !stepResult.Success {
			result.Success = false
			// Continue or break based on step criticality
			// For now, we continue to get all stats
		}
	}

	result.TotalDuration = time.Since(scenarioStart)
	return result
}

// executeStep executes a single step and returns the result
func (e *ScenarioExecutor) executeStep(ctx context.Context, step *config.StepConfig, variables map[string]string, stepIndex int) StepResult {
	result := StepResult{
		StepName:      step.Name,
		Success:       true,
		ExtractedVars: make(map[string]string),
	}

	stepStart := time.Now()

	// Resolve URL with variables
	url := resolveVariables(step.URL, variables)

	// Prepare body
	body, err := prepareStepBody(step, variables)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		e.stats.IncrementFailure()
		e.stats.AddError(err.Error())
		return result
	}

	// Create request
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(e.timeoutSec)*time.Second)
	defer cancel()

	var req *http.Request
	if body != "" {
		req, err = http.NewRequestWithContext(reqCtx, step.Method, url, bytes.NewBufferString(body))
	} else {
		req, err = http.NewRequestWithContext(reqCtx, step.Method, url, nil)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		e.stats.IncrementFailure()
		e.stats.AddError(err.Error())
		return result
	}

	// Add headers
	e.addStepHeaders(req, step, variables, body)

	// Verbose logging
	if e.verboseMode {
		fmt.Printf("[scenario] Step %d: %s %s\n", stepIndex+1, step.Method, url)
	}

	// Send request
	resp, err := e.client.Do(req)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.ResponseTime = time.Since(stepStart)
		e.stats.IncrementFailure()
		if !strings.Contains(err.Error(), "context") {
			e.stats.AddError(err.Error())
		}
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(stepStart)

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		e.stats.IncrementFailure()
		return result
	}

	respBodyStr := string(respBody)

	// Record stats
	e.stats.AddStatusCode(resp.StatusCode)
	e.stats.AddBytes(int64(len(respBody)))
	e.stats.AddResponseTime(result.ResponseTime.Microseconds())

	// Validate response
	if step.Validate != nil {
		validationErrs := e.validateResponse(resp, respBodyStr, step.Validate, result.ResponseTime)
		result.ValidationErrs = validationErrs
		if len(validationErrs) > 0 {
			result.Success = false
			for _, verr := range validationErrs {
				e.stats.AddError(fmt.Sprintf("[%s] %s", step.Name, verr))
			}
		}
	}

	// Extract variables from response
	if step.Extract != nil {
		for varName, jsonPath := range step.Extract {
			value := extractValue(respBodyStr, jsonPath, resp.Header)
			if value != "" {
				result.ExtractedVars[varName] = value
				if e.verboseMode {
					fmt.Printf("[scenario] Extracted %s = %s\n", varName, truncateString(value, 50))
				}
			}
		}
	}

	// Update per-request stats
	reqStats := e.stats.GetOrCreateRequestStats(step.Name, step.URL, step.Method)
	reqStats.Mutex.Lock()
	reqStats.RequestCount++
	reqStats.TotalLatency += result.ResponseTime.Microseconds()
	if result.Success && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		reqStats.SuccessCount++
		e.stats.IncrementSuccess()
	} else {
		reqStats.FailureCount++
		if result.Success { // Only increment if not already failed
			e.stats.IncrementFailure()
			result.Success = false
		}
	}
	reqStats.Mutex.Unlock()

	if e.verboseMode {
		status := "✓"
		if !result.Success {
			status = "✗"
		}
		fmt.Printf("[scenario] %s Step %d: %s -> %d (%s)\n", status, stepIndex+1, step.Name, resp.StatusCode, result.ResponseTime)
	}

	return result
}

// addStepHeaders adds headers to the request
func (e *ScenarioExecutor) addStepHeaders(req *http.Request, step *config.StepConfig, variables map[string]string, body string) {
	// Add default headers
	for key, value := range e.config.DefaultHeaders {
		req.Header.Set(key, resolveVariables(value, variables))
	}

	// Add step-specific headers
	for key, value := range step.Headers {
		req.Header.Set(key, resolveVariables(value, variables))
	}

	// Set default content type for body
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set user agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "benchmarking_go/2.2-scenario")
	}
}

// validateResponse validates the response against the validation config
func (e *ScenarioExecutor) validateResponse(resp *http.Response, body string, validate *config.ValidateConfig, responseTime time.Duration) []string {
	var errors []string

	// Validate status code
	if validate.Status != nil {
		if !validateStatusCode(resp.StatusCode, validate.Status) {
			errors = append(errors, fmt.Sprintf("unexpected status code: got %d", resp.StatusCode))
		}
	}

	// Validate status range
	if validate.StatusRange != nil {
		if resp.StatusCode < validate.StatusRange.Min || resp.StatusCode > validate.StatusRange.Max {
			errors = append(errors, fmt.Sprintf("status code %d not in range [%d, %d]",
				resp.StatusCode, validate.StatusRange.Min, validate.StatusRange.Max))
		}
	}

	// Validate body contains
	if validate.BodyContains != "" {
		if !strings.Contains(body, validate.BodyContains) {
			errors = append(errors, fmt.Sprintf("body does not contain: %s", validate.BodyContains))
		}
	}

	// Validate body not contains
	if validate.BodyNotContains != "" {
		if strings.Contains(body, validate.BodyNotContains) {
			errors = append(errors, fmt.Sprintf("body should not contain: %s", validate.BodyNotContains))
		}
	}

	// Validate JSONPath assertions
	if validate.JSONPath != nil {
		for path, expected := range validate.JSONPath {
			actual := gjson.Get(body, strings.TrimPrefix(path, "$."))
			if !matchJSONValue(actual, expected) {
				errors = append(errors, fmt.Sprintf("JSONPath %s: expected %v, got %v", path, expected, actual.Value()))
			}
		}
	}

	// Validate response headers
	if validate.Headers != nil {
		for key, expected := range validate.Headers {
			actual := resp.Header.Get(key)
			if actual != expected {
				errors = append(errors, fmt.Sprintf("header %s: expected %s, got %s", key, expected, actual))
			}
		}
	}

	// Validate response time
	if validate.ResponseTime != "" {
		maxTime, err := time.ParseDuration(validate.ResponseTime)
		if err == nil && responseTime > maxTime {
			errors = append(errors, fmt.Sprintf("response time %s exceeds max %s", responseTime, maxTime))
		}
	}

	return errors
}

// validateStatusCode checks if the status code matches the expected value(s)
func validateStatusCode(actual int, expected interface{}) bool {
	switch v := expected.(type) {
	case int:
		return actual == v
	case float64:
		return actual == int(v)
	case []interface{}:
		for _, e := range v {
			switch ev := e.(type) {
			case int:
				if actual == ev {
					return true
				}
			case float64:
				if actual == int(ev) {
					return true
				}
			}
		}
		return false
	default:
		return true // No validation if type is unknown
	}
}

// matchJSONValue checks if the actual gjson result matches the expected value
func matchJSONValue(actual gjson.Result, expected interface{}) bool {
	if !actual.Exists() {
		return false
	}

	switch v := expected.(type) {
	case bool:
		return actual.Bool() == v
	case int:
		return actual.Int() == int64(v)
	case float64:
		// Handle both int and float comparisons
		if float64(int64(v)) == v {
			return actual.Int() == int64(v)
		}
		return actual.Float() == v
	case string:
		// Check for comparison operators
		if strings.HasPrefix(v, "> ") {
			num, err := strconv.ParseFloat(strings.TrimPrefix(v, "> "), 64)
			if err == nil {
				return actual.Float() > num
			}
		}
		if strings.HasPrefix(v, ">= ") {
			num, err := strconv.ParseFloat(strings.TrimPrefix(v, ">= "), 64)
			if err == nil {
				return actual.Float() >= num
			}
		}
		if strings.HasPrefix(v, "< ") {
			num, err := strconv.ParseFloat(strings.TrimPrefix(v, "< "), 64)
			if err == nil {
				return actual.Float() < num
			}
		}
		if strings.HasPrefix(v, "<= ") {
			num, err := strconv.ParseFloat(strings.TrimPrefix(v, "<= "), 64)
			if err == nil {
				return actual.Float() <= num
			}
		}
		return actual.String() == v
	default:
		return true
	}
}

// extractValue extracts a value from response body or headers
func extractValue(body string, pathOrExpr string, headers http.Header) string {
	// Check if it's a header extraction (header:HeaderName)
	if strings.HasPrefix(pathOrExpr, "header:") {
		headerName := strings.TrimPrefix(pathOrExpr, "header:")
		return headers.Get(headerName)
	}

	// Check if it's a regex extraction (regex:pattern)
	if strings.HasPrefix(pathOrExpr, "regex:") {
		pattern := strings.TrimPrefix(pathOrExpr, "regex:")
		re, err := regexp.Compile(pattern)
		if err != nil {
			return ""
		}
		matches := re.FindStringSubmatch(body)
		if len(matches) > 1 {
			return matches[1] // Return first capture group
		} else if len(matches) > 0 {
			return matches[0]
		}
		return ""
	}

	// Default: JSONPath extraction using gjson
	// Remove $. prefix if present (gjson doesn't use it)
	jsonPath := strings.TrimPrefix(pathOrExpr, "$.")
	result := gjson.Get(body, jsonPath)
	if result.Exists() {
		return result.String()
	}
	return ""
}

// resolveVariables replaces {{varName}} placeholders with values
// Also supports dynamic functions:
//   - {{$uuid}} - generates a random UUID
//   - {{$randomInt}} - generates a random integer (0-999999)
//   - {{$timestamp}} - current Unix timestamp in milliseconds
//   - {{$iteration}} - current iteration number (globally unique)
//   - {{$randomUser}} - generates a unique user ID like "user-abc123"
func resolveVariables(input string, variables map[string]string) string {
	result := input

	// Handle dynamic functions first
	result = resolveDynamicFunctions(result)

	// Then resolve static variables
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	return result
}

// resolveDynamicFunctions replaces dynamic function placeholders with generated values
func resolveDynamicFunctions(input string) string {
	result := input

	// Replace all occurrences of {{$uuid}}
	for strings.Contains(result, "{{$uuid}}") {
		result = strings.Replace(result, "{{$uuid}}", generateUUID(), 1)
	}

	// Replace all occurrences of {{$randomInt}}
	for strings.Contains(result, "{{$randomInt}}") {
		result = strings.Replace(result, "{{$randomInt}}", generateRandomInt(), 1)
	}

	// Replace all occurrences of {{$timestamp}}
	for strings.Contains(result, "{{$timestamp}}") {
		result = strings.Replace(result, "{{$timestamp}}", fmt.Sprintf("%d", time.Now().UnixMilli()), 1)
	}

	// Replace all occurrences of {{$iteration}}
	for strings.Contains(result, "{{$iteration}}") {
		iteration := atomic.AddInt64(&iterationCounter, 1)
		result = strings.Replace(result, "{{$iteration}}", fmt.Sprintf("%d", iteration), 1)
	}

	// Replace all occurrences of {{$randomUser}}
	for strings.Contains(result, "{{$randomUser}}") {
		result = strings.Replace(result, "{{$randomUser}}", generateRandomUser(), 1)
	}

	return result
}

// generateUUID generates a random UUID v4
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to math/rand if crypto/rand fails
		for i := range uuid {
			uuid[i] = byte(mrand.Intn(256))
		}
	}
	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// generateRandomInt generates a random integer between 0 and 999999
func generateRandomInt() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return fmt.Sprintf("%d", mrand.Intn(1000000))
	}
	return n.String()
}

// generateRandomUser generates a unique user ID like "user-abc123def456"
func generateRandomUser() string {
	bytes := make([]byte, 6)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback
		for i := range bytes {
			bytes[i] = byte(mrand.Intn(256))
		}
	}
	return "user-" + hex.EncodeToString(bytes)
}

// prepareStepBody prepares the request body with variable substitution
func prepareStepBody(step *config.StepConfig, variables map[string]string) (string, error) {
	if step.BodyFile != "" {
		// For now, just read the file - file handling is done in config package
		return "", nil
	}

	if step.Body != nil {
		var bodyStr string
		switch v := step.Body.(type) {
		case string:
			bodyStr = v
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal body: %w", err)
			}
			bodyStr = string(data)
		}
		// Resolve variables in body
		return resolveVariables(bodyStr, variables), nil
	}

	return "", nil
}

// copyVariables creates a copy of the variables map
func copyVariables(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
