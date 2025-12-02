// Package main is the entry point for the benchmarking tool
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/benchmarking_go/pkg/config"
)

// CLIFlags holds all command line flags
type CLIFlags struct {
	URL             string
	ConcurrentUsers int
	RequestsPerUser int
	DurationSeconds int
	HTTPMethod      string
	Headers         config.HeaderSliceFlag
	RequestBody     string
	ContentType     string
	ShowHelp        bool
	ShowVersion     bool
	Timeout         int
	ConfigFile      string
	OutputFormat    string
	OutputFile      string
	Insecure        bool

	// Phase 2 features
	RateLimit        int
	RampUpSeconds    int
	QuietMode        bool
	VerboseMode      bool
	DisableKeepAlive bool
	Percentiles      config.IntSliceFlag
}

// parseFlags parses command line arguments and returns CLIFlags
func parseFlags() *CLIFlags {
	flags := &CLIFlags{}

	// Parse command line arguments
	flag.StringVar(&flags.URL, "url", "", "The URL to benchmark")
	flag.StringVar(&flags.URL, "u", "", "The URL to benchmark (shorthand)")

	flag.IntVar(&flags.ConcurrentUsers, "concurrent-users", 10, "Number of concurrent users")
	flag.IntVar(&flags.ConcurrentUsers, "c", 10, "Number of concurrent users (shorthand)")

	flag.IntVar(&flags.RequestsPerUser, "requests-per-user", 100, "Number of requests per user")
	flag.IntVar(&flags.RequestsPerUser, "r", 100, "Number of requests per user (shorthand)")

	flag.IntVar(&flags.DurationSeconds, "duration", 0, "Duration in seconds for the benchmark")
	flag.IntVar(&flags.DurationSeconds, "d", 0, "Duration in seconds for the benchmark (shorthand)")

	flag.StringVar(&flags.HTTPMethod, "method", "GET", "HTTP method to use")
	flag.StringVar(&flags.HTTPMethod, "m", "GET", "HTTP method to use (shorthand)")

	flag.Var(&flags.Headers, "header", "Custom header to include in the request (format: 'key:value')")
	flag.Var(&flags.Headers, "H", "Custom header to include in the request (shorthand) (format: 'key:value')")

	flag.StringVar(&flags.RequestBody, "body", "", "Request body for POST/PUT")
	flag.StringVar(&flags.RequestBody, "b", "", "Request body for POST/PUT (shorthand)")

	flag.StringVar(&flags.ContentType, "content-type", "", "Content-Type of the request body")
	flag.StringVar(&flags.ContentType, "t", "", "Content-Type of the request body (shorthand)")

	flag.IntVar(&flags.Timeout, "timeout", 30, "Timeout in seconds for each request")

	flag.StringVar(&flags.ConfigFile, "config", "", "Path to JSON configuration file")

	flag.StringVar(&flags.OutputFormat, "output", "", "Output format: json, csv, or empty for console")
	flag.StringVar(&flags.OutputFormat, "o", "", "Output format (shorthand)")

	flag.StringVar(&flags.OutputFile, "output-file", "", "Output file path (default: stdout for json/csv)")

	flag.BoolVar(&flags.Insecure, "insecure", false, "Skip TLS certificate verification")
	flag.BoolVar(&flags.Insecure, "k", false, "Skip TLS certificate verification (shorthand)")

	// Phase 2 flags
	flag.IntVar(&flags.RateLimit, "rate", 0, "Rate limit in requests per second (0 = unlimited)")
	flag.IntVar(&flags.RateLimit, "R", 0, "Rate limit (shorthand)")

	flag.IntVar(&flags.RampUpSeconds, "ramp-up", 0, "Ramp-up time in seconds to gradually start workers")

	flag.BoolVar(&flags.QuietMode, "quiet", false, "Quiet mode - only show final summary")
	flag.BoolVar(&flags.QuietMode, "q", false, "Quiet mode (shorthand)")

	flag.BoolVar(&flags.VerboseMode, "verbose", false, "Verbose mode - show detailed request info")
	flag.BoolVar(&flags.VerboseMode, "V", false, "Verbose mode (shorthand)")

	flag.BoolVar(&flags.DisableKeepAlive, "disable-keepalive", false, "Disable HTTP keep-alive connections")

	flag.Var(&flags.Percentiles, "percentiles", "Custom percentiles to report (comma-separated, e.g., '50,90,95,99')")
	flag.Var(&flags.Percentiles, "p", "Custom percentiles (shorthand)")

	flag.BoolVar(&flags.ShowHelp, "help", false, "Display help message")
	flag.BoolVar(&flags.ShowHelp, "h", false, "Display help message (shorthand)")

	flag.BoolVar(&flags.ShowVersion, "version", false, "Display version")
	flag.BoolVar(&flags.ShowVersion, "v", false, "Display version (shorthand)")

	flag.Parse()

	return flags
}

// validateFlags validates the parsed flags and returns any errors
func validateFlags(flags *CLIFlags) error {
	// Verbose and quiet are mutually exclusive
	if flags.VerboseMode && flags.QuietMode {
		return fmt.Errorf("--verbose and --quiet cannot be used together")
	}

	return nil
}

// setDefaults sets default values for flags
func setDefaults(flags *CLIFlags) {
	// Set default percentiles if none specified
	if len(flags.Percentiles) == 0 {
		flags.Percentiles = []int{50, 75, 90, 99}
	}
}

// loadConfiguration loads or creates configuration from flags
func loadConfiguration(flags *CLIFlags) (*config.Config, error) {
	var cfg *config.Config
	var err error

	if flags.ConfigFile != "" {
		cfg, err = config.Load(flags.ConfigFile)
		if err != nil {
			return nil, err
		}
		applyConfigOverrides(cfg, flags)
	} else if flags.URL != "" {
		cfg = config.NewFromCLI(
			flags.URL, flags.HTTPMethod, flags.Headers, flags.RequestBody, flags.ContentType,
			flags.ConcurrentUsers, flags.RequestsPerUser, flags.DurationSeconds, flags.Insecure,
			flags.OutputFormat, flags.OutputFile, flags.RateLimit, flags.RampUpSeconds,
			flags.DisableKeepAlive, flags.Percentiles,
		)
	} else {
		return nil, nil
	}

	return cfg, nil
}

// applyConfigOverrides applies CLI flag overrides to config loaded from file
func applyConfigOverrides(cfg *config.Config, flags *CLIFlags) {
	if flags.ConcurrentUsers != 10 {
		cfg.Settings.ConcurrentUsers = flags.ConcurrentUsers
	}
	if flags.RequestsPerUser != 100 {
		cfg.Settings.RequestsPerUser = flags.RequestsPerUser
	}
	if flags.DurationSeconds > 0 {
		cfg.Settings.Duration = fmt.Sprintf("%ds", flags.DurationSeconds)
	}
	if flags.Insecure {
		cfg.Settings.Insecure = true
	}
	if flags.OutputFormat != "" {
		cfg.Output.Format = flags.OutputFormat
	}
	if flags.OutputFile != "" {
		cfg.Output.File = flags.OutputFile
	}
	if flags.RateLimit > 0 {
		cfg.Settings.RateLimit = flags.RateLimit
	}
	if flags.RampUpSeconds > 0 {
		cfg.Settings.RampUp = fmt.Sprintf("%ds", flags.RampUpSeconds)
	}
	if flags.DisableKeepAlive {
		cfg.Settings.DisableKeepAlive = true
	}
	if len(flags.Percentiles) > 0 && !isDefaultPercentiles(flags.Percentiles) {
		cfg.Settings.Percentiles = flags.Percentiles
	}
}

// isDefaultPercentiles checks if the percentiles are the default values
func isDefaultPercentiles(percentiles []int) bool {
	return len(percentiles) == 4 &&
		percentiles[0] == 50 &&
		percentiles[1] == 75 &&
		percentiles[2] == 90 &&
		percentiles[3] == 99
}

// printConfiguration prints the benchmark configuration to console
func printConfiguration(cfg *config.Config, durationSec, timeoutSec, rampUpSec int, verboseMode bool) {
	if cfg.Name != "" {
		fmt.Printf("Benchmark: %s\n", cfg.Name)
	}
	if len(cfg.Requests) == 1 {
		fmt.Printf("URL: %s\n", cfg.Requests[0].URL)
	} else {
		fmt.Printf("URLs: %d endpoints\n", len(cfg.Requests))
		for _, req := range cfg.Requests {
			fmt.Printf("  - %s: %s %s (weight: %d)\n", req.Name, req.Method, req.URL, req.Weight)
		}
	}
	fmt.Printf("Concurrent users: %d\n", cfg.Settings.ConcurrentUsers)
	fmt.Printf("Request timeout: %d seconds\n", timeoutSec)

	if cfg.Settings.Insecure {
		fmt.Println("TLS verification: disabled")
	}
	if cfg.Settings.RateLimit > 0 {
		fmt.Printf("Rate limit: %d req/s\n", cfg.Settings.RateLimit)
	}
	if rampUpSec > 0 {
		fmt.Printf("Ramp-up: %d seconds\n", rampUpSec)
	}
	if cfg.IsKeepAliveDisabled() {
		fmt.Println("Keep-alive: disabled")
	}

	if durationSec > 0 {
		fmt.Printf("Duration: %d seconds\n", durationSec)
	} else {
		fmt.Printf("Requests per user: %d\n", cfg.Settings.RequestsPerUser)
	}

	if verboseMode {
		fmt.Printf("Percentiles: %v\n", cfg.Settings.Percentiles)
	}

	fmt.Println()
}

// handleSpecialFlags handles version and help flags
func handleSpecialFlags(flags *CLIFlags) bool {
	if flags.ShowVersion {
		fmt.Printf("benchmarking_go version %s\n", version)
		return true
	}

	if flags.ShowHelp {
		displayHelp()
		return true
	}

	return false
}

// exitWithError prints an error message and exits
func exitWithError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

