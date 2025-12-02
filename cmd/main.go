// Package main is the entry point for the benchmarking tool
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/benchmarking_go/pkg/benchmark"
	"github.com/benchmarking_go/pkg/config"
	"github.com/benchmarking_go/pkg/output"
)

const version = "2.2.0"

func main() {
	// Parse command line flags
	flags := parseFlags()

	// Handle version and help flags
	if handleSpecialFlags(flags) {
		return
	}

	// Validate flags
	if err := validateFlags(flags); err != nil {
		exitWithError("%v", err)
	}

	// Set default values
	setDefaults(flags)

	// Load or create configuration
	cfg, err := loadConfiguration(flags)
	if err != nil {
		exitWithError("%v", err)
	}

	if cfg == nil {
		displayHelp()
		return
	}

	// Parse duration and timeout
	durationSec, err := cfg.GetDurationSeconds()
	if err != nil {
		exitWithError("%v", err)
	}

	timeoutSec := cfg.GetTimeoutSeconds()
	if flags.Timeout != 30 { // CLI override
		timeoutSec = flags.Timeout
	}

	rampUpSec := cfg.GetRampUpSeconds()
	if flags.RampUpSeconds > 0 { // CLI override
		rampUpSec = flags.RampUpSeconds
	}

	// Resolve variables
	cfg.ResolveRequestVariables()

	// Determine quiet mode from output format
	isQuietOutput := cfg.Output.Format == "json" || cfg.Output.Format == "csv"
	effectiveQuietMode := flags.QuietMode || isQuietOutput

	// Print configuration
	if !effectiveQuietMode {
		printConfiguration(cfg, durationSec, timeoutSec, rampUpSec, flags.VerboseMode)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	setupSignalHandler(cancel, effectiveQuietMode)

	// Create and run benchmark
	runner := benchmark.NewRunner(cfg, durationSec, timeoutSec, rampUpSec, effectiveQuietMode, flags.VerboseMode)
	stats := runner.Run(ctx)

	// Output results
	writeResults(stats, cfg, flags.QuietMode)
}

// setupSignalHandler sets up handling for Ctrl+C
func setupSignalHandler(cancel context.CancelFunc, quietMode bool) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		if !quietMode {
			fmt.Println("\nBenchmark interrupted, shutting down...")
		}
		cancel()
	}()
}

// writeResults writes the benchmark results in the appropriate format
func writeResults(stats *benchmark.Stats, cfg *config.Config, quietMode bool) {
	switch cfg.Output.Format {
	case "json":
		if err := output.WriteJSON(stats, cfg); err != nil {
			exitWithError("%v", err)
		}
	case "csv":
		if err := output.WriteCSV(stats, cfg); err != nil {
			exitWithError("%v", err)
		}
	case "html":
		if err := output.WriteHTML(stats, cfg); err != nil {
			exitWithError("%v", err)
		}
	default:
		if quietMode {
			output.WriteConsoleQuiet(stats)
		} else {
			output.WriteConsole(stats, cfg)
		}
	}
}
