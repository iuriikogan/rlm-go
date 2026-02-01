#  RLM Implementation

#### Based on https://alexzhang13.github.io/rlm/

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

####  The service will autonomously execute Python code to solve the task, recursively calling Gemini if needed to analyze results, and return the final answer.
