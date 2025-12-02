// Package config handles JSON configuration loading and parsing
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config represents the root JSON configuration
type Config struct {
	Schema         string            `json:"$schema,omitempty"`
	Name           string            `json:"name,omitempty"`
	Description    string            `json:"description,omitempty"`
	Settings       Settings          `json:"settings,omitempty"`
	Variables      map[string]string `json:"variables,omitempty"`
	DefaultHeaders map[string]string `json:"defaultHeaders,omitempty"`
	Requests       []RequestConfig   `json:"requests"`
	Output         OutputConfig      `json:"output,omitempty"`
}

// Settings contains global benchmark settings
type Settings struct {
	ConcurrentUsers  int    `json:"concurrentUsers,omitempty"`
	Duration         string `json:"duration,omitempty"`
	RequestsPerUser  int    `json:"requestsPerUser,omitempty"`
	Timeout          string `json:"timeout,omitempty"`
	Insecure         bool   `json:"insecure,omitempty"`
	KeepAlive        *bool  `json:"keepAlive,omitempty"`        // Pointer to distinguish unset from false
	DisableKeepAlive bool   `json:"disableKeepAlive,omitempty"` // Alternative way to disable
	MaxConnections   int    `json:"maxConnections,omitempty"`
	RateLimit        int    `json:"rateLimit,omitempty"`        // Requests per second limit
	RampUp           string `json:"rampUp,omitempty"`           // Ramp-up duration (e.g., "10s")
	Percentiles      []int  `json:"percentiles,omitempty"`      // Custom percentiles to report
	ShowHistogram    bool   `json:"showHistogram,omitempty"`    // Show ASCII histogram in output
	DisableHdr       bool   `json:"disableHdr,omitempty"`       // Disable HdrHistogram
	HTTP2            bool   `json:"http2,omitempty"`            // Enable HTTP/2
	ShowLiveStats    bool   `json:"showLiveStats,omitempty"`    // Show real-time stats during benchmark
}

// RequestConfig represents a single request definition
type RequestConfig struct {
	Name     string            `json:"name"`
	URL      string            `json:"url"`
	Method   string            `json:"method,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     interface{}       `json:"body,omitempty"`
	BodyFile string            `json:"bodyFile,omitempty"`
	Weight   int               `json:"weight,omitempty"`
}

// OutputConfig defines output settings
type OutputConfig struct {
	Format string `json:"format,omitempty"`
	File   string `json:"file,omitempty"`
}

// Header represents an HTTP header (for CLI flags)
type Header struct {
	Key   string
	Value string
}

// HeaderSliceFlag is a custom flag type for handling multiple headers
type HeaderSliceFlag []Header

func (h *HeaderSliceFlag) String() string {
	return fmt.Sprintf("%v", *h)
}

func (h *HeaderSliceFlag) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("header must be in format 'key:value'")
	}
	*h = append(*h, Header{Key: strings.TrimSpace(parts[0]), Value: strings.TrimSpace(parts[1])})
	return nil
}

// IntSliceFlag is a custom flag type for handling multiple integers (percentiles)
type IntSliceFlag []int

func (i *IntSliceFlag) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *IntSliceFlag) Set(value string) error {
	// Parse comma-separated values
	parts := strings.Split(value, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		val, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("invalid percentile value: %s", p)
		}
		if val < 0 || val > 100 {
			return fmt.Errorf("percentile must be between 0 and 100: %d", val)
		}
		*i = append(*i, val)
	}
	return nil
}

// Load loads configuration from a JSON file
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	config.SetDefaults()

	return &config, nil
}

// SetDefaults sets default values for the configuration
func (c *Config) SetDefaults() {
	if c.Settings.ConcurrentUsers == 0 {
		c.Settings.ConcurrentUsers = 10
	}
	if c.Settings.RequestsPerUser == 0 {
		c.Settings.RequestsPerUser = 100
	}
	if c.Settings.Timeout == "" {
		c.Settings.Timeout = "30s"
	}

	// Set default percentiles if not specified
	if len(c.Settings.Percentiles) == 0 {
		c.Settings.Percentiles = []int{50, 75, 90, 99}
	}

	// Set default weights and methods if not specified
	for i := range c.Requests {
		if c.Requests[i].Weight == 0 {
			c.Requests[i].Weight = 1
		}
		if c.Requests[i].Method == "" {
			c.Requests[i].Method = "GET"
		}
		if c.Requests[i].Name == "" {
			c.Requests[i].Name = fmt.Sprintf("Request %d", i+1)
		}
	}
}

// GetDurationSeconds parses the duration string and returns seconds
func (c *Config) GetDurationSeconds() (int, error) {
	if c.Settings.Duration == "" {
		return 0, nil
	}
	dur, err := time.ParseDuration(c.Settings.Duration)
	if err != nil {
		return 0, fmt.Errorf("invalid duration format: %w", err)
	}
	return int(dur.Seconds()), nil
}

// GetTimeoutSeconds parses the timeout string and returns seconds
func (c *Config) GetTimeoutSeconds() int {
	if c.Settings.Timeout == "" {
		return 30
	}
	dur, err := time.ParseDuration(c.Settings.Timeout)
	if err != nil {
		return 30
	}
	return int(dur.Seconds())
}

// GetRampUpSeconds parses the ramp-up string and returns seconds
func (c *Config) GetRampUpSeconds() int {
	if c.Settings.RampUp == "" {
		return 0
	}
	dur, err := time.ParseDuration(c.Settings.RampUp)
	if err != nil {
		return 0
	}
	return int(dur.Seconds())
}

// IsKeepAliveDisabled returns true if keep-alive should be disabled
func (c *Config) IsKeepAliveDisabled() bool {
	if c.Settings.DisableKeepAlive {
		return true
	}
	if c.Settings.KeepAlive != nil && !*c.Settings.KeepAlive {
		return true
	}
	return false
}

// ResolveVariables replaces variables in a string with their values
func ResolveVariables(input string, variables map[string]string) string {
	result := input
	for key, value := range variables {
		result = strings.ReplaceAll(result, "{{"+key+"}}", value)
	}
	// Handle environment variables
	for strings.Contains(result, "{{env ") {
		start := strings.Index(result, "{{env ")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], "}}")
		if end == -1 {
			break
		}
		envExpr := result[start : start+end+2]
		// Extract env var name (format: {{env "VAR_NAME"}})
		varName := strings.TrimPrefix(envExpr, "{{env ")
		varName = strings.TrimSuffix(varName, "}}")
		varName = strings.Trim(varName, "\"'")
		envValue := os.Getenv(varName)
		result = strings.Replace(result, envExpr, envValue, 1)
	}
	return result
}

// PrepareRequestBody prepares the request body from config
func PrepareRequestBody(reqConfig *RequestConfig) (string, error) {
	if reqConfig.BodyFile != "" {
		data, err := os.ReadFile(reqConfig.BodyFile)
		if err != nil {
			return "", fmt.Errorf("failed to read body file: %w", err)
		}
		return string(data), nil
	}

	if reqConfig.Body != nil {
		switch v := reqConfig.Body.(type) {
		case string:
			return v, nil
		default:
			data, err := json.Marshal(v)
			if err != nil {
				return "", fmt.Errorf("failed to marshal body: %w", err)
			}
			return string(data), nil
		}
	}

	return "", nil
}

// ResolveRequestVariables resolves variables in all request configurations
func (c *Config) ResolveRequestVariables() {
	for i := range c.Requests {
		c.Requests[i].URL = ResolveVariables(c.Requests[i].URL, c.Variables)
	}
}

// NewFromCLI creates a Config from command-line arguments
func NewFromCLI(url, method string, headers HeaderSliceFlag, body, contentType string,
	concurrentUsers, requestsPerUser, durationSeconds int, insecure bool,
	outputFormat, outputFile string, rateLimit, rampUpSeconds int,
	disableKeepAlive bool, percentiles []int, showHistogram, disableHdr bool,
	http2, showLiveStats bool) *Config {

	config := &Config{
		Settings: Settings{
			ConcurrentUsers:  concurrentUsers,
			RequestsPerUser:  requestsPerUser,
			Insecure:         insecure,
			RateLimit:        rateLimit,
			DisableKeepAlive: disableKeepAlive,
			Percentiles:      percentiles,
			ShowHistogram:    showHistogram,
			DisableHdr:       disableHdr,
			HTTP2:            http2,
			ShowLiveStats:    showLiveStats,
		},
		Requests: []RequestConfig{
			{
				Name:   "Request",
				URL:    url,
				Method: method,
			},
		},
		Output: OutputConfig{
			Format: outputFormat,
			File:   outputFile,
		},
	}

	// Add headers from CLI
	if len(headers) > 0 {
		config.Requests[0].Headers = make(map[string]string)
		for _, h := range headers {
			config.Requests[0].Headers[h.Key] = h.Value
		}
	}

	// Add body from CLI
	if body != "" {
		config.Requests[0].Body = body
	}

	// Add content type
	if contentType != "" {
		if config.Requests[0].Headers == nil {
			config.Requests[0].Headers = make(map[string]string)
		}
		config.Requests[0].Headers["Content-Type"] = contentType
	}

	// Set duration
	if durationSeconds > 0 {
		config.Settings.Duration = fmt.Sprintf("%ds", durationSeconds)
	}

	// Set ramp-up
	if rampUpSeconds > 0 {
		config.Settings.RampUp = fmt.Sprintf("%ds", rampUpSeconds)
	}

	// Set default percentiles if empty
	if len(config.Settings.Percentiles) == 0 {
		config.Settings.Percentiles = []int{50, 75, 90, 99}
	}

	return config
}
