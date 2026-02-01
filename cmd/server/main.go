package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/iuriikogan/rlm-go/internal/client"
	"github.com/iuriikogan/rlm-go/internal/observability"
	"github.com/iuriikogan/rlm-go/internal/rlm"
)

type completionRequest struct {
	Prompt        string      `json:"prompt"`
	Context       interface{} `json:"context,omitempty"`
	MaxIterations int         `json:"max_iterations,omitempty"`
}

func main() {
	logger := observability.SetupLogger()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		logger.Error("GEMINI_API_KEY environment variable is not set")
		os.Exit(1)
	}

	modelName := os.Getenv("GEMINI_MODEL_NAME")

	geminiClient, err := client.NewGeminiClient(apiKey, modelName)
	if err != nil {
		logger.Error("Failed to create Gemini client", "error", err)
		os.Exit(1)
	}

	// Default max iterations, can be overridden per request (within limits)
	defaultMaxIter := 10
	rlmEngine := rlm.NewRLM(geminiClient, defaultMaxIter)

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/completion", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		defer func() {
			duration := time.Since(start).Seconds()
			observability.HttpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, http.StatusText(rw.status)).Inc()
			observability.HttpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
			logger.Info("Request handled", "method", r.Method, "path", r.URL.Path, "status", rw.status, "duration", duration)
		}()

		if r.Method != http.MethodPost {
			respondError(rw, http.StatusMethodNotAllowed, "Method not allowed")
			return
		}

		var req completionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(rw, http.StatusBadRequest, "Invalid JSON body")
			return
		}

		// Input Validation
		if req.Prompt == "" {
			respondError(rw, http.StatusBadRequest, "Prompt is required")
			return
		}
		// Prompt length limit removed per user request

		// Create a new RLM instance if specific max iterations requested, otherwise use default
		engine := rlmEngine
		if req.MaxIterations > 0 {
			// Max iterations hard limit removed per user request
			engine = rlm.NewRLM(geminiClient, req.MaxIterations)
		}

		ctx := r.Context()
		resp, err := engine.Completion(ctx, req.Prompt, req.Context)
		if err != nil {
			logger.Error("RLM Completion failed", "error", err)
			respondError(rw, http.StatusInternalServerError, err.Error())
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(rw).Encode(resp); err != nil {
			logger.Error("Failed to encode response", "error", err)
		}
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Starting server", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server exited properly")
}

func respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Custom ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
