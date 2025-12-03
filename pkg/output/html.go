// Package output handles benchmark result output in various formats
package output

import (
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/benchmarking_go/pkg/benchmark"
	"github.com/benchmarking_go/pkg/config"
)

// HTMLReport represents data for the HTML report template
type HTMLReport struct {
	Title            string
	Timestamp        string
	Duration         string
	TotalRequests    int64
	SuccessCount     int64
	FailureCount     int64
	SuccessRate      float64
	RequestsPerSec   float64
	ReqSecStdDev     float64
	ReqSecMax        float64
	AvgLatency       string
	MinLatency       string
	MaxLatency       string
	StdDevLatency    string
	Percentiles      []PercentileData
	HTTPCodes        HTTPCodeData
	Throughput       float64
	ThroughputBytes  int64
	HistogramBuckets []HistogramBucketData
	PerRequestStats  []PerRequestStatData
	Errors           []ErrorData
	Config           ConfigSummary
}

// PercentileData holds percentile information
type PercentileData struct {
	Percentile int
	Value      string
}

// HTTPCodeData holds HTTP status code counts
type HTTPCodeData struct {
	Code1xx int64
	Code2xx int64
	Code3xx int64
	Code4xx int64
	Code5xx int64
	Other   int64
}

// HistogramBucketData holds histogram bucket information
type HistogramBucketData struct {
	Range      string
	Count      int64
	Percentage float64
	BarWidth   int
}

// PerRequestStatData holds per-request statistics
type PerRequestStatData struct {
	Name       string
	URL        string
	Method     string
	Requests   int64
	Success    int64
	Failed     int64
	AvgLatency string
	Errors     []ErrorData // Per-endpoint errors
}

// ErrorData holds error information
type ErrorData struct {
	Message string
	Count   int
}

// ConfigSummary holds configuration summary
type ConfigSummary struct {
	URLs            int
	ConcurrentUsers int
	Duration        string
	RateLimit       int
	HTTP2           bool
	KeepAlive       bool
}

// WriteHTML generates an HTML report from benchmark statistics
func WriteHTML(stats *benchmark.Stats, cfg *config.Config) error {
	report := buildHTMLReport(stats, cfg)

	// Determine output destination
	outputFile := cfg.Output.File
	if outputFile == "" {
		outputFile = "benchmark-report.html"
	}

	// Create file
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating HTML file: %w", err)
	}
	defer f.Close()

	// Parse and execute template
	tmpl, err := template.New("report").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("error parsing HTML template: %w", err)
	}

	if err := tmpl.Execute(f, report); err != nil {
		return fmt.Errorf("error executing HTML template: %w", err)
	}

	fmt.Printf("HTML report saved to: %s\n", outputFile)
	return nil
}

func buildHTMLReport(stats *benchmark.Stats, cfg *config.Config) HTMLReport {
	// Build percentiles
	percentiles := cfg.Settings.Percentiles
	if len(percentiles) == 0 {
		percentiles = []int{50, 75, 90, 99}
	}

	percData := make([]PercentileData, len(percentiles))
	for i, p := range percentiles {
		percData[i] = PercentileData{
			Percentile: p,
			Value:      FormatLatency(float64(stats.GetLatencyPercentile(p))),
		}
	}

	// Build histogram buckets
	buckets := stats.GetHistogramBuckets()
	histData := make([]HistogramBucketData, len(buckets))
	maxPct := float64(0)
	for _, b := range buckets {
		if b.Percentage > maxPct {
			maxPct = b.Percentage
		}
	}
	if maxPct == 0 {
		maxPct = 1
	}

	for i, b := range buckets {
		var rangeStr string
		if b.RangeEnd == -1 {
			rangeStr = fmt.Sprintf("%s+", benchmark.FormatDuration(b.RangeStart))
		} else {
			rangeStr = fmt.Sprintf("%s - %s", benchmark.FormatDuration(b.RangeStart), benchmark.FormatDuration(b.RangeEnd))
		}
		histData[i] = HistogramBucketData{
			Range:      rangeStr,
			Count:      b.Count,
			Percentage: b.Percentage,
			BarWidth:   int(b.Percentage / maxPct * 100),
		}
	}

	// Build per-request stats
	stats.Lock()
	perReqData := make([]PerRequestStatData, 0, len(stats.RequestStats))
	for _, rs := range stats.RequestStats {
		avgLatency := float64(0)
		if rs.RequestCount > 0 {
			avgLatency = float64(rs.TotalLatency) / float64(rs.RequestCount)
		}
		// Build per-endpoint errors
		endpointErrors := make([]ErrorData, 0, len(rs.Errors))
		for msg, count := range rs.Errors {
			endpointErrors = append(endpointErrors, ErrorData{Message: msg, Count: count})
		}
		perReqData = append(perReqData, PerRequestStatData{
			Name:       rs.Name,
			URL:        rs.URL,
			Method:     rs.Method,
			Requests:   rs.RequestCount,
			Success:    rs.SuccessCount,
			Failed:     rs.FailureCount,
			AvgLatency: FormatLatency(avgLatency),
			Errors:     endpointErrors,
		})
	}
	stats.Unlock()

	// Build errors
	errors := stats.GetErrors()
	errData := make([]ErrorData, 0, len(errors))
	for msg, count := range errors {
		errData = append(errData, ErrorData{Message: msg, Count: count})
	}

	// Calculate success rate based on processed requests (success + failure)
	successRate := float64(0)
	totalProcessed := stats.SuccessCount + stats.FailureCount
	if totalProcessed > 0 {
		successRate = float64(stats.SuccessCount) / float64(totalProcessed) * 100
	}

	// Duration string
	durationStr := fmt.Sprintf("%.2fs", stats.TotalDuration)

	return HTMLReport{
		Title:           cfg.Name,
		Timestamp:       time.Now().Format(time.RFC3339),
		Duration:        durationStr,
		TotalRequests:   stats.TotalRequests,
		SuccessCount:    stats.SuccessCount,
		FailureCount:    stats.FailureCount,
		SuccessRate:     successRate,
		RequestsPerSec:  stats.RequestsPerSecond,
		ReqSecStdDev:    stats.RequestRateStdDev(),
		ReqSecMax:       stats.MaxRequestRate(),
		AvgLatency:      FormatLatency(stats.AverageResponseTime()),
		MinLatency:      FormatLatency(float64(stats.MinResponseTime())),
		MaxLatency:      FormatLatency(float64(stats.MaxResponseTime())),
		StdDevLatency:   FormatLatency(stats.StandardDeviation()),
		Percentiles:     percData,
		HTTPCodes: HTTPCodeData{
			Code1xx: stats.Http1xxCount,
			Code2xx: stats.Http2xxCount,
			Code3xx: stats.Http3xxCount,
			Code4xx: stats.Http4xxCount,
			Code5xx: stats.Http5xxCount,
			Other:   stats.OtherCount,
		},
		Throughput:       stats.ThroughputMBps(),
		ThroughputBytes:  stats.TotalBytes,
		HistogramBuckets: histData,
		PerRequestStats:  perReqData,
		Errors:           errData,
		Config: ConfigSummary{
			URLs:            len(cfg.Requests),
			ConcurrentUsers: cfg.Settings.ConcurrentUsers,
			Duration:        cfg.Settings.Duration,
			RateLimit:       cfg.Settings.RateLimit,
			HTTP2:           cfg.Settings.HTTP2,
			KeepAlive:       !cfg.IsKeepAliveDisabled(),
		},
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Title}}{{.Title}} - {{end}}Benchmark Report</title>
    <style>
        :root {
            --bg-primary: #0d1117;
            --bg-secondary: #161b22;
            --bg-tertiary: #21262d;
            --text-primary: #c9d1d9;
            --text-secondary: #8b949e;
            --accent: #58a6ff;
            --success: #3fb950;
            --warning: #d29922;
            --error: #f85149;
            --border: #30363d;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Noto Sans', Helvetica, Arial, sans-serif;
            background: var(--bg-primary);
            color: var(--text-primary);
            line-height: 1.6;
            padding: 2rem;
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        
        header {
            text-align: center;
            margin-bottom: 2rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border);
        }
        
        h1 {
            font-size: 2rem;
            font-weight: 600;
            margin-bottom: 0.5rem;
        }
        
        .timestamp {
            color: var(--text-secondary);
            font-size: 0.9rem;
        }
        
        .summary-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        
        .summary-card {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1.25rem;
        }
        
        .summary-card h3 {
            font-size: 0.8rem;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 0.05em;
            margin-bottom: 0.5rem;
        }
        
        .summary-card .value {
            font-size: 1.75rem;
            font-weight: 600;
        }
        
        .summary-card .value.success { color: var(--success); }
        .summary-card .value.error { color: var(--error); }
        .summary-card .value.accent { color: var(--accent); }
        
        .summary-card .sub {
            font-size: 0.85rem;
            color: var(--text-secondary);
        }
        
        section {
            background: var(--bg-secondary);
            border: 1px solid var(--border);
            border-radius: 8px;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
        }
        
        section h2 {
            font-size: 1.1rem;
            margin-bottom: 1rem;
            padding-bottom: 0.5rem;
            border-bottom: 1px solid var(--border);
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
        }
        
        th, td {
            padding: 0.75rem;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }
        
        th {
            color: var(--text-secondary);
            font-weight: 500;
            font-size: 0.85rem;
            text-transform: uppercase;
        }
        
        .histogram-bar {
            background: var(--bg-tertiary);
            border-radius: 4px;
            overflow: hidden;
            height: 24px;
        }
        
        .histogram-fill {
            background: linear-gradient(90deg, var(--accent), #79c0ff);
            height: 100%;
            border-radius: 4px;
            transition: width 0.3s ease;
        }
        
        .http-codes {
            display: flex;
            gap: 1rem;
            flex-wrap: wrap;
        }
        
        .http-code {
            padding: 0.5rem 1rem;
            background: var(--bg-tertiary);
            border-radius: 6px;
            font-family: monospace;
        }
        
        .http-code.success { border-left: 3px solid var(--success); }
        .http-code.redirect { border-left: 3px solid var(--warning); }
        .http-code.error { border-left: 3px solid var(--error); }
        
        .error-list {
            font-family: monospace;
            font-size: 0.85rem;
        }
        
        .error-item {
            padding: 0.75rem;
            background: var(--bg-tertiary);
            border-radius: 4px;
            margin-bottom: 0.5rem;
            border-left: 3px solid var(--error);
        }
        
        .endpoint-errors {
            display: flex;
            flex-wrap: wrap;
            gap: 0.25rem;
        }
        
        .error-badge {
            background: rgba(239, 68, 68, 0.2);
            color: var(--error);
            padding: 0.15rem 0.4rem;
            border-radius: 3px;
            font-size: 0.75rem;
            font-family: monospace;
            white-space: nowrap;
        }
        
        td.error {
            color: var(--error);
            font-weight: 600;
        }
        
        .config-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 1rem;
        }
        
        .config-item {
            font-size: 0.9rem;
        }
        
        .config-item label {
            color: var(--text-secondary);
            display: block;
            margin-bottom: 0.25rem;
        }
        
        footer {
            text-align: center;
            padding-top: 1rem;
            color: var(--text-secondary);
            font-size: 0.85rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>{{if .Title}}{{.Title}}{{else}}Benchmark Report{{end}}</h1>
            <p class="timestamp">Generated: {{.Timestamp}}</p>
        </header>
        
        <div class="summary-grid">
            <div class="summary-card">
                <h3>Total Requests</h3>
                <div class="value accent">{{.TotalRequests}}</div>
                <div class="sub">Duration: {{.Duration}}</div>
            </div>
            <div class="summary-card">
                <h3>Success Rate</h3>
                <div class="value {{if ge .SuccessRate 99.0}}success{{else if ge .SuccessRate 95.0}}warning{{else}}error{{end}}">{{printf "%.1f" .SuccessRate}}%</div>
                <div class="sub">{{.SuccessCount}} success / {{.FailureCount}} failed</div>
            </div>
            <div class="summary-card">
                <h3>Requests/sec</h3>
                <div class="value">{{printf "%.2f" .RequestsPerSec}}</div>
                <div class="sub">Max: {{printf "%.2f" .ReqSecMax}}</div>
            </div>
            <div class="summary-card">
                <h3>Avg Latency</h3>
                <div class="value">{{.AvgLatency}}</div>
                <div class="sub">Min: {{.MinLatency}} / Max: {{.MaxLatency}}</div>
            </div>
        </div>
        
        <section>
            <h2>Latency Percentiles</h2>
            <table>
                <thead>
                    <tr>
                        <th>Percentile</th>
                        <th>Latency</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .Percentiles}}
                    <tr>
                        <td>p{{.Percentile}}</td>
                        <td>{{.Value}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </section>
        
        {{if .HistogramBuckets}}
        <section>
            <h2>Latency Distribution</h2>
            <table>
                <thead>
                    <tr>
                        <th style="width: 150px;">Range</th>
                        <th>Distribution</th>
                        <th style="width: 100px;">Count</th>
                        <th style="width: 80px;">%</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .HistogramBuckets}}
                    <tr>
                        <td>{{.Range}}</td>
                        <td>
                            <div class="histogram-bar">
                                <div class="histogram-fill" style="width: {{.BarWidth}}%"></div>
                            </div>
                        </td>
                        <td>{{.Count}}</td>
                        <td>{{printf "%.1f" .Percentage}}%</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </section>
        {{end}}
        
        <section>
            <h2>HTTP Status Codes</h2>
            <div class="http-codes">
                {{if .HTTPCodes.Code1xx}}<div class="http-code">1xx: {{.HTTPCodes.Code1xx}}</div>{{end}}
                <div class="http-code success">2xx: {{.HTTPCodes.Code2xx}}</div>
                {{if .HTTPCodes.Code3xx}}<div class="http-code redirect">3xx: {{.HTTPCodes.Code3xx}}</div>{{end}}
                {{if .HTTPCodes.Code4xx}}<div class="http-code error">4xx: {{.HTTPCodes.Code4xx}}</div>{{end}}
                {{if .HTTPCodes.Code5xx}}<div class="http-code error">5xx: {{.HTTPCodes.Code5xx}}</div>{{end}}
                {{if .HTTPCodes.Other}}<div class="http-code">Other: {{.HTTPCodes.Other}}</div>{{end}}
            </div>
        </section>
        
        {{if .PerRequestStats}}
        <section>
            <h2>Per-Request Statistics</h2>
            <table>
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Method</th>
                        <th>Requests</th>
                        <th>Success</th>
                        <th>Failed</th>
                        <th>Avg Latency</th>
                        <th>Errors</th>
                    </tr>
                </thead>
                <tbody>
                    {{range .PerRequestStats}}
                    <tr>
                        <td>{{.Name}}</td>
                        <td>{{.Method}}</td>
                        <td>{{.Requests}}</td>
                        <td>{{.Success}}</td>
                        <td class="{{if gt .Failed 0}}error{{end}}">{{.Failed}}</td>
                        <td>{{.AvgLatency}}</td>
                        <td>{{if .Errors}}<div class="endpoint-errors">{{range .Errors}}<span class="error-badge">{{.Message}}: {{.Count}}</span>{{end}}</div>{{else}}-{{end}}</td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </section>
        {{end}}
        
        {{if .Errors}}
        <section>
            <h2>Errors</h2>
            <div class="error-list">
                {{range .Errors}}
                <div class="error-item">
                    <strong>{{.Count}}x</strong> {{.Message}}
                </div>
                {{end}}
            </div>
        </section>
        {{end}}
        
        <section>
            <h2>Configuration</h2>
            <div class="config-grid">
                <div class="config-item">
                    <label>URLs</label>
                    <span>{{.Config.URLs}}</span>
                </div>
                <div class="config-item">
                    <label>Concurrent Users</label>
                    <span>{{.Config.ConcurrentUsers}}</span>
                </div>
                {{if .Config.Duration}}
                <div class="config-item">
                    <label>Duration</label>
                    <span>{{.Config.Duration}}</span>
                </div>
                {{end}}
                {{if .Config.RateLimit}}
                <div class="config-item">
                    <label>Rate Limit</label>
                    <span>{{.Config.RateLimit}} req/s</span>
                </div>
                {{end}}
                <div class="config-item">
                    <label>HTTP/2</label>
                    <span>{{if .Config.HTTP2}}Enabled{{else}}Disabled{{end}}</span>
                </div>
                <div class="config-item">
                    <label>Keep-Alive</label>
                    <span>{{if .Config.KeepAlive}}Enabled{{else}}Disabled{{end}}</span>
                </div>
                <div class="config-item">
                    <label>Throughput</label>
                    <span>{{printf "%.2f" .Throughput}} MB/s</span>
                </div>
            </div>
        </section>
        
        <footer>
            <p>Generated by benchmarking_go</p>
        </footer>
    </div>
</body>
</html>`

