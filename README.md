<<<<<<< HEAD
#  RLM Implementation

#### Original Paper https://arxiv.org/abs/2512.24601, and repo https://github.com/alexzhang13/rlm

###  Implementation Summary

   * Go Orchestrator: Replaces the Python core, managing the iterative RLM loop, LLM interactions, and REPL orchestration.
   * Persistent Python REPL: A Go-managed Python process that maintains state between code blocks within a single request. It includes an embedded HTTP server to handle recursive llm_query and llm_query_batched calls from the Python environment.
   * Gemini Integration: Uses the official google.golang.org/genai SDK for high-performance interaction with Gemini models.
   * Cloud Run Optimized: Structured as a stateless HTTP service that adheres to Cloud Run's ephemeral execution model.

###  How to Deploy

   1. Set Environment Variables:
       * GEMINI_API_KEY: Your Google AI Studio or Vertex AI API key.
       * PORT: Port for the service (defaults to 8080).

   2. Build and Push to Artifact Registry:

```bash
cd rlm-go
gcloud builds submit --tag gcr.io/[PROJECT_ID]/rlm-go
```
   3. Deploy to Cloud Run:

```bash
gcloud run deploy rlm-go \
--image gcr.io/[PROJECT_ID]/rlm-go \
--platform managed \
--set-env-vars GEMINI_API_KEY=your_key_here \
--allow-unauthenticated
```

###  How to Use

  Send a POST request to the /completion endpoint:

```curl
curl -X POST https://[YOUR_CLOUD_RUN_URL]/completion \
-H "Content-Type: application/json" \
-d '{
"prompt": "Calculate the first 50 prime numbers and check if their sum is prime.",
"context": {"source": "arithmetic_task"}
}'
```

### The service will autonomously execute Python code to solve the task, recursively calling Gemini if needed to analyze results, and return the final answer.
=======
# RLM-Go: Recursive Language Model Implementation

RLM-Go is a high-performance implementation of the **Recursive Language Model (RLM)** paradigm in Go. It enables LLMs to process arbitrarily long contexts and perform complex, multi-step reasoning by interacting with a persistent Python REPL environment.

This implementation is based on the research paper: [**Recursive Language Models**](https://arxiv.org/abs/2512.24601).

## Key Features

-   **Symbolic Handle Architecture**: Context is kept in an external REPL environment. The LLM only receives metadata and a "symbolic handle" (the `context` variable) to manipulate it, effectively bypassing context window limits.
-   **Recursive Inference**: The LLM can programmatically invoke sub-RLM calls from within Python code (via `llm_query`), enabling $\Omega(N)$ or $\Omega(N^2)$ semantic work.
-   **Security by Design**: 
    -   Non-root execution in Docker.
    -   Stateless architecture (REPL state is per-request).
    -   Input validation and JSON-only error responses.
-   **Observability**: 
    -   Structured JSON logging with `slog`.
    -   Prometheus metrics for latency, iteration counts, token usage, and errors.
    -   Ready-to-use Grafana dashboard (`dashboard.json`).
-   **Idiomatic Go**: Refactored for clean separation of concerns, unit testing, and E2E verification.

## Architecture

1.  **Orchestrator**: Manages the iterative loop between the LLM and the REPL.
2.  **REPL**: A long-lived Python process (per request) that executes code and maintains state.
3.  **LM Handler**: An internal HTTP bridge that allows the Python environment to call back to the LLM.
4.  **Client**: Integration with Gemini via the Google Generative AI SDK.

## Getting Started

### Prerequisites

-   Go 1.24+
-   Python 3.x
-   Gemini API Key

### Configuration

Set the following environment variables:
-   `GEMINI_API_KEY`: Your API key.
-   `GEMINI_MODEL_NAME`: (Optional) Default is `gemini-2.5-flash`.
-   `PORT`: (Optional) Default is `8080`.

### Running Locally

```bash
go run cmd/server/main.go
```

### Running with Docker

```bash
docker build -t rlm-go .
docker run -p 8080:8080 -e GEMINI_API_KEY=your_key rlm-go
```

## API Usage

### POST `/completion`

Generate a recursive completion.

```bash
curl -X POST http://localhost:8080/completion \
-H "Content-Type: application/json" \
-d '{
  "prompt": "Summarize the key events in this data and provide a final answer.",
  "context": "Very long data string or object..."
}'
```

**Parameters:**
-   `prompt`: (String) Your query or instructions.
-   `context`: (Any) The data to be injected into the REPL's `context` variable.
-   `max_iterations`: (Integer, optional) Maximum number of RLM steps.

### GET `/metrics`

Exposes Prometheus metrics.

```bash
curl http://localhost:8080/metrics
```

## Monitoring

A Grafana dashboard is available in `dashboard.json`. It tracks:
-   **HTTP Request Rate**: Success vs. failure rates.
-   **P95 Latency**: Completion duration.
-   **RLM Iterations**: Distribution of steps taken to reach an answer.
-   **Token Usage**: Input/Output token counts per model.

## Documentation

-   [Detailed Specification](./docs/spec.md): SLIs/SLOs, security, and technical details.
-   [API Reference](./docs/api.yaml): OpenAPI 3.0 specification.
-   [RLM Paper Reference](./docs/rlm-paper/main.tex): Research background.

## Testing

Run all tests:
```bash
go test ./...
```

E2E tests require a valid `GEMINI_API_KEY`.
>>>>>>> fa9bdb4 (Refactor RLM-Go: Comprehensive overhaul for security, observability, and paper-based recursive logic)
