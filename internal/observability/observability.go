package observability

import (
	"log/slog"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rlm_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rlm_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// RLM Metrics
	RlmIterations = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rlm_iterations_count",
			Help:    "Number of iterations per RLM completion",
			Buckets: []float64{1, 2, 5, 10, 20, 50},
		},
	)

	RlmDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "rlm_completion_duration_seconds",
			Help:    "Total duration of RLM completion in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s, 2s, 4s, ..., 512s
		},
	)

	TokenUsage = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rlm_token_usage_total",
			Help: "Total number of tokens used",
		},
		[]string{"model", "type"}, // type: input, output
	)

	RlmErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rlm_errors_total",
			Help: "Total number of RLM errors",
		},
	)
)

func SetupLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
				a.Value = slog.StringValue(time.Now().Format(time.RFC3339))
			}
			return a
		},
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}
