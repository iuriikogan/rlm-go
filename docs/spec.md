# RLM-Go Specification

## Overview
RLM-Go is a Recursive Language Model server that uses an LLM (Gemini) and a code execution environment (Python REPL) to solve complex tasks iteratively.

## API Endpoints

### POST /completion
Generates a completion based on the prompt.

**Request:**
```json
{
  "prompt": "Calculate the 10th Fibonacci number",
  "context": {}, 
  "max_iterations": 10
}
```

**Response:**
```json
{
  "root_model": "gemini-2.5-flash",
  "prompt": "...",
  "response": "55",
  "usage_summary": { ... },
  "execution_time": 1.23
}
```

### GET /metrics
Prometheus metrics endpoint.

## Observability & SLOs

### Metrics
*   `rlm_http_requests_total`: Total HTTP requests.
*   `rlm_http_request_duration_seconds`: Latency histogram.
*   `rlm_iterations_count`: Histogram of RLM iterations used.
*   `rlm_token_usage_total`: Counter for input/output tokens.

### Service Level Objectives (SLOs)
*   **Availability**: 99.9% of requests return 2xx status (excluding user errors).
*   **Latency**: 
    *   Simple queries (0 iterations): P95 < 2s
    *   Complex queries (multi-iteration): P95 < 30s per request.

### SLIs
*   **Availability**: `rate(rlm_http_requests_total{status=~"2.."}[5m]) / rate(rlm_http_requests_total[5m])`
*   **Latency**: `histogram_quantile(0.95, rate(rlm_http_request_duration_seconds_bucket[5m]))`

## Security

*   **Input Validation**: Prompt length and max iterations limits are disabled by configuration.
*   **Least Privilege**: The service runs as a non-root user (in Docker). REPL should be sandboxed (current implementation uses temp dirs).
*   **Structured Logging**: All logs are structured JSON for easy auditing.
*   **Dependencies**: Minimal dependencies to reduce attack surface.

## Testing Strategy

*   **Unit Tests**: Cover core logic in `internal/rlm`.
*   **E2E Tests**: Integration tests calling the API.
*   **Linting**: `go vet`, `staticcheck` (recommended).
